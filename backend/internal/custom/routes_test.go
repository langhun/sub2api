package custom_test

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom"
	activitycheckin "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/checkin"
	activityleaderboard "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/leaderboard"
	activityredpacket "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/redpacket"
	activityrewards "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	walletextension "github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutesDoesNotAddRoutesWithoutRuntime(t *testing.T) {
	router := gin.New()

	custom.RegisterRoutes(router, nil, nil, nil, nil, nil)

	require.Empty(t, router.Routes())
}

func TestRegisterRoutesMountsGameHallModule(t *testing.T) {
	router := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }

	custom.RegisterRoutes(
		router,
		&custom.Runtime{
			ActivityCheckin:     activitycheckin.NewModule(nil, nil),
			ActivityLeaderboard: activityleaderboard.NewModule(nil, activityleaderboard.Readers{}),
			ActivityRedPacket:   activityredpacket.NewModule(nil, nil),
			ActivityRewardsHTTP: activityrewards.NewModule(routeHandlerStub{}),
			GameHall:            gamehall.NewModule(nil),
			WalletExtension:     walletextension.NewModule(nil),
		},
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	_, exists := routes["POST /api/v1/game-hall/play"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/admin/game-hall/rounds"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/transfer"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/transfer/stats"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/transfer/leaderboard"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/admin/transfers/batch"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/checkin"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/public/leaderboard/balance"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/redpacket"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/admin/blindbox/prize-items"]
	require.True(t, exists)
}

type routeHandlerStub struct{}

func (routeHandlerStub) ListPrizeItems(*gin.Context)           {}
func (routeHandlerStub) CreatePrizeItem(*gin.Context)          {}
func (routeHandlerStub) UpdatePrizeItem(*gin.Context)          {}
func (routeHandlerStub) DeletePrizeItem(*gin.Context)          {}
func (routeHandlerStub) GetStats(*gin.Context)                 {}
func (routeHandlerStub) ListRewardDeliveries(*gin.Context)     {}
func (routeHandlerStub) RetryRewardDelivery(*gin.Context)      {}
func (routeHandlerStub) CompensateRewardDelivery(*gin.Context) {}
