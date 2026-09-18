package smartrouting

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func accountHealthFixture(t *testing.T) (*Module, *miniredis.Miniredis, service.SmartRoutingAttempt) {
	t.Helper()
	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: r.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	return &Module{redis: client}, r, service.SmartRoutingAttempt{GroupID: 2, AccountID: 11, Model: "gpt-5", Endpoint: "responses", Stream: true, StatusCode: 503, CreatedAt: time.Now()}
}

func expireAccountCooldown(t *testing.T, r *miniredis.Miniredis, a service.SmartRoutingAttempt) {
	t.Helper()
	key := accountHealthKey(a.AccountID, a.Model, a.Endpoint, a.Stream)
	raw, err := r.Get(key)
	require.NoError(t, err)
	var h accountCircuit
	require.NoError(t, json.Unmarshal([]byte(raw), &h))
	h.Until = time.Now().Add(-time.Second).UnixMilli()
	data, err := json.Marshal(h)
	require.NoError(t, err)
	require.NoError(t, r.Set(key, string(data)))
}

func TestAccountHealthDoesNotEjectHealthyPeers(t *testing.T) {
	m, r, a := accountHealthFixture(t)
	group := &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive}
	m.groups = &routingGroupRepoStub{groups: map[int64]*service.Group{2: group}}
	m.accounts = &routingAccountRepoStub{groups: map[int64][]service.Account{2: {
		{ID: 11, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true},
		{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true},
	}}}
	for i := 0; i < 3; i++ {
		m.ObserveRoutingAttempt(a)
	}
	state := &service.SmartRoutingRequestState{SmartRoutingActive: true, EndpointType: "responses", Stream: true, RequestedModel: a.Model}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	n, known, _ := m.inspectModelAccounts(ctx, 2, service.PlatformOpenAI, a.Model, nil)
	require.True(t, known)
	require.Equal(t, 1, n)
	require.Contains(t, state.FailedAccountIDs, int64(11))
	require.NotContains(t, state.FailedAccountIDs, int64(12))
	// Old whole-group state must not affect the new account namespace.
	require.NoError(t, r.Set("custom:route:health:legacy:recovering", "1"))
	for i := 0; i < 30; i++ {
		token, ok := m.AcquireRoutingAccount(ctx, 12, a.Model, a.Endpoint, a.Stream)
		require.True(t, ok)
		require.Empty(t, token)
	}
	_, ok := m.AcquireRoutingAccount(ctx, 11, a.Model, a.Endpoint, a.Stream)
	require.False(t, ok)
}

func TestAccountRecoverySharedLeaseKeepsHealthyTrafficFlowing(t *testing.T) {
	m, r, a := accountHealthFixture(t)
	m.ObserveRoutingAttempt(a)
	expireAccountCooldown(t, r, a)
	other := &Module{redis: m.redis}
	var admitted atomic.Int32
	var tokens sync.Map
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, ok := other.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
			if ok {
				admitted.Add(1)
				tokens.Store(token, true)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, admitted.Load())
	token := ""
	tokens.Range(func(k, _ any) bool {
		var ok bool
		token, ok = k.(string)
		require.True(t, ok)
		return false
	})
	require.NotEmpty(t, token)
	for i := 0; i < 40; i++ {
		_, ok := m.AcquireRoutingAccount(context.Background(), 12, a.Model, a.Endpoint, a.Stream)
		require.True(t, ok)
	}
	// Cancellation releases immediately, without clearing failure history.
	other.ReleaseRoutingAccount(11, a.Model, a.Endpoint, a.Stream, token)
	next, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	require.NotEqual(t, token, next)
	m.ReleaseRoutingAccount(11, a.Model, a.Endpoint, a.Stream, token)
	_, ok = m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.False(t, ok, "old cancellation cannot release a new probe")
	a.Success = true
	a.ProbeToken = token
	m.ObserveRoutingAttempt(a)
	_, ok = m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.False(t, ok, "old completion cannot close a newer circuit")
	a.ProbeToken = next
	m.ObserveRoutingAttempt(a)
	second, ok := other.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	require.NotEmpty(t, second, "one success is still probation")
	a.ProbeToken = second
	other.ObserveRoutingAttempt(a)
	for i := 0; i < 10; i++ {
		lease, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
		require.True(t, ok)
		require.Empty(t, lease)
	}
}

