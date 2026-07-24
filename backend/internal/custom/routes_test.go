package custom_test

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutesDoesNotAddRoutesBeforeModulesExist(t *testing.T) {
	router := gin.New()

	custom.RegisterRoutes(router, custom.NewRuntime())

	require.Empty(t, router.Routes())
}
