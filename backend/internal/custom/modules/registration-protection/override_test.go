package registrationprotection

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	registrationsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection/settings"
	"github.com/stretchr/testify/require"
)

func expectRiskProfile(mock sqlmock.Sqlmock, action string, expires any, paid bool) {
	mock.ExpectQuery("SELECT risk_score, risk_action, risk_signals").WithArgs(int64(986)).
		WillReturnRows(sqlmock.NewRows([]string{"risk_score", "risk_action", "risk_signals", "risk_expires_at", "payment_unlocked"}).
			AddRow(2, action, `["exact_fingerprint_repeat"]`, expires, paid))
}

func TestManualRiskDecisionTakesPrecedenceOverPaymentsAndExpiry(t *testing.T) {
	for _, action := range []string{RiskActionManualLock, RiskActionManualUnlock} {
		t.Run(action, func(t *testing.T) {
			db, mock := newFingerprintSQLMock(t)
			defer func() { _ = db.Close() }()
			expectRiskProfile(mock, action, time.Now().Add(-time.Hour), true)
			decision, err := NewRiskStore(db, nil).Decision(context.Background(), 986)
			require.NoError(t, err)
			require.Equal(t, action, decision.Action)
			require.Equal(t, []string{"exact_fingerprint_repeat"}, decision.Signals)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestManualOverrideEnforcesBenefitGates(t *testing.T) {
	for _, action := range []string{RiskActionManualLock, RiskActionManualUnlock} {
		for _, operation := range []string{"checkin", "api_key", "reward"} {
			t.Run(action+"/"+operation, func(t *testing.T) {
				db, mock := newFingerprintSQLMock(t)
				defer func() { _ = db.Close() }()
				policy := NewPolicy(policySettingsStub{}, NewStore(db))
				expectRiskProfile(mock, action, nil, true)
				var err error
				switch operation {
				case "checkin":
					err = policy.CheckinRewardAllowed(context.Background(), 986)
				case "api_key":
					err = policy.AllowAPIKeyCreation(context.Background(), 986)
				case "reward":
					var blocked bool
					blocked, err = policy.IsRewardBlocked(context.Background(), 986)
					require.NoError(t, err)
					require.Equal(t, action == RiskActionManualLock, blocked)
				}
				if operation != "reward" && action == RiskActionManualLock {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}

func TestLowScoreObservationKeepsExistingEligibility(t *testing.T) {
	for _, repeat := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "duplicate"}[repeat], func(t *testing.T) {
			db, mock := newFingerprintSQLMock(t)
			defer func() { _ = db.Close() }()
			policy := NewPolicy(policySettingsStub{}, NewStore(db))
			expectRiskProfile(mock, RiskActionNone, nil, false)
			mock.ExpectQuery("SELECT COALESCE\\(BOOL_OR").WithArgs(int64(986)).WillReturnRows(sqlmock.NewRows([]string{"blocked"}).AddRow(repeat))
			err := policy.CheckinRewardAllowed(context.Background(), 986)
			if repeat {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAutomaticRiskExpiryAndPaymentUnlockRemainEffective(t *testing.T) {
	for _, paid := range []bool{false, true} {
		t.Run(map[bool]string{false: "expired", true: "paid"}[paid], func(t *testing.T) {
			db, mock := newFingerprintSQLMock(t)
			defer func() { _ = db.Close() }()
			expectRiskProfile(mock, RiskActionMedium, time.Now().Add(-time.Hour), false)
			settings := policySettingsStub{values: map[string]string{registrationsettings.KeyPaymentUnlock: "false"}}
			if paid {
				settings.values[registrationsettings.KeyPaymentUnlock] = "true"
				mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(int64(986)).WillReturnRows(sqlmock.NewRows([]string{"paid"}).AddRow(10.0))
				mock.ExpectExec("UPDATE custom_registration_risk_profiles").WithArgs(int64(986)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			decision, err := NewRiskStore(db, settings).Decision(context.Background(), 986)
			require.NoError(t, err)
			require.Equal(t, RiskActionNone, decision.Action)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
