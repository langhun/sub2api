package smartrouting

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

const accountProbeTTL = 5 * time.Minute

type accountCircuit struct {
	Failures   int    `json:"failures"`
	Successes  int    `json:"successes"`
	Until      int64  `json:"until"`
	Updated    int64  `json:"updated"`
	Probe      string `json:"-"`
	ProbeUntil int64  `json:"-"`
}

func accountHealthKey(id int64, model, endpoint string, stream bool) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%t", id, model, endpoint, stream)))
	// Ignore old whole-group recovering flags; both keys share a Cluster slot.
	return fmt.Sprintf("custom:route:account:v2:{%x}", digest)
}

// Completion is fenced by the probe token. Late in-flight successes cannot
// clear an open circuit, and expired probes cannot clear a newer lease.
const accountCircuitScript = `
local tm=redis.call('TIME'); local now=tonumber(tm[1])*1000+math.floor(tonumber(tm[2])/1000)
local raw=redis.call('GET',KEYS[1])
local h=raw and cjson.decode(raw) or {failures=0,successes=0,['until']=0,updated=now}
local op=ARGV[1]; local token=ARGV[2]
if op=='acquire' then
  if h['until']>now then return {0,''} end
  if h.failures==0 then return {1,''} end
  if redis.call('SET',KEYS[2],token,'NX','PX',ARGV[3]) then return {1,token} end
  return {0,''}
end
if op=='release' then
  if token~='' and redis.call('GET',KEYS[2])==token then redis.call('DEL',KEYS[2]) end
  return 1
end
if token~='' and redis.call('GET',KEYS[2])~=token then return 0 end
if op=='success' then
  if token=='' then return 0 end
  h.successes=h.successes+1
  if h.successes>=2 then redis.call('DEL',KEYS[1],KEYS[2]); return 1 end
else
  if token=='' and h['until']>now then return 0 end
  h.failures=math.min(h.failures+1,6); h.successes=0
  h['until']=now+math.min(tonumber(ARGV[3])*2^(h.failures-1),900000)
end
h.updated=now
redis.call('SET',KEYS[1],cjson.encode(h),'EX',86400)
redis.call('DEL',KEYS[2])
return 1`

// All local map access holds feedback.mu. It is only a bounded fallback when
// Redis is absent or unavailable, never an independently refreshed group lock.
func (m *Module) localCircuit(key string, now time.Time) accountCircuit {
	h := m.feedback.accounts[key]
	if now.UnixMilli()-h.Updated >= (24 * time.Hour).Milliseconds() {
		delete(m.feedback.accounts, key)
		return accountCircuit{}
	}
	return h
}

func (m *Module) storeLocalCircuit(key string, h accountCircuit) {
	if m.feedback.accounts == nil {
		m.feedback.accounts = map[string]accountCircuit{}
	}
	if len(m.feedback.accounts) >= 4096 {
		for k, state := range m.feedback.accounts {
			if time.Now().UnixMilli()-state.Updated >= (24 * time.Hour).Milliseconds() {
				delete(m.feedback.accounts, k)
			}
		}
	}
	if _, exists := m.feedback.accounts[key]; exists || len(m.feedback.accounts) < 4096 {
		m.feedback.accounts[key] = h
	}
}

func (m *Module) AcquireRoutingAccount(ctx context.Context, id int64, model, endpoint string, stream bool) (string, bool) {
	if ctx.Err() != nil {
		return "", false
	}
	key := accountHealthKey(id, model, endpoint, stream)
	m.feedback.mu.Lock()
	h := m.localCircuit(key, time.Now())
	locallyBlocked := h.Until > time.Now().UnixMilli() || h.ProbeUntil > time.Now().UnixMilli()
	m.feedback.mu.Unlock()
	if locallyBlocked {
		return "", false
	}
	token := uuid.NewString()
	if m.redis != nil {
		readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		result, err := m.redis.Eval(readCtx, accountCircuitScript, []string{key, key + ":probe"}, "acquire", token, accountProbeTTL.Milliseconds()).Slice()
		cancel()
		if err == nil && len(result) == 2 {
			admitted, _ := result[0].(int64)
			lease, _ := result[1].(string)
			return lease, admitted == 1
		}
		slog.Warn("smart_routing_account_admission_unavailable", "account_id", id, "error", err)
	}
	m.feedback.mu.Lock()
	defer m.feedback.mu.Unlock()
	now := time.Now()
	h = m.localCircuit(key, now)
	if h.Until > now.UnixMilli() || h.ProbeUntil > now.UnixMilli() {
		return "", false
	}
	if h.Failures == 0 {
		if m.redis != nil {
			// The command may have acquired a remote lease before its reply was
			// lost. Carry its token through cleanup even without local history.
			return token, true
		}
		return "", true
	}
	// During a Redis outage use one process-local probe, preserving availability.
	// The same token fences a SET NX that may have succeeded before timeout.
	h.Probe, h.ProbeUntil = token, now.Add(accountProbeTTL).UnixMilli()
	m.storeLocalCircuit(key, h)
	return token, true
}

