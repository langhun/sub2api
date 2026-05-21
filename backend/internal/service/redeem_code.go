package service

import (
	"time"
)

var registrationInvitationCodeCharset = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
var redeemCodeCharset = []byte("0123456789ABCDEF")

type RedeemCode struct {
	ID        int64
	Code      string
	Type      string
	Value     float64
	Status    string
	UsedBy    *int64
	UsedAt    *time.Time
	Notes     string
	CreatedAt time.Time
	ExpiresAt *time.Time

	GroupID      *int64
	ValidityDays int
	Multiplier   float64
	BetAmount    float64

	User  *User
	Group *Group

	SourceType      string
	SourceSummary   string
	SourceUser      *User
	InviterUser     *User
	WinningUser     *User
	WinningPrize    string
	WinningReward   string
	GeneratedByUser *User
}

func (r *RedeemCode) IsUsed() bool {
	return r.Status == StatusUsed
}

func (r *RedeemCode) IsExpired() bool {
	return r.IsExpiredAt(time.Now())
}

func (r *RedeemCode) IsExpiredAt(now time.Time) bool {
	if r == nil {
		return false
	}
	if r.Status == StatusExpired {
		return true
	}
	return r.Status == StatusUnused && r.ExpiresAt != nil && !r.ExpiresAt.After(now)
}

func (r *RedeemCode) CanUse() bool {
	return r.Status == StatusUnused && !r.IsExpired()
}

func NormalizeRegistrationInvitationCode(code string) string {
	return NormalizeCodeValueWithFormat(code, DefaultRegistrationInvitationCodeFormat())
}

func NormalizeRegistrationInvitationCodeWithSettings(code string, format CodeFormatSettings) string {
	return NormalizeCodeValueWithFormat(code, format)
}

func IsRegistrationInvitationCodeFormat(code string) bool {
	return IsCodeMatchingFormat(code, DefaultRegistrationInvitationCodeFormat())
}

func IsRegistrationInvitationCodeFormatWithSettings(code string, format CodeFormatSettings) bool {
	return IsCodeMatchingFormat(code, format)
}

func GenerateRegistrationInvitationCode() (string, error) {
	return GenerateRegistrationInvitationCodeWithFormat(DefaultRegistrationInvitationCodeFormat())
}

func GenerateRegistrationInvitationCodeWithFormat(format CodeFormatSettings) (string, error) {
	return GenerateCodeWithFormat(format, registrationInvitationCodeCharset)
}

func GenerateRedeemCode() (string, error) {
	return GenerateRedeemCodeWithFormat(DefaultRedeemCodeFormat())
}

func GenerateRedeemCodeWithFormat(format CodeFormatSettings) (string, error) {
	return GenerateCodeWithFormat(format, redeemCodeCharset)
}
