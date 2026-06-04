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
		BankTxTypeLotteryWin,
		BankTxTypeLoanBorrow,
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

func TestBuildBankLedgerPostings_LoanRepaySplitsInterestAndPrincipal(t *testing.T) {
	account := bankTestAccount("10", "0", "20", "5")
	account.ID = 99
	account.UserID = 42
	account.DebtPrincipal = bankDec("5")
	account.DebtInterest = bankDec("2")
	account.TotalDebt = bankDec("7")

	mutation, err := calculateBankMutation(account, bankDec("3"), BankTxTypeLoanRepay)
	require.NoError(t, err)

	postings, err := buildBankLedgerPostings(
		TransferFundsRequest{UserID: account.UserID, Type: BankTxTypeLoanRepay, BusinessModule: BankBusinessModuleLending},
		account,
		mutation,
	)

	require.NoError(t, err)
	require.Len(t, postings, 3)
	require.Equal(t, "USER:42:BALANCE", postings[0].account.code)
	require.Equal(t, bankLedgerSideDebit, postings[0].side)
	requireBankDecimal(t, "3", postings[0].amount)
	require.Equal(t, "PLATFORM:RECEIVABLE:LOAN_INTEREST", postings[1].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[1].side)
	requireBankDecimal(t, "2", postings[1].amount)
	require.Equal(t, "PLATFORM:RECEIVABLE:LOAN_PRINCIPAL", postings[2].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[2].side)
	requireBankDecimal(t, "1", postings[2].amount)
	requireBankLedgerBalanced(t, postings)
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

func TestBuildBankLedgerPostings_TransferRefundDebitsClearing(t *testing.T) {
	account := bankTestAccount("10", "0", "20", "0")
	account.ID = 99
	account.UserID = 42

	postings, err := buildBankLedgerPostings(
		TransferFundsRequest{
			UserID:         account.UserID,
			Type:           BankTxTypeRefund,
			BusinessModule: BankBusinessModuleTransfer,
			Metadata:       map[string]any{"refund_source": "transfer_clearing"},
		},
		account,
		bankMutation{signedAmount: bankDec("2")},
	)

	require.NoError(t, err)
	require.Len(t, postings, 2)
	require.Equal(t, "PLATFORM:CLEARING:TRANSFER", postings[0].account.code)
	require.Equal(t, bankLedgerSideDebit, postings[0].side)
	require.Equal(t, "USER:42:BALANCE", postings[1].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[1].side)
	requireBankLedgerBalanced(t, postings)
}

func TestBuildBankLedgerPostings_TransferFeeRefundDebitsRefundExpense(t *testing.T) {
	account := bankTestAccount("10", "0", "20", "0")
	account.ID = 99
	account.UserID = 42

	postings, err := buildBankLedgerPostings(
		TransferFundsRequest{UserID: account.UserID, Type: BankTxTypeRefund, BusinessModule: BankBusinessModuleTransfer},
		account,
		bankMutation{signedAmount: bankDec("2")},
	)

	require.NoError(t, err)
	require.Len(t, postings, 2)
	require.Equal(t, "PLATFORM:EXPENSE:REFUND", postings[0].account.code)
	require.Equal(t, bankLedgerSideDebit, postings[0].side)
	require.Equal(t, "USER:42:BALANCE", postings[1].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[1].side)
	requireBankLedgerBalanced(t, postings)
}

func TestBuildBankLedgerPostings_LotteryBetSplitsJackpotBurnAndRevenue(t *testing.T) {
	account := bankTestAccount("10", "0", "20", "0")
	account.ID = 99
	account.UserID = 42

	postings, err := buildBankLedgerPostings(
		TransferFundsRequest{
			UserID:         account.UserID,
			Type:           BankTxTypeLotteryBet,
			BusinessModule: BankBusinessModuleGame,
			Metadata: map[string]any{
				"jackpot_amount":  "70",
				"burn_amount":     "20",
				"platform_amount": "10",
			},
		},
		account,
		bankMutation{signedAmount: bankDec("-100")},
	)

	require.NoError(t, err)
	require.Len(t, postings, 4)
	require.Equal(t, "USER:42:BALANCE", postings[0].account.code)
	require.Equal(t, bankLedgerSideDebit, postings[0].side)
	require.Equal(t, "PLATFORM:LIABILITY:LOTTERY_JACKPOT", postings[1].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[1].side)
	require.Equal(t, "PLATFORM:EQUITY:LOTTERY_BURN", postings[2].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[2].side)
	require.Equal(t, "PLATFORM:REVENUE:LOTTERY", postings[3].account.code)
	require.Equal(t, bankLedgerSideCredit, postings[3].side)
	requireBankLedgerBalanced(t, postings)
}

func TestBuildBankLedgerPostings_LotteryWinDebitsJackpot(t *testing.T) {
	account := bankTestAccount("10", "0", "20", "0")
	account.ID = 99
	account.UserID = 42

	postings, err := buildBankLedgerPostings(
		TransferFundsRequest{UserID: account.UserID, Type: BankTxTypeLotteryWin, BusinessModule: BankBusinessModuleGame},
		account,
		bankMutation{signedAmount: bankDec("50")},
	)

	require.NoError(t, err)
	require.Len(t, postings, 2)
	require.Equal(t, "PLATFORM:LIABILITY:LOTTERY_JACKPOT", postings[0].account.code)
	require.Equal(t, bankLedgerSideDebit, postings[0].side)
	require.Equal(t, "USER:42:BALANCE", postings[1].account.code)
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
