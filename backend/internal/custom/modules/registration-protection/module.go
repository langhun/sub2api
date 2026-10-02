package registrationprotection

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type Module struct {
	db   *sql.DB
	risk *RiskStore
}

func NewModule(db *sql.DB, risk *RiskStore) *Module {
	return &Module{db: db, risk: risk}
}

func (m *Module) RegisterRoutes(admin *gin.RouterGroup) {
	if m == nil || admin == nil {
		return
	}
	group := admin.Group("/registration-risk")
	group.GET("", m.list)
	group.POST("/:user_id/unlock", m.unlock)
	group.POST("/:user_id/lock", m.lock)
}

type riskListItem struct {
	UserID          int64                `json:"user_id"`
	Email           string               `json:"email"`
	Score           int                  `json:"score"`
	Action          string               `json:"action"`
	Signals         []string             `json:"signals"`
	Evidence        []riskEvidence       `json:"evidence"`
	RelatedAccounts []riskRelatedAccount `json:"related_accounts"`
	ExpiresAt       any                  `json:"expires_at"`
	PaymentOpen     bool                 `json:"payment_unlocked"`
}

type riskRelatedAccount struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
}

// riskEvidence is intentionally descriptive rather than raw telemetry. The
// detail is a stable translation key consumed by the admin UI, so this API
// never exposes IP addresses, email addresses, user agents, or device values.
type riskEvidence struct {
	Signal string `json:"signal"`
	Detail string `json:"detail"`
}

func evidenceForSignals(signals []string) []riskEvidence {
	evidence := make([]riskEvidence, 0, len(signals))
	for _, signal := range signals {
		detail := "unknown"
		switch signal {
		case "exact_fingerprint_repeat":
			detail = "fingerprint_repeat"
		case "api_key_created":
			detail = "api_key_created"
		case "fast_checkin":
			detail = "fast_checkin"
		case "cluster_6h":
			detail = "cluster_6h"
		case "cluster_24h":
			detail = "cluster_24h"
		}
		evidence = append(evidence, riskEvidence{Signal: signal, Detail: detail})
	}
	return evidence
}

func (m *Module) list(c *gin.Context) {
	if m == nil || m.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registration risk store unavailable"})
		return
	}
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	offset := 0
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		offset = parsed
	}
	rows, err := m.db.QueryContext(c.Request.Context(), `
		SELECT p.user_id, COALESCE(u.email, ''), p.risk_score, p.risk_action,
		       p.risk_signals, p.risk_expires_at, p.payment_unlocked,
		       COALESCE(related.related_accounts, '[]'::jsonb)
		FROM custom_registration_risk_profiles p
		LEFT JOIN users u ON u.id = p.user_id
		LEFT JOIN LATERAL (
			SELECT jsonb_agg(
				jsonb_build_object('user_id', matches.user_id, 'email', matches.email)
				ORDER BY matches.registered_at, matches.user_id
			) AS related_accounts
			FROM (
				SELECT e.user_id, COALESCE(u2.email, '') AS email, MIN(e.registered_at) AS registered_at
				FROM custom_registration_fingerprint_events e
				LEFT JOIN users u2 ON u2.id = e.user_id
				WHERE e.user_id IS NOT NULL
				  AND e.user_id <> p.user_id
				  AND e.fingerprint_hash IN (
					SELECT DISTINCT e2.fingerprint_hash
					FROM custom_registration_fingerprint_events e2
					WHERE e2.user_id = p.user_id AND e2.is_repeat = TRUE
				  )
				GROUP BY e.user_id, u2.email
				ORDER BY MIN(e.registered_at), e.user_id
				LIMIT 50
			) matches
		) related ON TRUE
		WHERE p.risk_action <> 'none'
		   OR p.risk_score > 0 OR p.risk_signals <> '[]'::jsonb
		ORDER BY p.updated_at DESC, p.user_id DESC LIMIT $1 OFFSET $2`, limit+1, offset)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to load registration risk profiles"})
		return
	}
	defer func() { _ = rows.Close() }()
	items := make([]riskListItem, 0)
	for rows.Next() {
		var item riskListItem
		var raw []byte
		var relatedRaw []byte
		if err := rows.Scan(&item.UserID, &item.Email, &item.Score, &item.Action, &raw, &item.ExpiresAt, &item.PaymentOpen, &relatedRaw); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode registration risk profile"})
			return
		}
		var signals []string
		if err := json.Unmarshal(raw, &signals); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode registration risk signals"})
			return
		}
		item.Signals = make([]string, 0, len(signals))
		seen := make(map[string]bool, len(signals))
		for _, signal := range signals {
			if !seen[signal] {
				item.Signals = append(item.Signals, signal)
				seen[signal] = true
			}
		}
		item.Evidence = evidenceForSignals(item.Signals)
		if err := json.Unmarshal(relatedRaw, &item.RelatedAccounts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode related registration accounts"})
			return
		}
		if item.Action == RiskActionNone {
			item.Action = "observe"
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to read registration risk profiles"})
		return
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "has_more": hasMore})
}

func (m *Module) updateOverride(c *gin.Context, action string) {
	if m == nil || m.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registration risk store unavailable"})
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" || utf8.RuneCountInString(body.Reason) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason must contain 1 to 500 characters"})
		return
	}
	tx, err := m.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to update registration risk profile"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(c.Request.Context(), `
		UPDATE custom_registration_risk_profiles
		SET risk_action = $2, admin_override = $2, override_reason = $3, risk_expires_at = NULL, updated_at = NOW()
		WHERE user_id = $1`, userID, action, body.Reason)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to update registration risk profile"})
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to update registration risk profile"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "registration risk profile not found"})
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO custom_registration_risk_audit (user_id, action, reason) VALUES ($1, $2, $3)`, userID, action, body.Reason); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to audit registration risk override"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to commit registration risk override"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "action": action})
}

func (m *Module) unlock(c *gin.Context) { m.updateOverride(c, RiskActionManualUnlock) }
func (m *Module) lock(c *gin.Context)   { m.updateOverride(c, RiskActionManualLock) }
