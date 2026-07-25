package contract

import "context"

// Settings is the activity-owned subset of the legacy feature settings.
type Settings struct {
	Checkin     CheckinSettings
	Blindbox    BlindboxSettings
	RedPacket   RedPacketSettings
	Leaderboard LeaderboardSettings
	CodeFormat  CodeFormatSettings
}

type CheckinSettings struct {
	Enabled           bool
	MinimumReward     float64
	MaximumReward     float64
	LuckEnabled       bool
	MinimumMultiplier float64
	MaximumMultiplier float64
}

type BlindboxSettings struct {
	Enabled     bool
	TriggerType string
	Interval    int
}

type RedPacketSettings struct {
	Enabled      bool
	MaximumCount int
	ExpireHours  int
}

type LeaderboardSettings struct {
	Enabled            bool
	BalanceEnabled     bool
	ConsumptionEnabled bool
	CheckinEnabled     bool
	TransferEnabled    bool
	IncludeAdmin       bool
}

// CodeFormatSettings carries only the activity code formats.
type CodeFormatSettings struct {
	Invitation CodeFormat
	RedPacket  CodeFormat
}

// CodeFormat is an opaque validated generation configuration.
type CodeFormat struct {
	Prefix    string
	Charset   string
	Separator string
	GroupSize int
	Groups    int
}

// SettingsReader reads the effective activity settings for the current request.
type SettingsReader interface {
	GetActivitySettings(ctx context.Context) (Settings, error)
}
