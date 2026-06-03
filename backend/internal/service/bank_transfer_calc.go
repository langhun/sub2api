package service

import "github.com/shopspring/decimal"

type bankMutationState struct {
	signedAmount   decimal.Decimal
	balanceAfter   decimal.Decimal
	frozenAfter    decimal.Decimal
	principalAfter decimal.Decimal
	interestAfter  decimal.Decimal
}

// calculateBankMutation 根据交易类型计算账户新快照，数据库写入统一在事务层完成。
func calculateBankMutation(account bankAccountSnapshot, amount decimal.Decimal, txType string) (bankMutation, error) {
	switch txType {
	case BankTxTypeDeposit, BankTxTypeTransferIn, BankTxTypeSlotWin,
		BankTxTypeLendProfit, BankTxTypeReward, BankTxTypeRefund:
		return creditBalance(account, amount), nil
	case BankTxTypeLoanBorrow:
		return borrowAgainstCredit(account, amount)
	case BankTxTypeConsume, BankTxTypeWithdraw, BankTxTypeTransferOut:
		return debitWithCredit(account, amount)
	case BankTxTypeSlotBet:
		return debitCashOnly(account, amount)
	case BankTxTypeLoanRepay:
		return repayDebt(account, amount)
	case BankTxTypeLoanInterest:
		return accrueLoanInterest(account, amount)
	case BankTxTypeFreeze, BankTxTypeLendInvest:
		return freezeBalance(account, amount)
	case BankTxTypeUnfreeze:
		return unfreezeBalance(account, amount)
	default:
		return bankMutation{}, ErrBankInvalidType
	}
}

// creditBalance 处理入账类交易；如果账户为负余额，则同步减少对应的透支本金。
func creditBalance(account bankAccountSnapshot, amount decimal.Decimal) bankMutation {
	balanceAfter := account.Balance.Add(amount)
	principalAfter, interestAfter := applyOverdraftDebt(account, balanceAfter)
	return newBankMutation(bankMutationState{
		signedAmount:   amount,
		balanceAfter:   balanceAfter,
		frozenAfter:    account.FrozenAmount,
		principalAfter: principalAfter,
		interestAfter:  interestAfter,
	})
}

// borrowAgainstCredit 处理贷款入账：余额增加，同时把贷款本金纳入授信占用。
func borrowAgainstCredit(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	balanceAfter := account.Balance.Add(amount)
	principalAfter, interestAfter := applyOverdraftDebt(account, balanceAfter)
	principalAfter = principalAfter.Add(amount)
	if principalAfter.Add(interestAfter).GreaterThan(account.CreditLimit) {
		return bankMutation{}, ErrBankCreditLimitExceeded
	}
	return newBankMutation(bankMutationState{
		signedAmount:   amount,
		balanceAfter:   balanceAfter,
		frozenAfter:    account.FrozenAmount,
		principalAfter: principalAfter,
		interestAfter:  interestAfter,
	}), nil
}

// debitWithCredit 处理扣费类交易：允许在授信额度内把可用余额扣成负数。
func debitWithCredit(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	balanceAfter := account.Balance.Sub(amount)
	principalAfter, interestAfter := applyOverdraftDebt(account, balanceAfter)
	if principalAfter.Add(interestAfter).GreaterThan(account.CreditLimit) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	return newBankMutation(bankMutationState{
		signedAmount:   amount.Neg(),
		balanceAfter:   balanceAfter,
		frozenAfter:    account.FrozenAmount,
		principalAfter: principalAfter,
		interestAfter:  interestAfter,
	}), nil
}

// debitCashOnly 处理不能使用授信的扣款场景，例如游戏下注。
func debitCashOnly(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	if account.Balance.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	principal, interest, _ := bankDebtParts(account)
	return newBankMutation(bankMutationState{
		signedAmount:   amount.Neg(),
		balanceAfter:   account.Balance.Sub(amount),
		frozenAfter:    account.FrozenAmount,
		principalAfter: principal,
		interestAfter:  interest,
	}), nil
}

