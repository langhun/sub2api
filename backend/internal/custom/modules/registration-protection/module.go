package registrationprotection

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	UserID      int64    `json:"user_id"`
	Email       string   `json:"email"`
	Score       int      `json:"score"`
	Action      string   `json:"action"`
	Signals     []string `json:"signals"`
	ExpiresAt   any      `json:"expires_at"`
	PaymentOpen bool     `json:"payment_unlocked"`
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
	rows, err := m.db.QueryContext(c.Request.Context(), `
		SELECT p.user_id, COALESCE(u.email, ''), p.risk_score, p.risk_action,
		       p.risk_signals, p.risk_expires_at, p.payment_unlocked
		FROM custom_registration_risk_profiles p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE p.risk_action <> 'none'
		ORDER BY p.updated_at DESC LIMIT $1`, limit)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to load registration risk profiles"})
		return
	}
	defer rows.Close()
	items := make([]riskListItem, 0)
	for rows.Next() {
		var item riskListItem
		var raw []byte
		if err := rows.Scan(&item.UserID, &item.Email, &item.Score, &item.Action, &raw, &item.ExpiresAt, &item.PaymentOpen); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode registration risk profile"})
			return
		}
		_ = json.Unmarshal(raw, &item.Signals)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to read registration risk profiles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
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
	_ = c.ShouldBindJSON(&body)
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}
	if _, err := m.db.ExecContext(c.Request.Context(), `
		UPDATE custom_registration_risk_profiles
		SET risk_action = $2, admin_override = $2, override_reason = $3, risk_expires_at = NULL, updated_at = NOW()
		WHERE user_id = $1`, userID, action, body.Reason); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to update registration risk profile"})
		return
	}
	_, _ = m.db.ExecContext(c.Request.Context(), `INSERT INTO custom_registration_risk_audit (user_id, action, reason) VALUES ($1, $2, $3)`, userID, action, body.Reason)
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "action": action})
}

func (m *Module) unlock(c *gin.Context) { m.updateOverride(c, "manual_unlock") }
func (m *Module) lock(c *gin.Context)   { m.updateOverride(c, "manual_lock") }
