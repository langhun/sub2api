package rewards

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

func TestPrepareCheckinBlindboxFreezesInvitationCodeAndEnqueues(t *testing.T) {
	prizeID := int64(17)
	prizes := &prizeCatalogStub{enabled: []Prize{{
		ID: prizeID, Name: "Invite", Rarity: RarityRare, RewardType: RewardTypeInvitationCode,
		Weight: 10, Enabled: true,
	}}}
	outbox := &outboxStub{enqueueResult: &Delivery{ID: 42}}
	service := NewService(ServiceDependencies{
		Prizes: prizes, Outbox: outbox, Random: randomStub{intN: 0}, Codes: codeGeneratorStub{code: "INV-9"},
	})

	prepared, err := service.PrepareCheckinBlindbox(context.Background(), 5, 8, 3)
	if err != nil {
		t.Fatalf("prepare blind-box delivery: %v", err)
	}
	if prepared == nil || prepared.Delivery == nil || prepared.Delivery.ID != 42 {
		t.Fatalf("prepared delivery = %#v, want queued delivery", prepared)
	}
	if prepared.Result.PrizeName != "Invite" || prepared.Result.RewardType != RewardTypeInvitationCode {
		t.Fatalf("prepared result = %#v", prepared.Result)
	}
	if outbox.input.IdempotencyKey != "checkin_blindbox:8" || outbox.input.SourceID != 8 || outbox.input.UserID != 5 {
		t.Fatalf("enqueue input = %#v", outbox.input)
	}
	if err := outbox.input.Validate(); err != nil {
		t.Fatalf("enqueue input must be valid: %v", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(outbox.input.RewardSnapshot, &snapshot); err != nil {
		t.Fatalf("decode frozen snapshot: %v", err)
	}
	if snapshot.InvitationCode != "INV-9" || snapshot.PrizeID != prizeID || snapshot.StreakDays != 3 {
		t.Fatalf("frozen snapshot = %#v", snapshot)
	}
}

func TestShouldTriggerUsesTotalCheckinCounter(t *testing.T) {
	settings := settingsStub{settings: contract.Settings{
		Checkin:  contract.CheckinSettings{Enabled: true},
		Blindbox: contract.BlindboxSettings{Enabled: true, TriggerType: "total", Interval: 3},
	}}
	counter := &counterStub{count: 6}
	service := NewService(ServiceDependencies{Settings: settings, Checkins: counter})

	triggered, err := service.ShouldTrigger(context.Background(), 9, 1)
	if err != nil {
		t.Fatalf("should trigger: %v", err)
	}
	if !triggered || counter.userID != 9 {
		t.Fatalf("triggered=%t counter=%#v, want total-count trigger", triggered, counter)
	}
}

func TestDeliveryProcessorUsesCorePortsWithDeliveryKey(t *testing.T) {
	snapshot := Snapshot{
		PrizeID: 12, PrizeName: "Balance", Rarity: RarityEpic, RewardType: RewardTypeBalance,
		RewardValue: 6.25, StreakDays: 4,
	}
	payload, err := jsonMarshalSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	balance := &balanceWriterStub{}
	audit := &auditWriterStub{}
	history := &historyWriterStub{}
	processor := NewDeliveryProcessor(ProcessorDependencies{Balance: balance, Audit: audit, History: history})

	detail, err := processor.ProcessDelivery(context.Background(), Delivery{
		ID: 25, SourceType: SourceCheckinBlindbox, SourceID: 33, UserID: 44, PrizeID: int64Pointer(12),
		RewardSnapshot: payload, RewardType: RewardTypeBalance, RewardValue: 6.25,
		RuleVersion: CheckinBlindboxRuleV1, IdempotencyKey: "checkin_blindbox:33",
	})
	if err != nil {
		t.Fatalf("process delivery: %v", err)
	}
	if detail != "" || balance.operation.UserID != 44 || balance.operation.Amount != 6.25 || balance.operation.IdempotencyKey != "checkin_blindbox:33" {
		t.Fatalf("balance operation = %#v, detail=%q", balance.operation, detail)
	}
	if audit.entry.Type != SourceCheckinBlindbox || audit.entry.ReferenceID != "checkin_blindbox:33" || audit.entry.IdempotencyKey != "checkin_blindbox:33" {
		t.Fatalf("audit entry = %#v", audit.entry)
	}
	if history.record.DeliveryID != 25 || history.record.PrizeID != 12 || history.record.StreakDays != 4 {
		t.Fatalf("history record = %#v", history.record)
	}
}

func TestDeliveryProcessorDoesNotWriteZeroBalanceReward(t *testing.T) {
	snapshot := Snapshot{
		PrizeID: 13, PrizeName: "Zero", Rarity: RarityCommon, RewardType: RewardTypeBalance,
		RewardValue: 0, StreakDays: 1,
	}
	payload, err := jsonMarshalSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	audit := &auditWriterStub{}
	history := &historyWriterStub{}
	processor := NewDeliveryProcessor(ProcessorDependencies{Audit: audit, History: history})

	_, err = processor.ProcessDelivery(context.Background(), Delivery{
		ID: 26, SourceType: SourceCheckinBlindbox, SourceID: 34, UserID: 45, PrizeID: int64Pointer(13),
		RewardSnapshot: payload, RewardType: RewardTypeBalance, RewardValue: 0,
		RuleVersion: CheckinBlindboxRuleV1, IdempotencyKey: "checkin_blindbox:34",
	})
	if err != nil {
		t.Fatalf("process zero balance delivery: %v", err)
	}
	if audit.entry.Amount != 0 || history.record.DeliveryID != 26 {
		t.Fatalf("zero reward should still be audited and recorded: audit=%#v history=%#v", audit.entry, history.record)
	}
}

func TestWorkerFailureIsPersistedForRetry(t *testing.T) {
	fixedNow := time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
	outbox := &outboxStub{claimed: []Delivery{{ID: 9, Attempts: 1}}}
	worker := NewWorker(outbox, processorStub{err: errors.New("core unavailable")}, WorkerOptions{RetryDelay: 10 * time.Second})
	worker.now = func() time.Time { return fixedNow }

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once should persist domain failure without stopping worker: %v", err)
	}
	if outbox.markedID != 9 || outbox.markedError != "core unavailable" || outbox.nextRetryAt == nil || !outbox.nextRetryAt.Equal(fixedNow.Add(10*time.Second)) {
		t.Fatalf("failed transition = id:%d error:%q retry:%v", outbox.markedID, outbox.markedError, outbox.nextRetryAt)
	}
}

func TestRuntimeStopsWorkerExactlyOnce(t *testing.T) {
	runner := &blockingRunner{started: make(chan struct{}), stopped: make(chan struct{})}
	runtime := NewRuntime(runner)
	runtime.Start(context.Background())
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	runtime.Stop()
	runtime.Stop()
	select {
	case <-runner.stopped:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}

type prizeCatalogStub struct{ enabled []Prize }

func (s *prizeCatalogStub) ListEnabled(context.Context) ([]Prize, error) { return s.enabled, nil }
func (s *prizeCatalogStub) List(context.Context) ([]Prize, error)        { return s.enabled, nil }
func (s *prizeCatalogStub) Get(_ context.Context, prizeID int64) (*Prize, error) {
	for index := range s.enabled {
		if s.enabled[index].ID == prizeID {
			return &s.enabled[index], nil
		}
	}
	return nil, nil
}
func (s *prizeCatalogStub) Save(_ context.Context, prize Prize) (Prize, error) {
	return prize, nil
}
func (s *prizeCatalogStub) Archive(context.Context, int64) error      { return nil }
func (s *prizeCatalogStub) Stats(context.Context) (PrizeStats, error) { return PrizeStats{}, nil }

type randomStub struct {
	intN  int
	float float64
}

func (s randomStub) IntN(int) (int, error)     { return s.intN, nil }
func (s randomStub) Float64() (float64, error) { return s.float, nil }

type codeGeneratorStub struct{ code string }

func (s codeGeneratorStub) GenerateInvitationCode(context.Context) (string, error) {
	return s.code, nil
}

type settingsStub struct {
	settings contract.Settings
	err      error
}

func (s settingsStub) GetActivitySettings(context.Context) (contract.Settings, error) {
	return s.settings, s.err
}

type counterStub struct {
	count  int
	userID int64
}

func (s *counterStub) CountCheckins(_ context.Context, userID int64) (int, error) {
	s.userID = userID
	return s.count, nil
}

type balanceWriterStub struct{ operation contract.BalanceOperation }

func (s *balanceWriterStub) Credit(_ context.Context, operation contract.BalanceOperation) error {
	s.operation = operation
	return nil
}
func (*balanceWriterStub) DebitIfSufficient(context.Context, contract.BalanceOperation) (bool, error) {
	return false, nil
}

type auditWriterStub struct{ entry contract.AuditEntry }

func (s *auditWriterStub) WriteActivityAudit(_ context.Context, entry contract.AuditEntry) error {
	s.entry = entry
	return nil
}

type historyWriterStub struct{ record BlindboxRecord }

func (s *historyWriterStub) RecordBlindboxDelivery(_ context.Context, record BlindboxRecord) error {
	s.record = record
	return nil
}

type processorStub struct{ err error }

func (s processorStub) ProcessDelivery(context.Context, Delivery) (string, error) { return "", s.err }

type outboxStub struct {
	enqueueResult *Delivery
	input         CreateDelivery
	claimed       []Delivery
	markedID      int64
	markedError   string
	nextRetryAt   *time.Time
}

func (s *outboxStub) Enqueue(_ context.Context, input CreateDelivery) (*Delivery, error) {
	s.input = input
	return s.enqueueResult, nil
}
func (s *outboxStub) ClaimDue(context.Context, time.Time, int) ([]Delivery, error) {
	return s.claimed, nil
}
func (*outboxStub) ClaimByID(context.Context, int64, time.Time) (*Delivery, error) { return nil, nil }
func (s *outboxStub) ExecuteClaimed(ctx context.Context, _ int64, apply DeliveryApply) error {
	if len(s.claimed) == 0 {
		return errors.New("missing claimed delivery")
	}
	_, err := apply(ctx, s.claimed[0])
	return err
}
func (s *outboxStub) MarkFailed(_ context.Context, id int64, lastError string, nextRetryAt *time.Time) error {
	s.markedID, s.markedError, s.nextRetryAt = id, lastError, nextRetryAt
	return nil
}
func (*outboxStub) RecoverStale(context.Context, time.Time, time.Time) (int, error) { return 0, nil }
func (*outboxStub) Get(context.Context, int64) (*Delivery, error)                   { return nil, nil }
func (*outboxStub) List(context.Context, DeliveryFilter) ([]Delivery, int64, error) {
	return nil, 0, nil
}

type blockingRunner struct {
	started chan struct{}
	stopped chan struct{}
	once    sync.Once
}

func (r *blockingRunner) Run(ctx context.Context) {
	r.once.Do(func() { close(r.started) })
	<-ctx.Done()
	close(r.stopped)
}

func int64Pointer(value int64) *int64 { return &value }
