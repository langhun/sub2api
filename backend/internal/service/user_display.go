package service

import (
	"fmt"
	"strings"
)

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
