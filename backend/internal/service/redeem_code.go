package service

import (
	"strings"
	"time"
)

var registrationInvitationCodeCharset = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

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

	GroupID      *int64
	ValidityDays int
	Multiplier   float64
	BetAmount    float64

	User  *User
	Group *Group
}

func (r *RedeemCode) IsUsed() bool {
	return r.Status == StatusUsed
}

func (r *RedeemCode) CanUse() bool {
	return r.Status == StatusUnused
}

func NormalizeRegistrationInvitationCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
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
	raw, err := GenerateCodeWithFormat(format, []byte("0123456789ABCDEF"))
	if err != nil {
		return "", err
	}
	// 当随机部分长度是按 16 进制历史语义配置时，保持大写 0-9A-F 输出。
	return strings.ToUpper(raw), nil
}
