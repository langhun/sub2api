package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
)

const (
	// 手动设置 OpenAI privacy 时，若 access_token 即将过期，先复用现有刷新链路自愈。
	manualPrivacyOpenAIRefreshWindow = 3 * time.Minute
	// Antigravity 的隐私请求本身很短，仅在非常接近过期时才提前刷新，避免无意义刷新。
	manualPrivacyAntigravityRefreshWindow = time.Minute
	manualPrivacyErrorMessageLimit        = 200
)

func (h *AccountHandler) prepareAccountForPrivacy(ctx context.Context, account *service.Account) (*service.Account, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_PRIVACY_ACCOUNT_REQUIRED", "account is required")
	}

	switch account.Platform {
	case service.PlatformOpenAI:
		return h.prepareOpenAIAccountForPrivacy(ctx, account)
	case service.PlatformAntigravity:
		return h.prepareAntigravityAccountForPrivacy(ctx, account)
	default:
		return account, nil
	}
}

func (h *AccountHandler) prepareOpenAIAccountForPrivacy(ctx context.Context, account *service.Account) (*service.Account, error) {
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return account, nil
	}

	now := time.Now()
	if !now.Before(*expiresAt) && strings.TrimSpace(account.GetOpenAIRefreshToken()) == "" {
		return nil, infraerrors.BadRequest(
			"ACCOUNT_PRIVACY_REFRESH_TOKEN_MISSING",
			"access_token expired and refresh_token is missing; re-authorize this account",
		)
	}
	if h.adminService == nil {
		return account, nil
	}
	if time.Until(*expiresAt) > manualPrivacyOpenAIRefreshWindow {
		return account, nil
	}
	if strings.TrimSpace(account.GetOpenAIRefreshToken()) == "" {
		return account, nil
	}

	updated, err := h.adminService.RefreshAccountCredentials(ctx, account.ID)
	if err != nil {
		return nil, normalizePrivacyPreparationError(
			err,
			http.StatusBadGateway,
			"ACCOUNT_PRIVACY_REFRESH_FAILED",
			"failed to refresh OpenAI access token",
		)
	}
	return updated, nil
}

func (h *AccountHandler) prepareAntigravityAccountForPrivacy(ctx context.Context, account *service.Account) (*service.Account, error) {
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return account, nil
	}

	now := time.Now()
	if !now.Before(*expiresAt) && strings.TrimSpace(account.GetCredential("refresh_token")) == "" {
		return nil, infraerrors.BadRequest(
			"ACCOUNT_PRIVACY_REFRESH_TOKEN_MISSING",
			"access_token expired and refresh_token is missing; re-authorize this account",
		)
	}
	if h.adminService == nil {
		return account, nil
	}
	if time.Until(*expiresAt) > manualPrivacyAntigravityRefreshWindow {
		return account, nil
	}
	if strings.TrimSpace(account.GetCredential("refresh_token")) == "" {
		return account, nil
	}

	updated, err := h.adminService.RefreshAccountCredentials(ctx, account.ID)
	if err != nil {
		return nil, normalizePrivacyPreparationError(
			err,
			http.StatusBadGateway,
			"ACCOUNT_PRIVACY_REFRESH_FAILED",
			"failed to refresh Antigravity access token",
		)
	}
	return updated, nil
}

func normalizePrivacyPreparationError(err error, statusCode int, reason, fallback string) error {
	if err == nil {
		return nil
	}

	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		message := strings.TrimSpace(appErr.Message)
		if message == "" {
			message = fallback
		}
		normalized := infraerrors.New(int(appErr.Code), appErr.Reason, truncatePrivacyPreparationMessage(message))
		if len(appErr.Metadata) > 0 {
			normalized = normalized.WithMetadata(appErr.Metadata)
		}
		return normalized.WithCause(err)
	}

	message := strings.TrimSpace(logredact.RedactText(err.Error()))
	if message == "" {
		message = fallback
	} else if message != fallback {
		message = fallback + ": " + message
	}
	return infraerrors.New(statusCode, reason, truncatePrivacyPreparationMessage(message)).WithCause(err)
}

func writePrivacyPreparationError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	statusCode := http.StatusBadGateway
	reason := ""
	var metadata map[string]string
	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		statusCode = int(appErr.Code)
		reason = appErr.Reason
		metadata = appErr.Metadata
	}

	responseMessage := "Cannot set privacy: " + privacyPreparationErrorMessage(err)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(responseMessage)), "cannot set privacy: cannot set privacy:") {
		responseMessage = "Cannot set privacy: " + strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(responseMessage), "Cannot set privacy:"))
	}
	response.ErrorWithDetails(c, statusCode, responseMessage, reason, metadata)
}

func privacyPreparationErrorMessage(err error) string {
	if err == nil {
		return "privacy preparation failed"
	}

	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		if message := strings.TrimSpace(appErr.Message); message != "" {
			return truncatePrivacyPreparationMessage(message)
		}
	}

	message := strings.TrimSpace(logredact.RedactText(err.Error()))
	if message == "" {
		return "privacy preparation failed"
	}
	return truncatePrivacyPreparationMessage(message)
}

func truncatePrivacyPreparationMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}

	runes := []rune(message)
	if len(runes) <= manualPrivacyErrorMessageLimit {
		return message
	}
	return string(runes[:manualPrivacyErrorMessageLimit]) + "..."
}
