package service

import (
	"errors"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestCalculateBankMutation_ConsumeUsesBalanceFirst(t *testing.T) {
	account := bankTestAccount("10", "0", "5", "1")
	got, err := calculateBankMutation(account, bankDec("4"), BankTxTypeConsume)

	require.NoError(t, err)
	requireBankDecimal(t, "-4", got.signedAmount)
	requireBankDecimal(t, "6", got.balanceAfter)
	requireBankDecimal(t, "0", got.frozenAfter)
	requireBankDecimal(t, "1", got.debtAfter)
}

func TestCalculateBankMutation_ConsumeUsesRemainingCredit(t *testing.T) {
	account := bankTestAccount("3", "0", "10", "4")
	got, err := calculateBankMutation(account, bankDec("7"), BankTxTypeConsume)

	require.NoError(t, err)
	requireBankDecimal(t, "-7", got.signedAmount)
	requireBankDecimal(t, "-4", got.balanceAfter)
	requireBankDecimal(t, "0", got.frozenAfter)
	requireBankDecimal(t, "8", got.debtAfter)
	requireBankDecimal(t, "8", got.debtPrincipalAfter)
}

func TestCalculateBankMutation_DepositRepaysOverdraftPrincipal(t *testing.T) {
	account := bankTestAccount("-4", "0", "10", "4")
	got, err := calculateBankMutation(account, bankDec("6"), BankTxTypeDeposit)

	require.NoError(t, err)
	requireBankDecimal(t, "6", got.signedAmount)
	requireBankDecimal(t, "2", got.balanceAfter)
	requireBankDecimal(t, "0", got.debtPrincipalAfter)
	requireBankDecimal(t, "0", got.debtAfter)
}

func TestCalculateBankMutation_ConsumeRejectsInsufficientFunds(t *testing.T) {
	account := bankTestAccount("3", "0", "10", "9")
	_, err := calculateBankMutation(account, bankDec("5"), BankTxTypeConsume)

	require.ErrorIs(t, err, ErrBankInsufficientFunds)
}

func TestCalculateBankMutation_SlotBetRejectsCreditUsage(t *testing.T) {
	account := bankTestAccount("3", "0", "10", "0")
	_, err := calculateBankMutation(account, bankDec("5"), BankTxTypeSlotBet)

	require.ErrorIs(t, err, ErrBankInsufficientFunds)
}

func TestCalculateBankMutation_FreezeAndUnfreeze(t *testing.T) {
	account := bankTestAccount("6", "1", "0", "0")
	frozen, err := calculateBankMutation(account, bankDec("4"), BankTxTypeFreeze)

	require.NoError(t, err)
	requireBankDecimal(t, "-4", frozen.signedAmount)
	requireBankDecimal(t, "2", frozen.balanceAfter)
	requireBankDecimal(t, "5", frozen.frozenAfter)

	account.Balance = frozen.balanceAfter
	account.FrozenAmount = frozen.frozenAfter
	unfrozen, err := calculateBankMutation(account, bankDec("3"), BankTxTypeUnfreeze)

	require.NoError(t, err)
	requireBankDecimal(t, "3", unfrozen.signedAmount)
	requireBankDecimal(t, "5", unfrozen.balanceAfter)
	requireBankDecimal(t, "2", unfrozen.frozenAfter)
}

func TestCalculateBankMutation_LoanBorrowAndRepay(t *testing.T) {
	account := bankTestAccount("2", "0", "10", "1")
	borrowed, err := calculateBankMutation(account, bankDec("4"), BankTxTypeLoanBorrow)

	require.NoError(t, err)
	requireBankDecimal(t, "4", borrowed.signedAmount)
	requireBankDecimal(t, "6", borrowed.balanceAfter)
	requireBankDecimal(t, "5", borrowed.debtAfter)

	account.Balance = borrowed.balanceAfter
	account.DebtPrincipal = borrowed.debtPrincipalAfter
	account.TotalDebt = borrowed.debtAfter
	repaid, err := calculateBankMutation(account, bankDec("3"), BankTxTypeLoanRepay)

	require.NoError(t, err)
	requireBankDecimal(t, "-3", repaid.signedAmount)
	requireBankDecimal(t, "3", repaid.balanceAfter)
	requireBankDecimal(t, "2", repaid.debtAfter)
}

func TestCalculateBankMutation_LoanInterestIncreasesDebtOnly(t *testing.T) {
	account := bankTestAccount("2", "0", "10", "1")
	got, err := calculateBankMutation(account, bankDec("0.5"), BankTxTypeLoanInterest)

	require.NoError(t, err)
	requireBankDecimal(t, "-0.5", got.signedAmount)
	requireBankDecimal(t, "2", got.balanceAfter)
	requireBankDecimal(t, "1", got.debtPrincipalAfter)
	requireBankDecimal(t, "0.5", got.debtInterestAfter)
	requireBankDecimal(t, "1.5", got.debtAfter)
}

func TestCalculateBankMutation_LoanBorrowRejectsCreditLimitExceeded(t *testing.T) {
	account := bankTestAccount("0", "0", "5", "4")
	_, err := calculateBankMutation(account, bankDec("2"), BankTxTypeLoanBorrow)

	require.ErrorIs(t, err, ErrBankCreditLimitExceeded)
}

func TestNormalizeTransferFundsRequest_RequiresIdempotencyKey(t *testing.T) {
	_, err := normalizeTransferFundsRequest(TransferFundsRequest{
		UserID: 1,
		Amount: bankDec("1"),
		Type:   BankTxTypeDeposit,
	})

	require.ErrorIs(t, err, ErrBankIdempotencyKeyRequired)
}

func TestNormalizeTransferFundsRequest_DefaultsScopeAndRoundsAmount(t *testing.T) {
	got, err := normalizeTransferFundsRequest(TransferFundsRequest{
		UserID:         42,
		Amount:         bankDec("1.1234567891234567894"),
		Type:           " consume ",
		IdempotencyKey: "bank-test-key",
	})

	require.NoError(t, err)
	require.Equal(t, "CONSUME", got.Type)
	require.Equal(t, "bank:transfer:user:42", got.IdempotencyScope)
	requireBankDecimal(t, "1.123456789123456789", got.Amount)
	require.NotNil(t, got.Metadata)
}

func TestNormalizeTransferFundsRequest_RejectsLongScope(t *testing.T) {
	_, err := normalizeTransferFundsRequest(TransferFundsRequest{
		UserID:           1,
		Amount:           bankDec("1"),
		Type:             BankTxTypeDeposit,
		IdempotencyKey:   "bank-test-key",
		IdempotencyScope: string(make([]byte, 129)),
	})

	require.True(t, errors.Is(err, ErrBankIdempotencyScopeInvalid))
}

func TestBankTransactionMatchesRequestRejectsDifferentPayload(t *testing.T) {
	log := &dbent.TransactionLog{
		UserID:      1,
		TxType:      BankTxTypeConsume,
		Amount:      bankDec("-1"),
		Description: "consume",
	}
	req := TransferFundsRequest{
		UserID:      1,
		Amount:      bankDec("2"),
		Type:        BankTxTypeConsume,
		Description: "consume",
	}

	require.False(t, bankTransactionMatchesRequest(log, req))
}

func TestBankTransactionMatchesRequestAcceptsSameSignedPayload(t *testing.T) {
	requestID := "request-1"
	log := &dbent.TransactionLog{
		UserID:      1,
		TxType:      BankTxTypeConsume,
		Amount:      bankDec("-2"),
		Description: "consume",
		RequestID:   &requestID,
	}
	req := TransferFundsRequest{
		UserID:      1,
		Amount:      bankDec("2"),
		Type:        BankTxTypeConsume,
		Description: "consume",
		RequestID:   requestID,
	}

	require.True(t, bankTransactionMatchesRequest(log, req))
}

func bankTestAccount(balance, frozen, creditLimit, debt string) bankAccountSnapshot {
	return bankAccountSnapshot{
		Balance:       bankDec(balance),
		FrozenAmount:  bankDec(frozen),
		CreditLimit:   bankDec(creditLimit),
		DebtPrincipal: bankDec(debt),
		TotalDebt:     bankDec(debt),
		Status:        BankAccountStatusActive,
	}
}

func bankDec(raw string) decimal.Decimal {
	return decimal.RequireFromString(raw)
}

func requireBankDecimal(t *testing.T, want string, got decimal.Decimal) {
	t.Helper()
	require.True(t, bankDec(want).Equal(got), "want %s, got %s", want, got.String())
}
