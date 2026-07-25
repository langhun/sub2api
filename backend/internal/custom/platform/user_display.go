package platform

import "github.com/Wei-Shaw/sub2api/internal/service"

// UserDisplayName exposes the stable platform identity projection to custom
// modules without making their business ports depend on service.User.
func UserDisplayName(username, email string, userID int64) string {
	return service.UserDisplayName(username, email, userID)
}

// MaskedUserDisplayName preserves the core's response-safe identity format.
func MaskedUserDisplayName(username, email string, userID int64) string {
	return service.MaskedUserDisplayName(username, email, userID)
}
