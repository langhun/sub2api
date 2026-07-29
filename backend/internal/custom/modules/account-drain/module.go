// Package accountdrain owns the optional account-directed consumption Overlay.
package accountdrain

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const ModuleID = "account_drain"

// Module groups the custom plan lifecycle, HTTP surface, and gateway policy.
type Module struct {
	Service *Service
	Handler *Handler
	Policy  *Policy
}

func NewModule(ctx context.Context, db *sql.DB, gateway *service.OpenAIGatewayService) (*Module, error) {
	planService := NewService(NewRepository(db))
	if err := planService.RefreshPolicy(ctx); err != nil {
		return nil, err
	}
	if gateway != nil {
		gateway.SetOpenAIAccountSchedulingPolicy(planService.Policy())
	}
	return &Module{
		Service: planService,
		Handler: NewHandler(planService),
		Policy:  planService.Policy(),
	}, nil
}
