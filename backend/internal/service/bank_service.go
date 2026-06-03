package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// BankService 是虚拟银行的唯一资金变动服务，后续扣费、贷款、放贷都应从这里进入。
type BankService struct {
	client *dbent.Client
}

// NewBankService 创建账本服务实例，暂不接入 wire，后续 API/middleware 阶段再统一注入。
func NewBankService(client *dbent.Client) *BankService {
	return &BankService{client: client}
}

// TransferFunds 在一个 Serializable 事务中完成账户更新与流水写入。
func (s *BankService) TransferFunds(ctx context.Context, req TransferFundsRequest) (*TransferFundsResult, error) {
	if s == nil || s.client == nil {
		return nil, ErrBankClientUnavailable
	}
	normalized, err := normalizeTransferFundsRequest(req)
	if err != nil {
		return nil, err
	}
	return s.runSerializableBankTx(ctx, func(txClient *dbent.Client) (*TransferFundsResult, error) {
		return s.transferFundsInTx(ctx, txClient, normalized)
	})
}

// transferFunds 保留 PRD 指定的核心函数形态；正式业务应优先调用带显式幂等键的 TransferFunds。
func (s *BankService) transferFunds(
	ctx context.Context,
	userID int64,
	amount decimal.Decimal,
	txType string,
	description string,
) (*TransferFundsResult, error) {
	key := fmt.Sprintf("legacy:%d:%s:%s:%s", userID, strings.ToUpper(strings.TrimSpace(txType)), amount.String(), description)
	return s.TransferFunds(ctx, TransferFundsRequest{
		UserID:         userID,
		Amount:         amount,
		Type:           txType,
		Description:    description,
		IdempotencyKey: key,
	})
}

// runSerializableBankTx 用 defer/recover 模拟 try-catch，确保错误和 panic 都触发回滚。
func (s *BankService) runSerializableBankTx(
	ctx context.Context,
	fn func(*dbent.Client) (*TransferFundsResult, error),
) (result *TransferFundsResult, err error) {
	var lastErr error
	for attempt := 0; attempt < bankTxMaxRetries; attempt++ {
		tx, beginErr := s.client.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if beginErr != nil {
			return nil, fmt.Errorf("begin bank transaction: %w", beginErr)
		}
		committed := false
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					_ = tx.Rollback()
					err = infraerrors.InternalServer("BANK_TRANSACTION_PANIC", "bank transaction panicked").
						WithCause(fmt.Errorf("%v", recovered))
					return
				}
				if !committed && err != nil {
					_ = tx.Rollback()
				}
			}()
			result, err = fn(tx.Client())
			if err != nil {
				return
			}
			if commitErr := tx.Commit(); commitErr != nil {
				err = fmt.Errorf("commit bank transaction: %w", commitErr)
				return
			}
			committed = true
		}()
		if err == nil {
			return result, nil
		}
		if isRetryableBankTxErr(err) {
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

// transferFundsInTx 是单次账本操作的事务体：幂等检查、行锁、计算、更新、写流水。
func (s *BankService) transferFundsInTx(
	ctx context.Context,
	client *dbent.Client,
	req TransferFundsRequest,
) (*TransferFundsResult, error) {
	if existing, err := findBankTransaction(ctx, client, req); err != nil || existing != nil {
		return existing, err
	}
	account, err := lockBankAccountForUpdate(ctx, client, req.UserID)
	if err != nil {
		return nil, err
	}
	if account.Status != bankAccountStatusActive {
		return nil, ErrBankAccountNotActive
	}
	if existing, err := findBankTransaction(ctx, client, req); err != nil || existing != nil {
		return existing, err
	}
	mutation, err := calculateBankMutation(account, req.Amount, req.Type)
	if err != nil {
		return nil, err
	}
	if err := updateBankAccount(ctx, client, account, mutation); err != nil {
		return nil, err
	}
	return createBankTransaction(ctx, client, req, account, mutation)
}

// isRetryableBankTxErr 识别 PostgreSQL 可重试的序列化失败和死锁错误。
func isRetryableBankTxErr(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "could not serialize access") ||
		strings.Contains(msg, "serialization failure") ||
		strings.Contains(msg, "deadlock detected")
}

// isUniqueBankTxErr 将数据库唯一索引冲突识别为幂等冲突。
func isUniqueBankTxErr(err error) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