// repayDebt 处理还款：只能使用正余额偿还旧债，并按先息后本顺序减少债务。
func repayDebt(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	principal, interest, totalDebt := bankDebtParts(account)
	if account.Balance.LessThan(amount) || totalDebt.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	interestPaid := minBankDecimal(amount, interest)
	principalPaid := amount.Sub(interestPaid)
	principalAfter := principal.Sub(principalPaid)
	interestAfter := interest.Sub(interestPaid)
	return newBankMutation(bankMutationState{
		signedAmount:   amount.Neg(),
		balanceAfter:   account.Balance.Sub(amount),
		frozenAfter:    account.FrozenAmount,
		principalAfter: principalAfter,
		interestAfter:  interestAfter,
	}), nil
}

// accrueLoanInterest 处理计息：不改变可用余额，只增加待还利息。
func accrueLoanInterest(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	principal, interest, _ := bankDebtParts(account)
	interestAfter := interest.Add(amount)
	if principal.Add(interestAfter).GreaterThan(account.CreditLimit) {
		return bankMutation{}, ErrBankCreditLimitExceeded
	}
	return newBankMutation(bankMutationState{
		signedAmount:   amount.Neg(),
		balanceAfter:   account.Balance,
		frozenAfter:    account.FrozenAmount,
		principalAfter: principal,
		interestAfter:  interestAfter,
	}), nil
}

// freezeBalance 处理冻结与放贷投资：可用余额转入冻结金额，不允许使用授信冻结。
func freezeBalance(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	if account.Balance.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	principal, interest, _ := bankDebtParts(account)
	return newBankMutation(bankMutationState{
		signedAmount:   amount.Neg(),
		balanceAfter:   account.Balance.Sub(amount),
		frozenAfter:    account.FrozenAmount.Add(amount),
		principalAfter: principal,
		interestAfter:  interest,
	}), nil
}

// unfreezeBalance 处理解冻：冻结金额回到可用余额。
func unfreezeBalance(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	if account.FrozenAmount.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	principal, interest, _ := bankDebtParts(account)
	return newBankMutation(bankMutationState{
		signedAmount:   amount,
		balanceAfter:   account.Balance.Add(amount),
		frozenAfter:    account.FrozenAmount.Sub(amount),
		principalAfter: principal,
		interestAfter:  interest,
	}), nil
}

// applyOverdraftDebt 根据负余额变化调整透支本金，保留贷款本金和利息不变。
func applyOverdraftDebt(account bankAccountSnapshot, balanceAfter decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	principal, interest, _ := bankDebtParts(account)
	beforeOverdraft := bankOverdraftAmount(account.Balance)
	afterOverdraft := bankOverdraftAmount(balanceAfter)
	principal = principal.Sub(beforeOverdraft).Add(afterOverdraft)
	if principal.IsNegative() {
		principal = decimal.Zero
	}
	return principal, interest
}

// bankDebtParts 读取当前债务拆分；旧数据只有 total_debt 时默认按本金处理。
func bankDebtParts(account bankAccountSnapshot) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	principal := account.DebtPrincipal
	interest := account.DebtInterest
	total := effectiveBankDebt(principal, interest, account.TotalDebt)
	if principal.Add(interest).IsZero() && account.TotalDebt.GreaterThan(decimal.Zero) {
		principal = account.TotalDebt
		total = account.TotalDebt
	}
	return principal, interest, total
}

// bankOverdraftAmount 将负余额转换为正数透支额。
func bankOverdraftAmount(balance decimal.Decimal) decimal.Decimal {
	if balance.IsNegative() {
		return balance.Neg()
	}
	return decimal.Zero
}

// newBankMutation 统一收口快照字段，确保 total_debt 始终等于本金加利息。
func newBankMutation(state bankMutationState) bankMutation {
	return bankMutation{
		signedAmount:       state.signedAmount,
		balanceAfter:       state.balanceAfter,
		frozenAfter:        state.frozenAfter,
		debtPrincipalAfter: state.principalAfter,
		debtInterestAfter:  state.interestAfter,
		debtAfter:          state.principalAfter.Add(state.interestAfter),
	}
}

// minBankDecimal 返回两个金额中的较小值。
func minBankDecimal(left decimal.Decimal, right decimal.Decimal) decimal.Decimal {
	if left.LessThan(right) {
		return left
	}
	return right
}
