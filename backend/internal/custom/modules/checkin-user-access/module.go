// Package checkinuseraccess owns the custom per-user check-in eligibility
// Overlay. It never changes upstream user status or account permissions.
package checkinuseraccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	activitycontract "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const unavailableReasonAdminDisabled = "admin_disabled"

var errUserNotFound = errors.New("check-in access user not found")

// Access is the administrator-visible per-user decision.
type Access struct {
	UserID    int64     `json:"user_id"`
	Disabled  bool      `json:"disabled"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Module owns both the administrator API and the narrow policy injected into
// the existing check-in Overlay.
type Module struct {
	db          *sql.DB
	Eligibility *Eligibility
}

// Eligibility composes the existing reward-risk policy after this module's
// per-user access decision. The access decision runs first so disabled users
// cannot create a check-in, change a balance, or trigger a blind-box reward.
type Eligibility struct {
	db   *sql.DB
	next activitycontract.RewardEligibility
}

var _ activitycontract.RewardEligibility = (*Eligibility)(nil)
var _ activitycontract.CheckinAvailabilityProvider = (*Eligibility)(nil)

func NewModule(db *sql.DB, next activitycontract.RewardEligibility) *Module {
	return &Module{db: db, Eligibility: &Eligibility{db: db, next: next}}
}

// RegisterRoutes mounts only the administrator-facing Overlay endpoints.
func (m *Module) RegisterRoutes(admin *gin.RouterGroup) {
	if m == nil || admin == nil {
		return
	}
	group := admin.Group("/checkin/users")
	group.GET("/:user_id/access", m.getAccess)
	group.PUT("/:user_id/access", m.updateAccess)
}

func (m *Module) getAccess(c *gin.Context) {
	userID, ok := requestUserID(c)
	if !ok {
		return
	}
	access, err := getAccess(c.Request.Context(), m.db, userID)
	if err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, access)
}

type updateAccessRequest struct {
	Disabled *bool `json:"disabled" binding:"required"`
}

func (m *Module) updateAccess(c *gin.Context) {
	userID, ok := requestUserID(c)
	if !ok {
		return
	}
	var request updateAccessRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Disabled == nil {
		response.ErrorFrom(c, infraerrors.BadRequest("CHECKIN_ACCESS_INVALID_REQUEST", "disabled is required"))
		return
	}
	access, err := setAccess(c.Request.Context(), m.db, userID, *request.Disabled, administratorID(c))
	if err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, access)
}

func requestUserID(c *gin.Context) (int64, bool) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("CHECKIN_ACCESS_INVALID_USER", "invalid user_id"))
		return 0, false
	}
	return userID, true
}

func administratorID(c *gin.Context) *int64 {
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		return &subject.UserID
	}
	return nil
}

func writeAccessError(c *gin.Context, err error) {
	if errors.Is(err, errUserNotFound) {
		response.ErrorFrom(c, infraerrors.NotFound("CHECKIN_ACCESS_USER_NOT_FOUND", "user not found"))
		return
	}
	response.ErrorFrom(c, infraerrors.InternalServer("CHECKIN_ACCESS_UNAVAILABLE", "check-in access is temporarily unavailable").WithCause(err))
}

func getAccess(ctx context.Context, db *sql.DB, userID int64) (*Access, error) {
	if db == nil {
		return nil, errors.New("nil check-in access database")
	}
	access := &Access{UserID: userID}
	err := db.QueryRowContext(ctx, `
		SELECT disabled, reason, updated_by, updated_at
		FROM custom_checkin_user_access
		WHERE user_id = $1`, userID).
		Scan(&access.Disabled, &access.Reason, &access.UpdatedBy, &access.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, userID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("verify check-in access user: %w", err)
		}
		if !exists {
			return nil, errUserNotFound
		}
		return access, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read check-in access: %w", err)
	}
	return access, nil
}

func setAccess(ctx context.Context, db *sql.DB, userID int64, disabled bool, updatedBy *int64) (*Access, error) {
	if db == nil {
		return nil, errors.New("nil check-in access database")
	}
	access := &Access{}
	err := db.QueryRowContext(ctx, `
		INSERT INTO custom_checkin_user_access (user_id, disabled, reason, updated_by, created_at, updated_at)
		SELECT id, $2, 'refund', $3, NOW(), NOW()
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		ON CONFLICT (user_id) DO UPDATE
		SET disabled = EXCLUDED.disabled,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
		RETURNING user_id, disabled, reason, updated_by, updated_at`, userID, disabled, updatedBy).
		Scan(&access.UserID, &access.Disabled, &access.Reason, &access.UpdatedBy, &access.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update check-in access: %w", err)
	}
	return access, nil
}

// CheckinRewardAllowed is consumed by ordinary and lucky check-in mutations.
func (e *Eligibility) CheckinRewardAllowed(ctx context.Context, userID int64) error {
	availability, err := e.CheckinAvailability(ctx, userID)
	if err != nil {
		return err
	}
	if !availability.Allowed {
		return infraerrors.Forbidden("CHECKIN_ACCESS_DISABLED", "check-in is disabled for this account")
	}
	if e != nil && e.next != nil {
		return e.next.CheckinRewardAllowed(ctx, userID)
	}
	return nil
}

// CheckinAvailability lets the existing custom check-in status endpoint hide
// actions before a disabled user tries a mutation.
func (e *Eligibility) CheckinAvailability(ctx context.Context, userID int64) (activitycontract.CheckinAvailability, error) {
	if e == nil || e.db == nil {
		return activitycontract.CheckinAvailability{}, infraerrors.ServiceUnavailable("CHECKIN_ACCESS_UNAVAILABLE", "check-in access is temporarily unavailable")
	}
	var disabled bool
	err := e.db.QueryRowContext(ctx, `SELECT disabled FROM custom_checkin_user_access WHERE user_id = $1`, userID).Scan(&disabled)
	if errors.Is(err, sql.ErrNoRows) {
		return activitycontract.CheckinAvailability{Allowed: true}, nil
	}
	if err != nil {
		return activitycontract.CheckinAvailability{}, infraerrors.ServiceUnavailable("CHECKIN_ACCESS_UNAVAILABLE", "check-in access is temporarily unavailable").WithCause(err)
	}
	if disabled {
		return activitycontract.CheckinAvailability{Allowed: false, Reason: unavailableReasonAdminDisabled}, nil
	}
	return activitycontract.CheckinAvailability{Allowed: true}, nil
}

// StatusUnavailableReason is deliberately stable for the user-facing
// check-in Overlay; it does not disclose the administrator's refund note.
func StatusUnavailableReason() string { return unavailableReasonAdminDisabled }
