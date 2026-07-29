package custom

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	codeformat "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ProvideRuntime starts module-owned background workers after composition.
func ProvideRuntime(
	client *dbent.Client,
	db *sql.DB,
	activityWalletCapabilities *platform.ActivityWalletCapabilities,
	redeemService *service.RedeemService,
	adminService service.AdminService,
	customSettingsRegistry *customsettings.Registry,
	codeGenerator *codeformat.Generator,
) (*Runtime, error) {
	runtime, err := NewRuntime(
		client,
		db,
		activityWalletCapabilities,
		redeemService,
		adminService,
		customSettingsRegistry,
		codeGenerator,
	)
	if err != nil {
		return nil, err
	}
	runtime.Start()
	return runtime, nil
}

// Start launches the module-owned background workers once dependencies exist.
func (r *Runtime) Start() {
	if r == nil {
		return
	}
	if r.ActivityRedPacket != nil {
		r.ActivityRedPacket.Start()
	}
	if r.ActivityRewards != nil {
		r.ActivityRewards.Start(context.Background())
	}
}

// Stop waits for every module-owned background worker to finish.
func (r *Runtime) Stop() {
	if r == nil {
		return
	}
	if r.ActivityRewards != nil {
		r.ActivityRewards.Stop()
	}
	if r.ActivityRedPacket != nil {
		r.ActivityRedPacket.Stop()
	}
}
