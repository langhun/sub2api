package service

import "github.com/shopspring/decimal"

// UpdateUserBalanceProjection 是 Legacy Compatibility Layer 唯一允许写入 User.Balance 的出口。
// User.Balance 仅保留给旧接口、旧页面和旧测试做展示兼容，不能再作为真实资金来源。
func UpdateUserBalanceProjection(user *User, balance float64) {
	if user == nil {
		return
	}
	user.Balance = balance
}

// UpdateUserBalanceProjectionFromDecimal 将高精度账本余额投影到旧展示字段。
func UpdateUserBalanceProjectionFromDecimal(user *User, balance decimal.Decimal) {
	UpdateUserBalanceProjection(user, balance.InexactFloat64())
}

// UpdateUserBalanceProjectionFromTransferResult 使用账本写入结果刷新旧展示字段。
func UpdateUserBalanceProjectionFromTransferResult(user *User, result *TransferFundsResult) {
	if result == nil {
		return
	}
	UpdateUserBalanceProjectionFromDecimal(user, result.Balance)
}

// UpdateUserBalanceProjectionFromBankAccount 使用银行账户权威快照刷新用户展示镜像。
func UpdateUserBalanceProjectionFromBankAccount(user *User, account *BankAccountView) {
	if user == nil {
		return
	}
	user.BankAccount = account
	if account == nil {
		return
	}
	UpdateUserBalanceProjectionFromDecimal(user, account.Balance)
}