func (m *Module) ReleaseRoutingAccount(id int64, model, endpoint string, stream bool, token string) {
	if token == "" {
		return
	}
	key := accountHealthKey(id, model, endpoint, stream)
	if m.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := m.redis.Eval(ctx, accountCircuitScript, []string{key, key + ":probe"}, "release", token, 0).Err(); err != nil {
			slog.Warn("smart_routing_probe_release_failed", "account_id", id, "error", err)
		}
	}
	m.feedback.mu.Lock()
	defer m.feedback.mu.Unlock()
	h := m.localCircuit(key, time.Now())
	if h.Probe == token {
		h.Probe = ""
		h.ProbeUntil = 0
		m.storeLocalCircuit(key, h)
	}
}

func (m *Module) observeAccountHealth(a service.SmartRoutingAttempt) {
	if a.AccountID <= 0 || (a.Success && a.ProbeToken == "") {
		return
	}
	key := accountHealthKey(a.AccountID, a.Model, a.Endpoint, a.Stream)
	base := 30 * time.Second
	if a.StatusCode == 401 || a.StatusCode == 402 || a.StatusCode == 403 {
		base = 5 * time.Minute
	}
	op := "failure"
	if a.Success {
		op = "success"
	}
	if m.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		result, err := m.redis.Eval(ctx, accountCircuitScript, []string{key, key + ":probe"}, op, a.ProbeToken, base.Milliseconds()).Int()
		cancel()
		if err == nil {
			if result != 0 {
				m.feedback.mu.Lock()
				delete(m.feedback.accounts, key)
				m.feedback.mu.Unlock()
			}
			return
		}
		slog.Warn("smart_routing_account_health_write_failed", "account_id", a.AccountID, "error", err)
	}
	m.feedback.mu.Lock()
	defer m.feedback.mu.Unlock()
	now := time.Now()
	h := m.localCircuit(key, now)
	validLocalProbe := a.ProbeToken != "" && h.Probe == a.ProbeToken && h.ProbeUntil > now.UnixMilli()
	if a.ProbeToken != "" && !validLocalProbe {
		// A Redis-owned probe can finish during an outage, without a local
		// lease. Preserve its failure locally; never accept an unfenced success
		// or overwrite a newer local probe.
		if a.Success || h.Probe != "" {
			return
		}
	}
	if a.Success {
		h.Successes++
		if h.Successes >= 2 {
			delete(m.feedback.accounts, key)
			return
		}
	} else {
		if !validLocalProbe && h.Until > now.UnixMilli() {
			return
		}
		h.Failures = min(h.Failures+1, 6)
		h.Successes = 0
		delay := min(base*time.Duration(1<<(h.Failures-1)), 15*time.Minute)
		h.Until = now.Add(delay).UnixMilli()
	}
	h.Probe = ""
	h.ProbeUntil = 0
	h.Updated = now.UnixMilli()
	m.storeLocalCircuit(key, h)
}

// Scoring only observes health; it must never reserve unused candidate probes.
func (m *Module) unavailableRoutingAccounts(ctx context.Context, accounts []service.Account, model string) map[int64]bool {
	blocked := map[int64]bool{}
	s := service.SmartRoutingStateFromContext(ctx)
	if s == nil || !s.SmartRoutingActive || len(accounts) == 0 {
		return blocked
	}
	keys := make([]string, 0, len(accounts)*2)
	for _, a := range accounts {
		key := accountHealthKey(a.ID, model, s.EndpointType, s.Stream)
		keys = append(keys, key, key+":probe")
	}
	now := time.Now()
	if m.redis != nil {
		readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		values, err := m.redis.MGet(readCtx, keys...).Result()
		cancel()
		if err == nil && len(values) == len(keys) {
			for i, a := range accounts {
				var h accountCircuit
				if raw, ok := values[2*i].(string); ok {
					_ = json.Unmarshal([]byte(raw), &h)
				}
				blocked[a.ID] = h.Until > now.UnixMilli() || values[2*i+1] != nil
			}
		}
	}
	m.feedback.mu.Lock()
	defer m.feedback.mu.Unlock()
	for i, a := range accounts {
		h := m.localCircuit(keys[2*i], now)
		blocked[a.ID] = blocked[a.ID] || h.Until > now.UnixMilli() || h.ProbeUntil > now.UnixMilli()
		if blocked[a.ID] {
			if s.AccountExclusions == nil {
				s.AccountExclusions = map[int64]string{}
			}
			s.AccountExclusions[a.ID] = "account cooldown or recovery probe in flight"
		}
	}
	return blocked
}
