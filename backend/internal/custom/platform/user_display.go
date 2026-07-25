package platform

import (
	"fmt"
	"strings"
)

// UserDisplayName owns the identity projection used by custom modules.
func UserDisplayName(username, email string, userID int64) string {
	if value := strings.TrimSpace(username); value != "" {
		return value
	}
	if value := strings.TrimSpace(email); value != "" {
		return value
	}
	if userID > 0 {
		return fmt.Sprintf("用户 #%d", userID)
	}
	return "用户"
}

// MaskedUserDisplayName hides email addresses when no username is available.
func MaskedUserDisplayName(username, email string, userID int64) string {
	if value := strings.TrimSpace(username); value != "" {
		return value
	}
	if value := strings.TrimSpace(email); value != "" {
		return maskEmail(value)
	}
	return UserDisplayName("", "", userID)
}

func maskEmail(email string) string {
	if len(email) < 3 {
		return "***"
	}
	atIndex := strings.IndexByte(email, '@')
	if atIndex < 1 {
		return email[:1] + "***"
	}
	localPart := email[:atIndex]
	if len(localPart) <= 2 {
		return localPart[:1] + "***" + email[atIndex:]
	}
	return localPart[:1] + "***" + localPart[len(localPart)-1:] + email[atIndex:]
}