func TestAccountRecoveryFailureBackoffAndLeaseExpiry(t *testing.T) {
	m, r, a := accountHealthFixture(t)
	a.StatusCode = 402
	m.ObserveRoutingAttempt(a)
	key := accountHealthKey(a.AccountID, a.Model, a.Endpoint, a.Stream)
	raw, err := r.Get(key)
	require.NoError(t, err)
	var first accountCircuit
	require.NoError(t, json.Unmarshal([]byte(raw), &first))
	require.GreaterOrEqual(t, first.Until-first.Updated, int64((5 * time.Minute).Milliseconds()))
	expireAccountCooldown(t, r, a)
	lease, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	r.FastForward(accountProbeTTL + time.Second)
	next, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	require.NotEqual(t, lease, next)
	a.ProbeToken = next
	m.ObserveRoutingAttempt(a)
	raw, err = r.Get(key)
	require.NoError(t, err)
	var second accountCircuit
	require.NoError(t, json.Unmarshal([]byte(raw), &second))
	require.Equal(t, 2, second.Failures)
	require.EqualValues(t, (10 * time.Minute).Milliseconds(), second.Until-second.Updated)
	a.Success = true
	a.ProbeToken = lease
	m.ObserveRoutingAttempt(a)
	_, ok = m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.False(t, ok)
	_, ok = m.AcquireRoutingAccount(context.Background(), 11, "other-model", a.Endpoint, a.Stream)
	require.True(t, ok, "a model failure must not block unrelated models")
}

func TestAccountHealthLocalFallbackAndLateSuccess(t *testing.T) {
	m := &Module{}
	a := service.SmartRoutingAttempt{AccountID: 11, Model: "gpt-5", Endpoint: "responses", Stream: true, CreatedAt: time.Now()}
	m.ObserveRoutingAttempt(a)
	a.Success = true
	m.ObserveRoutingAttempt(a)
	_, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.False(t, ok, "normal late success is not a recovery probe")
	key := accountHealthKey(11, a.Model, a.Endpoint, a.Stream)
	m.feedback.mu.Lock()
	h := m.feedback.accounts[key]
	h.Until = time.Now().Add(-time.Second).UnixMilli()
	m.feedback.accounts[key] = h
	m.feedback.mu.Unlock()
	for i := 0; i < 2; i++ {
		token, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
		require.True(t, ok)
		require.NotEmpty(t, token)
		a.ProbeToken = token
		m.ObserveRoutingAttempt(a)
	}
	token, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	require.Empty(t, token)
}

func TestAccountHealthRedisOutageStillProtectsPeersAndRecoversLocally(t *testing.T) {
	m, _, a := accountHealthFixture(t)
	require.NoError(t, m.redis.Close())
	m.ObserveRoutingAttempt(a)
	_, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.False(t, ok)
	_, ok = m.AcquireRoutingAccount(context.Background(), 12, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	key := accountHealthKey(11, a.Model, a.Endpoint, a.Stream)
	m.feedback.mu.Lock()
	h := m.feedback.accounts[key]
	h.Until = time.Now().Add(-time.Second).UnixMilli()
	m.feedback.accounts[key] = h
	m.feedback.mu.Unlock()
	token, ok := m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok)
	require.NotEmpty(t, token)
	m.ReleaseRoutingAccount(11, a.Model, a.Endpoint, a.Stream, token)
	_, ok = m.AcquireRoutingAccount(context.Background(), 11, a.Model, a.Endpoint, a.Stream)
	require.True(t, ok, "cancelled local probe must not remain stuck")
}

