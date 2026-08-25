//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type registrationRiskPolicyStub struct {
	reservation *registrationRiskReservationStub
	err         error
	blocked     bool
	blockedErr  error
	inputs      []registrationRiskPolicyInput
}

type registrationRiskPolicyInput struct {
	clientIP  string
	userAgent string
	deviceID  string
	source    string
}

func (s *registrationRiskPolicyStub) Begin(_ context.Context, clientIP, userAgent, deviceID, signupSource string) (RegistrationRiskReservation, error) {
	s.inputs = append(s.inputs, registrationRiskPolicyInput{
		clientIP:  clientIP,
		userAgent: userAgent,
		deviceID:  deviceID,
		source:    signupSource,
	})
	if s.err != nil {
		return nil, s.err
	}
	return s.reservation, nil
}

func (s *registrationRiskPolicyStub) IsRewardBlocked(context.Context, int64) (bool, error) {
	return s.blocked, s.blockedErr
}

type registrationRiskReservationStub struct {
	rewardBlocked bool
	blockSignup   bool
	blockOnCommit bool
	commitErr     error
	abortErr      error
	committedIDs  []int64
	aborts        int
}

func (s *registrationRiskReservationStub) RewardBlocked() bool     { return s.rewardBlocked }
func (s *registrationRiskReservationStub) BlockRegistration() bool { return s.blockSignup }

func (s *registrationRiskReservationStub) Commit(_ context.Context, userID int64) error {
	s.committedIDs = append(s.committedIDs, userID)
	if s.blockOnCommit {
		s.rewardBlocked = true
	}
	return s.commitErr
}

func (s *registrationRiskReservationStub) Abort(context.Context) error {
	s.aborts++
	return s.abortErr
}

func TestRegistrationClientMetadataContext(t *testing.T) {
	ctx := WithRegistrationClientMetadata(context.Background(), " 203.0.113.7 ", " Browser/1.0 ")
	require.Equal(t, RegistrationClientMetadata{
		ClientIP:  "203.0.113.7",
		UserAgent: "Browser/1.0",
	}, RegistrationClientMetadataFromContext(ctx))
	require.Empty(t, RegistrationClientMetadataFromContext(context.Background()))
}

func TestAuthService_Register_RegistrationRiskDuplicateSuppressesSignupRewards(t *testing.T) {
	repo := &userRepoStub{nextID: 801}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyDefaultSubscriptions: `[{"group_id":11,"validity_days":30}]`,
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	riskReservation := &registrationRiskReservationStub{rewardBlocked: true}
	risk := &registrationRiskPolicyStub{reservation: riskReservation}
	svc.SetRegistrationRiskPolicy(risk)

	ctx := WithRegistrationClientMetadata(context.Background(), "203.0.113.7", "Mozilla/5.0")
	_, user, err := svc.Register(ctx, "risk@example.com", "password")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Empty(t, assigner.calls)
	require.Equal(t, []int64{801}, riskReservation.committedIDs)
	require.Zero(t, riskReservation.aborts)
	require.Equal(t, []registrationRiskPolicyInput{{
		clientIP:  "203.0.113.7",
		userAgent: "Mozilla/5.0",
		source:    "email",
	}}, risk.inputs)
}

func TestAuthService_Register_RegistrationRiskUnavailableFailsClosedForRewards(t *testing.T) {
	repo := &userRepoStub{nextID: 802}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyDefaultSubscriptions: `[{"group_id":11,"validity_days":30}]`,
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	svc.SetRegistrationRiskPolicy(&registrationRiskPolicyStub{err: errors.New("store unavailable")})

	_, user, err := svc.Register(context.Background(), "failclosed@example.com", "password")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Empty(t, assigner.calls)
}

