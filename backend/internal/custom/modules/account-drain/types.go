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

type CreatePlanInput struct {
	Name       string     `json:"name" binding:"required,max=120"`
	AccountIDs []int64    `json:"account_ids" binding:"required,min=1,max=50"`
	ExpiresAt  *time.Time `json:"expires_at"`
}
