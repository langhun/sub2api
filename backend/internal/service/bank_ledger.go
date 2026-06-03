package service

import (
	"context"
	"database/sql"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

const (
	bankLedgerSideDebit  = "DEBIT"
	bankLedgerSideCredit = "CREDIT"

	bankLedgerOwnerPlatform = "PLATFORM"
	bankLedgerOwnerUser     = "USER"

	bankLedgerTypeAsset     = "ASSET"
	bankLedgerTypeLiability = "LIABILITY"
	bankLedgerTypeRevenue   = "REVENUE"
	bankLedgerTypeExpense   = "EXPENSE"
)

type bankLedgerAccountSpec struct {
	code             string
	name             string
	accountType      string
	normalBalance    string
	ownerType        string
	ownerUserID      int64
	hasOwnerUserID   bool
	bankAccountID    int64
	hasBankAccountID bool
}

type bankLedgerPosting struct {
	account   bankLedgerAccountSpec
	side      string
	amount    decimal.Decimal
	userID    int64
	hasUserID bool
}

// createBankLedgerEntries 为一条用户流水写入完整借贷分录，任一失败都会由外层事务回滚。
func createBankLedgerEntries(
	ctx context.Context,
	client *dbent.Client,
	req TransferFundsRequest,
	account bankAccountSnapshot,
	mutation bankMutation,
	log *dbent.TransactionLog,
) error {
	postings, err := buildBankLedgerPostings(req, account, mutation)
	if err != nil {
		return err
	}
	for _, posting := range postings {
		ledgerAccountID, err := ensureBankLedgerAccount(ctx, client, posting.account)
		if err != nil {
			return err
		}
		create := client.LedgerEntry.Create().
			SetTransactionLogID(log.ID).
			SetTxID(log.TxID).
			SetLedgerAccountID(ledgerAccountID).
			SetEntrySide(posting.side).
			SetAmount(posting.amount).
			SetBusinessModule(req.BusinessModule).
			SetTxType(req.Type).
			SetDescription(req.Description).
			SetMetadata(req.Metadata)
		if posting.hasUserID {
			create.SetUserID(posting.userID)
		}
		if req.ReferenceType != "" {
			create.SetReferenceType(req.ReferenceType)
		}
		if req.ReferenceID != "" {
			create.SetReferenceID(req.ReferenceID)
		}
		if _, err := create.Save(ctx); err != nil {
			return fmt.Errorf("create bank ledger entry: %w", err)
		}
	}
	return nil
}

// buildBankLedgerPostings 将业务流水映射为双重记账分录；每个分支都必须借贷平衡。
func buildBankLedgerPostings(
	req TransferFundsRequest,
	account bankAccountSnapshot,
	mutation bankMutation,
) ([]bankLedgerPosting, error) {
	amount := mutation.signedAmount.Abs()
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrBankInvalidAmount
	}
	available := userAvailableLedgerAccount(account)
	frozen := userFrozenLedgerAccount(account)
	switch req.Type {
	case BankTxTypeConsume:
		return bankDebitCredit(available, bankPlatformAPIRevenue(), amount, account.UserID), nil
	case BankTxTypeDeposit:
		return bankDebitCredit(bankPlatformCash(), available, amount, account.UserID), nil
	case BankTxTypeWithdraw:
		return bankDebitCredit(available, bankPlatformCash(), amount, account.UserID), nil
	case BankTxTypeTransferOut:
		return bankDebitCredit(available, bankPlatformTransferClearing(), amount, account.UserID), nil
	case BankTxTypeTransferIn:
		return bankDebitCredit(bankPlatformTransferClearing(), available, amount, account.UserID), nil
	case BankTxTypeSlotBet:
		return bankDebitCredit(available, bankPlatformGameRevenue(), amount, account.UserID), nil
	case BankTxTypeSlotWin:
		return bankDebitCredit(bankPlatformGameExpense(), available, amount, account.UserID), nil
	case BankTxTypeLoanBorrow:
		return bankDebitCredit(bankPlatformLoanReceivable(), available, amount, account.UserID), nil
	case BankTxTypeLoanRepay:
		return bankDebitCredit(available, bankPlatformLoanReceivable(), amount, account.UserID), nil
	case BankTxTypeLoanInterest:
		return bankDebitCredit(bankPlatformInterestReceivable(), bankPlatformInterestRevenue(), amount, 0), nil
	case BankTxTypeLendInvest, BankTxTypeFreeze:
		return bankDebitCredit(available, frozen, amount, account.UserID), nil
	case BankTxTypeLendProfit:
		return bankDebitCredit(bankPlatformLendingProfitExpense(), available, amount, account.UserID), nil
	case BankTxTypeReward:
		return bankDebitCredit(bankPlatformRewardExpense(), available, amount, account.UserID), nil
	case BankTxTypeRefund:
		if req.BusinessModule == BankBusinessModuleTransfer {
			if req.Metadata != nil && req.Metadata["refund_source"] == "transfer_clearing" {
				return bankDebitCredit(bankPlatformTransferClearing(), available, amount, account.UserID), nil
			}
		}
		return bankDebitCredit(bankPlatformRefundExpense(), available, amount, account.UserID), nil
	case BankTxTypeUnfreeze:
		return bankDebitCredit(frozen, available, amount, account.UserID), nil
	default:
		return nil, ErrBankInvalidType
	}
}

// bankDebitCredit 生成一借一贷两条分录，金额相同、方向相反。
func bankDebitCredit(
	debitAccount bankLedgerAccountSpec,
	creditAccount bankLedgerAccountSpec,
	amount decimal.Decimal,
	userID int64,
) []bankLedgerPosting {
	return []bankLedgerPosting{
		bankLedgerPostingFor(debitAccount, bankLedgerSideDebit, amount, userID),
		bankLedgerPostingFor(creditAccount, bankLedgerSideCredit, amount, userID),
	}
}

