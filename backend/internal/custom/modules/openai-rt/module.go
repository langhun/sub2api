// Package openairt provides the isolated administrator workflow for viewing
// and rotating OpenAI OAuth refresh tokens.
package openairt

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

type OAuthService interface {
	RefreshTokenWithClientID(context.Context, string, string, string) (*service.OpenAITokenInfo, error)
	BuildAccountCredentials(*service.OpenAITokenInfo) map[string]any
}

type Module struct {
	accounts     AccountStore
	oauthService OAuthService
	invalidator  service.TokenCacheInvalidator
}

func NewModule(accounts AccountStore, oauthService OAuthService, invalidator service.TokenCacheInvalidator) *Module {
	return &Module{accounts: accounts, oauthService: oauthService, invalidator: invalidator}
}

type replaceRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RegisterRoutes mounts only this module's administrator endpoints.
func (m *Module) RegisterRoutes(admin *gin.RouterGroup) {
	if m == nil || admin == nil {
		return
	}
	group := admin.Group("/openai/accounts")
	group.GET("/:id/refresh-token", m.getRefreshToken)
	group.POST("/:id/refresh-token", m.replaceRefreshToken)
}

func (m *Module) getRefreshToken(c *gin.Context) {
	account, ok := m.accountFromRequest(c)
	if !ok {
		return
	}
	if !isRTAccount(account) {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_RT_REQUIRED", "account is not an OpenAI OAuth refresh token account"))
		return
	}

	token := credentialString(account, "refresh_token")
	if token == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_RT_MISSING", "account refresh token is not configured"))
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	response.Success(c, gin.H{
		"account_id":           account.ID,
		"refresh_token":        token,
		"masked_refresh_token": maskToken(token),
	})
}

func (m *Module) replaceRefreshToken(c *gin.Context) {
	account, ok := m.accountFromRequest(c)
	if !ok {
		return
	}
	if !isRTAccount(account) {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_RT_REQUIRED", "account is not an OpenAI OAuth refresh token account"))
		return
	}

	var req replaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_RT_REQUIRED", "refresh token is required"))
		return
	}
	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("OPENAI_RT_REQUIRED", "refresh token is required"))
		return
	}

	proxyURL, ok := m.proxyURL(c, account)
	if !ok {
		return
	}
	tokenInfo, err := m.oauthService.RefreshTokenWithClientID(c.Request.Context(), token, proxyURL, credentialString(account, "client_id"))
	if err != nil {
		writeRefreshTokenError(c, err)
		return
	}

	credentials := m.oauthService.BuildAccountCredentials(tokenInfo)
	// Some OAuth servers do not rotate the RT. The submitted, validated RT must
	// still replace the old one instead of being overwritten during the merge.
	if credentialStringFromMap(credentials, "refresh_token") == "" {
		credentials["refresh_token"] = token
	}
	for key, value := range account.Credentials {
		if _, exists := credentials[key]; !exists {
			credentials[key] = value
		}
	}
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
		"account":              dto.AccountFromService(updated),
		"masked_refresh_token": maskToken(credentialString(updated, "refresh_token")),
	})
}

func writeRefreshTokenError(c *gin.Context, err error) {
	if infraerrors.Reason(err) == "OPENAI_OAUTH_TOKEN_REFRESH_FAILED" {
		message := strings.ToLower(infraerrors.Message(err))
		if strings.Contains(message, "refresh_token_invalidated") ||
			strings.Contains(message, "session has ended") ||
			strings.Contains(message, "status 401") {
			response.ErrorFrom(c, infraerrors.BadRequest(
				"OPENAI_RT_INVALID",
				"refresh token is invalid or expired; sign in again and provide a new RT",
			))
			return
		}
	}
	response.ErrorFrom(c, err)
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

func isRTAccount(account *service.Account) bool {
	return account != nil &&
		account.Platform == service.PlatformOpenAI &&
		account.Type == service.AccountTypeOAuth &&
		!account.IsOpenAIPersonalAccessToken() &&
		!account.IsCredentialShadow() &&
		credentialString(account, "refresh_token") != ""
}

func credentialString(account *service.Account, key string) string {
	if account == nil || account.Credentials == nil {
		return ""
	}
	value, _ := account.Credentials[key].(string)
	return strings.TrimSpace(value)
}

func credentialStringFromMap(credentials map[string]any, key string) string {
	value, _ := credentials[key].(string)
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
