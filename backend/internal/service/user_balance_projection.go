package service

import "github.com/shopspring/decimal"

// setLegacyBalanceProjection 仅同步旧 User.Balance 展示镜像。
// 真实资金来源始终以银行账户快照和账本流水为准。
func setLegacyBalanceProjection(user *User, balance float64) {
	if user == nil {
		return
	}
	user.Balance = balance
}

// setLegacyBalanceProjectionFromDecimal 将高精度账本余额投影到旧展示字段。
func setLegacyBalanceProjectionFromDecimal(user *User, balance decimal.Decimal) {
	setLegacyBalanceProjection(user, balance.InexactFloat64())
}

// applyLegacyBalanceProjectionFromTransferResult 使用账本写入结果刷新旧展示字段。
func applyLegacyBalanceProjectionFromTransferResult(user *User, result *TransferFundsResult) {
	if result == nil {
		return
	}
	setLegacyBalanceProjectionFromDecimal(user, result.Balance)
}

// applyBankAccountProjectionToUser 使用银行账户权威快照刷新用户展示镜像。
func applyBankAccountProjectionToUser(user *User, account *BankAccountView) {
	if user == nil {
		return
	}
	user.BankAccount = account
	if account == nil {
		return
	}
	setLegacyBalanceProjectionFromDecimal(user, account.Balance)
}
