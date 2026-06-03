package service

import "github.com/shopspring/decimal"

// calculateBankMutation 根据交易类型计算账户新快照，数据库写入统一在事务层完成。
func calculateBankMutation(account bankAccountSnapshot, amount decimal.Decimal, txType string) (bankMutation, error) {
	switch txType {
	case BankTxTypeDeposit, BankTxTypeLendProfit:
		return creditBalance(account, amount), nil
	case BankTxTypeLoanBorrow:
		return borrowAgainstCredit(account, amount)
	case BankTxTypeConsume:
		return debitWithCredit(account, amount)
	case BankTxTypeLoanRepay:
		return repayDebt(account, amount)
	case BankTxTypeFreeze, BankTxTypeLendInvest:
		return freezeBalance(account, amount)
	case BankTxTypeUnfreeze:
		return unfreezeBalance(account, amount)
	default:
		return bankMutation{}, ErrBankInvalidType
	}
}

// creditBalance 处理充值和放贷收益：余额增加，负债与冻结金额不变。
func creditBalance(account bankAccountSnapshot, amount decimal.Decimal) bankMutation {
	return bankMutation{
		signedAmount: amount,
		balanceAfter: account.Balance.Add(amount),
		frozenAfter:  account.FrozenAmount,
		debtAfter:    account.TotalDebt,
	}
}

// borrowAgainstCredit 处理贷款入账：余额增加，同时占用信用额度形成负债。
func borrowAgainstCredit(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	debtAfter := account.TotalDebt.Add(amount)
	if debtAfter.GreaterThan(account.CreditLimit) {
		return bankMutation{}, ErrBankCreditLimitExceeded
	}
	return bankMutation{
		signedAmount: amount,
		balanceAfter: account.Balance.Add(amount),
		frozenAfter:  account.FrozenAmount,
		debtAfter:    debtAfter,
	}, nil
}

// debitWithCredit 处理 API 消费：先扣余额，不足部分再占用剩余信用额度。
func debitWithCredit(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	creditAvailable := account.CreditLimit.Sub(account.TotalDebt)
	if creditAvailable.IsNegative() {
		creditAvailable = decimal.Zero
	}
	if account.Balance.Add(creditAvailable).LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	if account.Balance.GreaterThanOrEqual(amount) {
		return bankMutation{
			signedAmount: amount.Neg(),
			balanceAfter: account.Balance.Sub(amount),
			frozenAfter:  account.FrozenAmount,
			debtAfter:    account.TotalDebt,
		}, nil
	}
	deficit := amount.Sub(account.Balance)
	return bankMutation{
		signedAmount: amount.Neg(),
		balanceAfter: decimal.Zero,
		frozenAfter:  account.FrozenAmount,
		debtAfter:    account.TotalDebt.Add(deficit),
	}, nil
}

// repayDebt 处理还款：只能使用可用余额，不能用新的信用额度偿还旧负债。
func repayDebt(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	if account.Balance.LessThan(amount) || account.TotalDebt.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	return bankMutation{
		signedAmount: amount.Neg(),
		balanceAfter: account.Balance.Sub(amount),
		frozenAfter:  account.FrozenAmount,
		debtAfter:    account.TotalDebt.Sub(amount),
	}, nil
}

// freezeBalance 处理冻结与放贷投资：可用余额转入冻结金额，不允许透支冻结。
func freezeBalance(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	if account.Balance.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	return bankMutation{
		signedAmount: amount.Neg(),
		balanceAfter: account.Balance.Sub(amount),
		frozenAfter:  account.FrozenAmount.Add(amount),
		debtAfter:    account.TotalDebt,
	}, nil
}

// unfreezeBalance 处理解冻：冻结金额回到可用余额。
func unfreezeBalance(account bankAccountSnapshot, amount decimal.Decimal) (bankMutation, error) {
	if account.FrozenAmount.LessThan(amount) {
		return bankMutation{}, ErrBankInsufficientFunds
	}
	return bankMutation{
		signedAmount: amount,
		balanceAfter: account.Balance.Add(amount),
		frozenAfter:  account.FrozenAmount.Sub(amount),
		debtAfter:    account.TotalDebt,
	}, nil
}