func bankLedgerPostingFor(
	account bankLedgerAccountSpec,
	side string,
	amount decimal.Decimal,
	userID int64,
) bankLedgerPosting {
	posting := bankLedgerPosting{account: account, side: side, amount: amount}
	if account.ownerType == bankLedgerOwnerUser && userID > 0 {
		posting.userID = userID
		posting.hasUserID = true
	}
	return posting
}

// ensureBankLedgerAccount 并发安全地创建或复用总账科目，并返回科目主键。
func ensureBankLedgerAccount(ctx context.Context, client *dbent.Client, spec bankLedgerAccountSpec) (int64, error) {
	rows, err := client.QueryContext(ctx, `
INSERT INTO ledger_accounts (
    account_code,
    account_name,
    account_type,
    normal_balance,
    owner_type,
    owner_user_id,
    user_bank_account_id,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
ON CONFLICT (account_code) DO UPDATE SET
    account_name = EXCLUDED.account_name,
    account_type = EXCLUDED.account_type,
    normal_balance = EXCLUDED.normal_balance,
    owner_type = EXCLUDED.owner_type,
    owner_user_id = EXCLUDED.owner_user_id,
    user_bank_account_id = EXCLUDED.user_bank_account_id,
    updated_at = NOW()
RETURNING id
`,
		spec.code,
		spec.name,
		spec.accountType,
		spec.normalBalance,
		spec.ownerType,
		nullableBankInt64(spec.ownerUserID, spec.hasOwnerUserID),
		nullableBankInt64(spec.bankAccountID, spec.hasBankAccountID),
	)
	if err != nil {
		return 0, fmt.Errorf("ensure bank ledger account: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if rowsErr := rows.Err(); rowsErr != nil {
			return 0, fmt.Errorf("scan bank ledger account: %w", rowsErr)
		}
		return 0, sql.ErrNoRows
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return 0, fmt.Errorf("scan bank ledger account id: %w", err)
	}
	return id, rows.Err()
}

func nullableBankInt64(value int64, valid bool) any {
	if !valid {
		return nil
	}
	return value
}

func userAvailableLedgerAccount(account bankAccountSnapshot) bankLedgerAccountSpec {
	return bankUserLedgerAccount(account, "BALANCE", "Available Balance")
}

func userFrozenLedgerAccount(account bankAccountSnapshot) bankLedgerAccountSpec {
	return bankUserLedgerAccount(account, "FROZEN", "Frozen Balance")
}

func bankUserLedgerAccount(account bankAccountSnapshot, suffix string, name string) bankLedgerAccountSpec {
	return bankLedgerAccountSpec{
		code:             fmt.Sprintf("USER:%d:%s", account.UserID, suffix),
		name:             fmt.Sprintf("User %d %s", account.UserID, name),
		accountType:      bankLedgerTypeLiability,
		normalBalance:    bankLedgerSideCredit,
		ownerType:        bankLedgerOwnerUser,
		ownerUserID:      account.UserID,
		hasOwnerUserID:   true,
		bankAccountID:    account.ID,
		hasBankAccountID: true,
	}
}

func bankPlatformLedgerAccount(code, name, accountType, normalBalance string) bankLedgerAccountSpec {
	return bankLedgerAccountSpec{
		code:          code,
		name:          name,
		accountType:   accountType,
		normalBalance: normalBalance,
		ownerType:     bankLedgerOwnerPlatform,
	}
}

func bankPlatformCash() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:CASH", "Platform Cash", bankLedgerTypeAsset, bankLedgerSideDebit)
}

func bankPlatformTransferClearing() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:CLEARING:TRANSFER", "Platform Transfer Clearing", bankLedgerTypeLiability, bankLedgerSideCredit)
}

func bankPlatformLoanReceivable() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:RECEIVABLE:LOAN_PRINCIPAL", "Platform Loan Principal Receivable", bankLedgerTypeAsset, bankLedgerSideDebit)
}

func bankPlatformInterestReceivable() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:RECEIVABLE:LOAN_INTEREST", "Platform Loan Interest Receivable", bankLedgerTypeAsset, bankLedgerSideDebit)
}

func bankPlatformAPIRevenue() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:REVENUE:API", "Platform API Revenue", bankLedgerTypeRevenue, bankLedgerSideCredit)
}

func bankPlatformGameRevenue() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:REVENUE:GAME", "Platform Game Revenue", bankLedgerTypeRevenue, bankLedgerSideCredit)
}

func bankPlatformInterestRevenue() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:REVENUE:INTEREST", "Platform Interest Revenue", bankLedgerTypeRevenue, bankLedgerSideCredit)
}

func bankPlatformGameExpense() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:EXPENSE:GAME_PAYOUT", "Platform Game Payout Expense", bankLedgerTypeExpense, bankLedgerSideDebit)
}

func bankPlatformLendingProfitExpense() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:EXPENSE:LENDING_PROFIT", "Platform Lending Profit Expense", bankLedgerTypeExpense, bankLedgerSideDebit)
}

func bankPlatformRewardExpense() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:EXPENSE:REWARD", "Platform Reward Expense", bankLedgerTypeExpense, bankLedgerSideDebit)
}

func bankPlatformRefundExpense() bankLedgerAccountSpec {
	return bankPlatformLedgerAccount("PLATFORM:EXPENSE:REFUND", "Platform Refund Expense", bankLedgerTypeExpense, bankLedgerSideDebit)
}