func TestResolveKeepsHealthyAccountsWhileOtherAccountsRecover(t *testing.T) {
	m, r, a := accountHealthFixture(t)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	m.repo = NewRepository(db)
	main := &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: "standard", RateMultiplier: 1}
	other := &service.Group{ID: 21, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: "standard", RateMultiplier: 2}
	m.groups = &routingGroupRepoStub{groups: map[int64]*service.Group{2: main, 21: other}}
	m.availableGroups = allowRoutingGroups(2, 21)
	m.accounts = &routingAccountRepoStub{groups: map[int64][]service.Account{
		2:  {{ID: 11, Status: service.StatusActive, Schedulable: true}, {ID: 12, Status: service.StatusActive, Schedulable: true}},
		21: {{ID: 21, Status: service.StatusActive, Schedulable: true}, {ID: 22, Status: service.StatusActive, Schedulable: true}},
	}}
	for _, id := range []int64{11, 21} {
		a.AccountID = id
		m.ObserveRoutingAttempt(a)
		expireAccountCooldown(t, r, a)
		_, ok := m.AcquireRoutingAccount(context.Background(), id, a.Model, a.Endpoint, a.Stream)
		require.True(t, ok)
	}
	expectRoutingProfile(mock, 9)
	expectRoutingStats(mock, 2, 50, 50, 100)
	expectRoutingStats(mock, 21, 50, 50, 200)
	state := &service.SmartRoutingRequestState{MainGroupID: 2, CurrentGroupID: 2, EndpointType: a.Endpoint, Stream: true}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	id := int64(2)
	decision, err := m.Resolve(ctx, &service.APIKey{ID: 9, GroupID: &id, Group: main}, a.Model, "", nil)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.False(t, state.NoRoute)
	require.Contains(t, state.FailedAccountIDs, int64(11))
	require.Contains(t, state.FailedAccountIDs, int64(21))
	require.NotContains(t, state.FailedAccountIDs, int64(12))
	require.NotContains(t, state.FailedAccountIDs, int64(22))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRecoveryRedisLostDuringProbeRetainsFailureLocally(t *testing.T) {
	m, r, a := accountHealthFixture(t)
	m.ObserveRoutingAttempt(a)
	expireAccountCooldown(t, r, a)
	token, admitted := m.AcquireRoutingAccount(context.Background(), a.AccountID, a.Model, a.Endpoint, a.Stream)
	require.True(t, admitted)
	require.NotEmpty(t, token)
	require.NoError(t, m.redis.Close())
	a.ProbeToken = token
	m.ObserveRoutingAttempt(a)
	m.ReleaseRoutingAccount(a.AccountID, a.Model, a.Endpoint, a.Stream, token)
	_, admitted = m.AcquireRoutingAccount(context.Background(), a.AccountID, a.Model, a.Endpoint, a.Stream)
	require.False(t, admitted, "a failed remote probe must remain isolated during a Redis outage")
	_, admitted = m.AcquireRoutingAccount(context.Background(), 12, a.Model, a.Endpoint, a.Stream)
	require.True(t, admitted)
}

type lostAccountLeaseReply struct{}

func (lostAccountLeaseReply) DialHook(next redis.DialHook) redis.DialHook { return next }
func (lostAccountLeaseReply) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (lostAccountLeaseReply) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if err := next(ctx, cmd); err != nil {
			return err
		}
		args := cmd.Args()
		if cmd.Name() == "eval" && len(args) > 5 && args[5] == "acquire" {
			return errors.New("simulated lost reply after remote lease acquisition")
		}
		return nil
	}
}

func TestAccountRecoveryLostLeaseReplyStillReleasesRemoteProbe(t *testing.T) {
	m, r, a := accountHealthFixture(t)
	m.ObserveRoutingAttempt(a)
	expireAccountCooldown(t, r, a)
	m.redis.AddHook(lostAccountLeaseReply{})
	token, admitted := m.AcquireRoutingAccount(context.Background(), a.AccountID, a.Model, a.Endpoint, a.Stream)
	require.True(t, admitted)
	key := accountHealthKey(a.AccountID, a.Model, a.Endpoint, a.Stream) + ":probe"
	lease, err := r.Get(key)
	require.NoError(t, err)
	require.NotEmpty(t, token, "lost Redis reply must not lose cleanup ownership")
	require.Equal(t, lease, token)
	m.ReleaseRoutingAccount(a.AccountID, a.Model, a.Endpoint, a.Stream, token)
	require.False(t, r.Exists(key), "cancellation must not strand an ambiguously acquired lease")
}

func TestNoRouteAuditRecordsExcludedCandidatesWithoutFakeSelection(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	m := &Module{repo: NewRepository(db)}
	state := &service.SmartRoutingRequestState{EndpointType: "responses", AccountExclusions: map[int64]string{11: "account cooldown"}}
	ctx := service.WithSmartRoutingState(context.WithValue(context.Background(), ctxkey.RequestID, "audit-no-route"), state)
	mock.ExpectExec("INSERT INTO custom_route_decisions.*NULL").WithArgs("audit-no-route", int64(9), int64(9), "gpt-5", "responses", "speed", sqlmock.AnyArg(), 0).WillReturnResult(sqlmock.NewResult(1, 1))
	m.recordNoRoute(ctx, &service.APIKey{ID: 9}, &Profile{APIKeyID: 9, Strategy: "speed"}, "gpt-5", []Group{{GroupID: 2, Reason: "no schedulable accounts"}})
	require.Eventually(t, func() bool { return mock.ExpectationsWereMet() == nil }, time.Second, time.Millisecond)
}
