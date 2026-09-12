package contract

import "context"

// CheckinResult is the activity-owned response for a completed check-in.
// It intentionally contains no persistence or accounting implementation types.
type CheckinResult struct {
	RewardAmount float64         `json:"reward_amount"`
	StreakDays   int             `json:"streak_days"`
	CheckedAt    string          `json:"checked_at"`
	CheckinType  string          `json:"checkin_type"`
	BetAmount    float64         `json:"bet_amount,omitempty"`
	Multiplier   float64         `json:"multiplier,omitempty"`
	Blindbox     *BlindboxResult `json:"blindbox,omitempty"`
}

// BlindboxResult is the check-in-facing view of a blind-box reward.
type BlindboxResult struct {
	PrizeName        string  `json:"prize_name"`
	Rarity           string  `json:"rarity"`
	RewardType       string  `json:"reward_type"`
	RewardValue      float64 `json:"reward_value"`
	SubscriptionDays int     `json:"subscription_days,omitempty"`
	RewardDetail     string  `json:"reward_detail,omitempty"`
}

// CheckinStatus describes whether the caller can check in and the configured limits.
type CheckinStatus struct {
	Enabled             bool     `json:"enabled"`
	LuckEnabled         bool     `json:"luck_enabled"`
	BlindboxEnabled     bool     `json:"blindbox_enabled"`
	BlindboxTriggerType string   `json:"blindbox_trigger_type,omitempty"`
	BlindboxInterval    int      `json:"blindbox_interval,omitempty"`
	CanCheckin          bool     `json:"can_checkin"`
	UnavailableReason   string   `json:"unavailable_reason,omitempty"`
	StreakDays          int      `json:"streak_days"`
	TodayReward         *float64 `json:"today_reward,omitempty"`
	TodayCheckinType    string   `json:"today_checkin_type,omitempty"`
	TodayMultiplier     *float64 `json:"today_multiplier,omitempty"`
	MinReward           float64  `json:"min_reward"`
	MaxReward           float64  `json:"max_reward"`
	MinMultiplier       float64  `json:"min_multiplier"`
	MaxMultiplier       float64  `json:"max_multiplier"`
	Balance             float64  `json:"balance"`
}

type CheckinCalendarDay struct {
	Date        string  `json:"date"`
	CheckedIn   bool    `json:"checked_in"`
	RewardType  string  `json:"reward_type,omitempty"`
	RewardValue float64 `json:"reward_value,omitempty"`
	StreakDays  int     `json:"streak_days,omitempty"`
}

type CheckinCalendar struct {
	Days []CheckinCalendarDay `json:"days"`
}

// CheckinService is the activity boundary for all user check-in operations.
// Implementations must keep their balance and audit mutations atomic.
type CheckinService interface {
	Checkin(ctx context.Context, userID int64) (*CheckinResult, error)
	LuckCheckin(ctx context.Context, userID int64, betAmount float64, useMaxBalance bool) (*CheckinResult, error)
	GetStatus(ctx context.Context, userID int64) (*CheckinStatus, error)
	GetCalendar(ctx context.Context, userID int64) (*CheckinCalendar, error)
}

// RewardEligibility is the narrow policy port used by check-in mutations to
// protect promotional rewards. Implementations may return a typed forbidden
// error when an account is ineligible, or a typed service-unavailable error
// when the protection backend cannot be consulted. Check-in propagates that
// error before creating a record or changing the balance.
type RewardEligibility interface {
	CheckinRewardAllowed(ctx context.Context, userID int64) error
}

// CheckinAvailability is an optional read-only extension of RewardEligibility.
// It lets an Overlay policy hide unavailable actions before a user attempts a
// check-in mutation. Implementations that do not expose it preserve the
// existing status behavior.
type CheckinAvailability struct {
	Allowed bool
	Reason  string
}

type CheckinAvailabilityProvider interface {
	CheckinAvailability(ctx context.Context, userID int64) (CheckinAvailability, error)
}

// BlindboxRecord is the check-in endpoint's read-only record representation.
type BlindboxRecord struct {
	ID               int64   `json:"id"`
	PrizeName        string  `json:"prize_name"`
	Rarity           string  `json:"rarity"`
	RewardType       string  `json:"reward_type"`
	RewardValue      float64 `json:"reward_value"`
	RewardDetail     string  `json:"reward_detail,omitempty"`
	SubscriptionDays int     `json:"subscription_days,omitempty"`
	StreakDays       int     `json:"streak_days"`
	CreatedAt        string  `json:"created_at"`
}

type BlindboxRecordList struct {
	Items []BlindboxRecord `json:"items"`
	Total int64            `json:"total"`
}

// BlindboxRecordsReader is kept separate from CheckinService because reward
// delivery is an independent activity concern.
type BlindboxRecordsReader interface {
	GetUserRecords(ctx context.Context, userID int64, page, pageSize int) (*BlindboxRecordList, error)
}
