package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBankLedgerPostings_AllSupportedTypesAreBalanced(t *testing.T) {
	account := bankTestAccount("10", "3", "20", "0")
	account.ID = 99
	account.UserID = 42
	txTypes := []string{
		BankTxTypeConsume,
		BankTxTypeDeposit,
		BankTxTypeWithdraw,
		BankTxTypeTransferOut,
		BankTxTypeTransferIn,
		BankTxTypeSlotBet,
		BankTxTypeSlotWin,
		BankTxTypeLoanBorrow,
		BankTxTypeLoanRepay,
		BankTxTypeLoanInterest,
		BankTxTypeLendInvest,
		BankTxTypeLendProfit,
		BankTxTypeReward,
		BankTxTypeRefund,
		BankTxTypeFreeze,
		BankTxTypeUnfreeze,
	}

	for _, txType := range txTypes {
		t.Run(txType, func(t *testing.T) {
			postings, err := buildBankLedgerPostings(
				TransferFundsRequest{UserID: account.UserID, Type: txType, BusinessModule: BankBusinessModuleFinancialHub},
				account,
				bankMutation{signedAmount: bankDec("2")},
			)

			require.NoError(t, err)
			require.Len(t, postings, 2)
			requireBankLedgerBalanced(t, postings)
		})
	}
}

func TestBuildBankLedgerPostings_ConsumeDebitsUserAndCreditsRevenue(t *testing.T) {
	account := bankTestAccount("10", "0", "20", "0")
	account.ID = 99
	account.UserID = 42

	postings, err := buildBankLedgerPostings(
		TransferFundsRequest{UserID: account.UserID, Type: BankTxTypeConsume, BusinessModule: BankBusinessModuleAPIGateway},
		account,
		bankMutation{signedAmount: bankDec("-2")},
	)

	require.NoError(t, err)
	require.Len(t, postings, 2)
	require.Equal(t, "USER:42:BALANCE", postings[0].account.code)
	require.Equal(t, bankLedgerSideDebit, postings[0].side)
	require.Equal(t, "PLATFORM:REVENUE:API", postings[1].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[1].side)
	requireBankLedgerBalanced(t, postings)
}

func requireBankLedgerBalanced(t *testing.T, postings []bankLedgerPosting) {
	t.Helper()
	debitTotal := bankDec("0")
	creditTotal := bankDec("0")
	for _, posting := range postings {
		require.True(t, posting.amount.GreaterThan(bankDec("0")), "posting amount must be positive")
		switch posting.side {
		case bankLedgerSideDebit:
			debitTotal = debitTotal.Add(posting.amount)
		case bankLedgerSideCredit:
			creditTotal = creditTotal.Add(posting.amount)
		default:
			t.Fatalf("unexpected ledger side %q", posting.side)
		}
	}
	require.True(t, debitTotal.Equal(creditTotal), "debit=%s credit=%s", debitTotal, creditTotal)
}
