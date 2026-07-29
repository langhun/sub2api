package accountdrain

import "time"

const (
	StatusActive  = "active"
	StatusStopped = "stopped"
	StatusExpired = "expired"
)

type Plan struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expires_at"`
	AccountIDs []int64    `json:"account_ids"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AccountTargetStatus is the account-management view of the internal plans.
// Plan names and grouping are deliberately not part of this API surface.
type AccountTargetStatus struct {
	AccountID int64 `json:"account_id"`
	Active    bool  `json:"active"`
}
