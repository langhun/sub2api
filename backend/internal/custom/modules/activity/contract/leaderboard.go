package contract

import "context"

// LeaderboardKind identifies one public activity leaderboard. It is intentionally
// independent of the core tables that provide the read model.
type LeaderboardKind string

const (
	LeaderboardBalance     LeaderboardKind = "balance"
	LeaderboardConsumption LeaderboardKind = "consumption"
	LeaderboardCheckin     LeaderboardKind = "checkin"
	LeaderboardTransfer    LeaderboardKind = "transfer"
)

// LeaderboardPeriod limits time-based leaderboards.
type LeaderboardPeriod string

const (
	LeaderboardPeriodDaily   LeaderboardPeriod = "daily"
	LeaderboardPeriodWeekly  LeaderboardPeriod = "weekly"
	LeaderboardPeriodMonthly LeaderboardPeriod = "monthly"
)

// LeaderboardQuery is the public, read-only query passed to the core read model.
// IncludeAdmin is determined by activity settings and must not be supplied by an
// HTTP client.
type LeaderboardQuery struct {
	Page         int
	PageSize     int
	Period       LeaderboardPeriod
	IncludeAdmin bool
}

// LeaderboardEntry is deliberately presentation-safe: it contains no user ID,
// account, or transfer details that would couple activity to another module.
type LeaderboardEntry struct {
	Rank       int     `json:"rank"`
	Username   string  `json:"username"`
	Value      float64 `json:"value"`
	ExtraInt   int     `json:"extra_int,omitempty"`
	ExtraInt2  int     `json:"extra_int2,omitempty"`
	ExtraFloat float64 `json:"extra_float,omitempty"`
	ExtraDate  string  `json:"extra_date,omitempty"`
}

// LeaderboardPage is the pageable public result for every leaderboard kind.
type LeaderboardPage struct {
	Entries []LeaderboardEntry `json:"items"`
	Total   int64              `json:"total"`
}

// LeaderboardFeatureSettings owns the flags activity needs to expose a board.
// TransferEnabled is a public capability flag, not a dependency on any wallet
// implementation.
type LeaderboardFeatureSettings struct {
	Enabled              bool
	BalanceEnabled       bool
	ConsumptionEnabled   bool
	CheckinEnabled       bool
	TransferEnabled      bool
	TransferBoardEnabled bool
	IncludeAdmin         bool
}

// LeaderboardSettingsReader supplies effective activity flags for the request.
type LeaderboardSettingsReader interface {
	GetActivityLeaderboardSettings(ctx context.Context) (LeaderboardFeatureSettings, error)
}

// BalanceLeaderboardReader reads the account-balance ranking without exposing
// core account persistence to activity.
type BalanceLeaderboardReader interface {
	ListBalanceLeaderboard(ctx context.Context, query LeaderboardQuery) (LeaderboardPage, error)
}

// ConsumptionLeaderboardReader reads the usage-consumption ranking.
type ConsumptionLeaderboardReader interface {
	ListConsumptionLeaderboard(ctx context.Context, query LeaderboardQuery) (LeaderboardPage, error)
}

// CheckinLeaderboardReader reads the activity check-in ranking.
type CheckinLeaderboardReader interface {
	ListCheckinLeaderboard(ctx context.Context, query LeaderboardQuery) (LeaderboardPage, error)
}

// TransferLeaderboardReader reads completed-transfer ranking data through a
// public read model. It must not be implemented by importing wallet internals.
type TransferLeaderboardReader interface {
	ListTransferLeaderboard(ctx context.Context, query LeaderboardQuery) (LeaderboardPage, error)
}
