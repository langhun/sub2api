package xznoauthinvoice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) SaveOwnership(ctx context.Context, userID, remoteID int64, status string) error {
	if r == nil || r.db == nil {
		return errors.New("invoice database is unavailable")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO custom_invoice_applications (local_user_id, remote_application_id, status)
VALUES ($1,$2,$3)
ON CONFLICT (local_user_id, remote_application_id) DO UPDATE SET status=EXCLUDED.status, updated_at=NOW()`,
		userID, remoteID, status)
	return err
}

func (r *Repository) Owned(ctx context.Context, userID, remoteID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("invoice database is unavailable")
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM custom_invoice_applications WHERE local_user_id=$1 AND remote_application_id=$2)`, userID, remoteID).Scan(&exists)
	return exists, err
}

func (r *Repository) ListOwnedIDs(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice database is unavailable")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT remote_application_id FROM custom_invoice_applications WHERE local_user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]struct{})
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

func (r *Repository) OwnedPage(ctx context.Context, userID int64, page, pageSize int) ([]int64, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, errors.New("invoice database is unavailable")
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM custom_invoice_applications WHERE local_user_id=$1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT remote_application_id FROM custom_invoice_applications
WHERE local_user_id=$1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`,
		userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	ids := make([]int64, 0, pageSize)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

func (r *Repository) SaveTaxOrders(ctx context.Context, userID, remoteID int64, payments map[string]TaxPayment) error {
	if r == nil || r.db == nil {
		return errors.New("invoice database is unavailable")
	}
	for _, payment := range payments {
		if payment.TaxOrderNo == "" {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `INSERT INTO custom_invoice_tax_orders
(local_user_id, remote_application_id, tax_order_no)
VALUES ($1,$2,$3) ON CONFLICT (local_user_id, remote_application_id, tax_order_no) DO NOTHING`,
			userID, remoteID, payment.TaxOrderNo); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) OwnsTaxOrder(ctx context.Context, userID, remoteID int64, taxOrderNo string) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("invoice database is unavailable")
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM custom_invoice_tax_orders
WHERE local_user_id=$1 AND remote_application_id=$2 AND tax_order_no=$3)`,
		userID, remoteID, taxOrderNo).Scan(&exists)
	return exists, err
}

// OwnedTaxOrderApplication resolves a tax order to the local application that
// owns it. The browser facade can therefore accept the merchant contract's
// taxOrderNo-only status request without trusting an application ID from the client.
func (r *Repository) OwnedTaxOrderApplication(ctx context.Context, userID int64, taxOrderNo string) (int64, bool, error) {
	if r == nil || r.db == nil {
		return 0, false, errors.New("invoice database is unavailable")
	}
	var remoteID int64
	err := r.db.QueryRowContext(ctx, `SELECT remote_application_id FROM custom_invoice_tax_orders
WHERE local_user_id=$1 AND tax_order_no=$2 LIMIT 1`, userID, taxOrderNo).Scan(&remoteID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return remoteID, err == nil, err
}

func (r *Repository) UpdateStatus(ctx context.Context, userID, remoteID int64, status string) error {
	if r == nil || r.db == nil {
		return errors.New("invoice database is unavailable")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE custom_invoice_applications SET status=$3, updated_at=NOW() WHERE local_user_id=$1 AND remote_application_id=$2`, userID, remoteID, status)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return fmt.Errorf("invoice ownership not found")
	}
	return err
}

type ownershipRecord struct {
	UserID    int64
	RemoteID  int64
	Status    string
	CreatedAt time.Time
}
