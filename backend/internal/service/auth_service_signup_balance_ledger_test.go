//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type signupBalanceLedgerStub struct {
	records   []signupBalanceLedgerRecord
	recordErr error
	deletes   []signupBalanceLedgerRecord
}

type signupBalanceLedgerRecord struct {
	userID int64
	amount float64
	source string
}

func (s *signupBalanceLedgerStub) RecordSignupBalance(_ context.Context, userID int64, amount float64, source string) error {
	if s.recordErr != nil {
		return s.recordErr
	}
	s.records = append(s.records, signupBalanceLedgerRecord{userID: userID, amount: amount, source: source})
	return nil
}

func (s *signupBalanceLedgerStub) DeleteSignupBalance(_ context.Context, userID int64, amount float64, source string) error {
	s.deletes = append(s.deletes, signupBalanceLedgerRecord{userID: userID, amount: amount, source: source})
	return nil
}

func TestAuthServiceRegisterRecordsSignupBalanceLedger(t *testing.T) {
	repo := &userRepoStub{nextID: 905}
	svc := newAuthService(repo, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	ledger := &signupBalanceLedgerStub{}
	svc.SetSignupBalanceLedger(ledger)

	_, user, err := svc.Register(context.Background(), "ledger@example.com", "password")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.InDelta(t, 3.5, user.Balance, 0.00001)
	require.Equal(t, []signupBalanceLedgerRecord{{userID: 905, amount: 3.5, source: "email"}}, ledger.records)
	require.Empty(t, ledger.deletes)
}

func TestAuthServiceRegisterSuppressesBalanceWhenSignupLedgerFails(t *testing.T) {
	repo := &userRepoStub{nextID: 906}
	svc := newAuthService(repo, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	ledger := &signupBalanceLedgerStub{recordErr: errors.New("ledger unavailable")}
	svc.SetSignupBalanceLedger(ledger)

	_, user, err := svc.Register(context.Background(), "ledger-failure@example.com", "password")

	require.NoError(t, err, "audit storage failure must not prevent account creation")
	require.NotNil(t, user)
	require.Zero(t, user.Balance)
	require.Zero(t, repo.created[0].Balance)
	require.Empty(t, ledger.records)
}