func TestAuthService_Register_HighRegistrationRiskBlocksBeforeUserCreation(t *testing.T) {
	repo := &userRepoStub{nextID: 807}
	svc := newAuthService(repo, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	riskReservation := &registrationRiskReservationStub{blockSignup: true}
	svc.SetRegistrationRiskPolicy(&registrationRiskPolicyStub{reservation: riskReservation})

	_, user, err := svc.Register(WithRegistrationClientMetadata(context.Background(), "203.0.113.80", "Mozilla/5.0"), "blocked@example.com", "password")

	require.ErrorIs(t, err, ErrRegistrationRiskBlocked)
	require.Nil(t, user)
	require.Empty(t, repo.created)
	require.Equal(t, 1, riskReservation.aborts)
}

func TestAuthService_Register_RegistrationRiskCommitFailureLeavesPersistedBalanceAtZero(t *testing.T) {
	repo := &userRepoStub{nextID: 805}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyDefaultSubscriptions: `[{"group_id":11,"validity_days":30}]`,
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	riskReservation := &registrationRiskReservationStub{commitErr: errors.New("store unavailable")}
	svc.SetRegistrationRiskPolicy(&registrationRiskPolicyStub{reservation: riskReservation})

	_, user, err := svc.Register(context.Background(), "commit-failure@example.com", "password")

	require.NoError(t, err, "risk errors do not prevent account creation")
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Zero(t, repo.created[0].Balance, "signup balance must not be inserted before risk commit")
	require.Empty(t, assigner.calls)
	require.Equal(t, 1, riskReservation.aborts)
}

func TestAuthService_Register_RegistrationRiskConcurrentDecisionSuppressesRewards(t *testing.T) {
	repo := &userRepoStub{nextID: 806}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyDefaultSubscriptions: `[{"group_id":11,"validity_days":30}]`,
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	riskReservation := &registrationRiskReservationStub{blockOnCommit: true}
	svc.SetRegistrationRiskPolicy(&registrationRiskPolicyStub{reservation: riskReservation})

	_, user, err := svc.Register(context.Background(), "concurrent@example.com", "password")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Zero(t, repo.created[0].Balance)
	require.Empty(t, assigner.calls)
}

func TestAuthService_Register_RegistrationRiskAbortsWhenUserCreationFails(t *testing.T) {
	repo := &userRepoStub{createErr: errors.New("database unavailable")}
	svc := newAuthService(repo, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	riskReservation := &registrationRiskReservationStub{}
	svc.SetRegistrationRiskPolicy(&registrationRiskPolicyStub{reservation: riskReservation})

	_, _, err := svc.Register(context.Background(), "failed@example.com", "password")

	require.ErrorIs(t, err, ErrServiceUnavailable)
	require.Zero(t, riskReservation.committedIDs)
	require.Equal(t, 1, riskReservation.aborts)
}

func TestAuthService_LoginOrRegisterOAuth_RegistrationRiskSuppressesSignupRewards(t *testing.T) {
	repo := &userRepoStub{nextID: 803}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyDefaultSubscriptions: `[{"group_id":12,"validity_days":14}]`,
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	riskReservation := &registrationRiskReservationStub{rewardBlocked: true}
	risk := &registrationRiskPolicyStub{reservation: riskReservation}
	svc.SetRegistrationRiskPolicy(risk)

	_, user, err := svc.LoginOrRegisterOAuth(
		WithRegistrationClientMetadata(context.Background(), "203.0.113.8", "OAuth Browser"),
		"oauth@example.com",
		"oauth-user",
	)

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Empty(t, assigner.calls)
	require.Equal(t, []int64{803}, riskReservation.committedIDs)
	require.Equal(t, "email", risk.inputs[0].source)
}

func TestAuthService_LoginOrRegisterOAuthWithTokenPair_RegistrationRiskUsesProviderSource(t *testing.T) {
	repo := &userRepoStub{nextID: 804}
	svc := newAuthService(repo, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	svc.refreshTokenCache = &refreshTokenCacheStub{}
	riskReservation := &registrationRiskReservationStub{rewardBlocked: true}
	risk := &registrationRiskPolicyStub{reservation: riskReservation}
	svc.SetRegistrationRiskPolicy(risk)

	pair, user, err := svc.LoginOrRegisterOAuthWithTokenPair(
		WithRegistrationClientMetadata(context.Background(), "203.0.113.9", "OAuth Browser"),
		"provider@example.com",
		"provider-user",
		"",
		"",
		"google",
	)

	require.NoError(t, err)
	require.NotNil(t, pair)
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Equal(t, []int64{804}, riskReservation.committedIDs)
	require.Equal(t, "google", risk.inputs[0].source)
}
