// Package openaiat provides the isolated administrator workflow for viewing
// and rotating OpenAI Codex personal access tokens.
package openaiat

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// AccountStore is the deliberately small account surface consumed by this
// module. The upstream AdminService satisfies it without exposing the rest of
// the administrator API to the module.
type AccountStore interface {
	GetAccount(context.Context, int64) (*service.Account, error)
	GetProxy(context.Context, int64) (*service.Proxy, error)
	UpdateAccount(context.Context, int64, *service.UpdateAccountInput) (*service.Account, error)
	ClearAccountError(context.Context, int64) (*service.Account, error)
}

type PATService interface {
	ValidateCodexPersonalAccessToken(context.Context, string, string) (*service.OpenAITokenInfo, error)
	BuildAccountCredentials(*service.OpenAITokenInfo) map[string]any
}

type Module struct {
	accounts    AccountStore
	patService  PATService
	invalidator service.TokenCacheInvalidator
}

func NewModule(accounts AccountStore, patService PATService, invalidator service.TokenCacheInvalidator) *Module {
	return &Module{accounts: accounts, patService: patService, invalidator: invalidator}
}

type replaceRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// RegisterRoutes mounts only this module's administrator endpoints.
func (m *Module) RegisterRoutes(admin *gin.RouterGroup) {
	if m == nil || admin == nil {
		return
	}
	group := admin.Group("/openai/accounts")
	group.GET("/:id/access-token", m.getAccessToken)
	group.POST("/:id/access-token", m.replaceAccessToken)
}

func (m *Module) getAccessToken(c *gin.Context) {
	account, ok := m.accountFromRequest(c)
	if !ok {
		return
	}
	if !isPATAccount(account) {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_PAT_REQUIRED", "account is not an OpenAI personal access token account"))
		return
	}

	token := credentialString(account, "access_token")
	if token == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_PAT_MISSING", "account access token is not configured"))
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	response.Success(c, gin.H{
		"account_id":          account.ID,
		"access_token":        token,
		"masked_access_token": maskToken(token),
	})
}

func (m *Module) replaceAccessToken(c *gin.Context) {
	account, ok := m.accountFromRequest(c)
	if !ok {
		return
	}
	if !isPATAccount(account) {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_PAT_REQUIRED", "account is not an OpenAI personal access token account"))
		return
	}

	var req replaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_PAT_REQUIRED", "access token is required"))
		return
	}
	token := strings.TrimSpace(req.AccessToken)
	if token == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_PAT_REQUIRED", "access token is required"))
		return
	}

	proxyURL, ok := m.proxyURL(c, account)
	if !ok {
		return
	}
	tokenInfo, err := m.patService.ValidateCodexPersonalAccessToken(c.Request.Context(), token, proxyURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	credentials := m.patService.BuildAccountCredentials(tokenInfo)
	for key, value := range account.Credentials {
		if _, exists := credentials[key]; !exists {
			credentials[key] = value
		}
	}
	credentials = service.NormalizeOpenAIPersonalAccessTokenCredentials(account, tokenInfo, credentials)

	updated, err := m.accounts.UpdateAccount(c.Request.Context(), account.ID, &service.UpdateAccountInput{Credentials: credentials})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if cleared, clearErr := m.accounts.ClearAccountError(c.Request.Context(), account.ID); clearErr == nil && cleared != nil {
		updated = cleared
	}
	if m.invalidator != nil {
		_ = m.invalidator.InvalidateToken(c.Request.Context(), updated)
	}

	c.Header("Cache-Control", "no-store")
	response.Success(c, gin.H{
		"account":             dto.AccountFromService(updated),
		"masked_access_token": maskToken(token),
	})
}

func (m *Module) accountFromRequest(c *gin.Context) (*service.Account, bool) {
	accountID, err := parseAccountID(c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return nil, false
	}
	account, err := m.accounts.GetAccount(c.Request.Context(), accountID)
	if err != nil || account == nil {
		if err == nil {
			err = infraerrors.NotFound("ACCOUNT_NOT_FOUND", "account not found")
		}
		response.ErrorFrom(c, err)
		return nil, false
	}
	return account, true
}

func (m *Module) proxyURL(c *gin.Context, account *service.Account) (string, bool) {
	if account.ProxyID == nil || *account.ProxyID == 0 {
		return "", true
	}
	proxy, err := m.accounts.GetProxy(c.Request.Context(), *account.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return "", false
	}
	if proxy == nil {
		return "", true
	}
	return proxy.URL(), true
}

func isPATAccount(account *service.Account) bool {
	return account != nil && account.Platform == service.PlatformOpenAI && account.IsOpenAIPersonalAccessToken() && !account.IsCredentialShadow()
}

func credentialString(account *service.Account, key string) string {
	if account == nil || account.Credentials == nil {
		return ""
	}
	value, _ := account.Credentials[key].(string)
	return strings.TrimSpace(value)
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 10 {
		return strings.Repeat("*", len(token))
	}
	return token[:7] + "..." + token[len(token)-4:]
}

func parseAccountID(value string) (int64, error) {
	var id int64
	if _, err := fmt.Sscan(value, &id); err != nil || id <= 0 {
		return 0, infraerrors.BadRequest("INVALID_ACCOUNT_ID", "invalid account ID")
	}
	return id, nil
}
