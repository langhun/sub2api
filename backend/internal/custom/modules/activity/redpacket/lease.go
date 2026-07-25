package redpacket

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	coreservice "github.com/Wei-Shaw/sub2api/internal/service"
)

// LeaseCoordinator adapts the existing leader-lock cache and a PostgreSQL
// advisory-lock fallback to the Activity worker contract.
type LeaseCoordinator struct {
	cache coreservice.LeaderLockCache
	db    *sql.DB
}

func NewLeaseCoordinator(cache coreservice.LeaderLockCache, db *sql.DB) *LeaseCoordinator {
	return &LeaseCoordinator{cache: cache, db: db}
}

func (c *LeaseCoordinator) AcquireSingletonLease(ctx context.Context, key, owner string, ttl time.Duration) (contract.Lease, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c != nil && c.cache != nil {
		acquired, err := c.cache.TryAcquireLeaderLock(ctx, key, owner, ttl)
		if err == nil {
			if !acquired {
				return nil, false, nil
			}
			return cacheLease{cache: c.cache, key: key, owner: owner}, true, nil
		}
	}
	if c != nil && c.db != nil {
		return acquireDatabaseLease(ctx, c.db, advisoryLockID(key))
	}
	// A no-backend deployment must still process expiries. This mirrors the
	// legacy single-instance fallback and is never used when Redis or Postgres
	// coordination is available.
	return noopLease{}, true, nil
}

type cacheLease struct {
	cache coreservice.LeaderLockCache
	key   string
	owner string
}

func (l cacheLease) Release(ctx context.Context) error {
	if l.cache == nil {
		return nil
	}
	return l.cache.ReleaseLeaderLock(ctx, l.key, l.owner)
}

type databaseLease struct {
	conn   *sql.Conn
	lockID int64
}

func (l databaseLease) Release(ctx context.Context) error {
	if l.conn == nil {
		return nil
	}
	defer l.conn.Close()
	if _, err := l.conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", l.lockID); err != nil {
		return fmt.Errorf("release advisory lease: %w", err)
	}
	return nil
}

func acquireDatabaseLease(ctx context.Context, db *sql.DB, lockID int64) (contract.Lease, bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("open advisory-lock connection: %w", err)
	}
	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, fmt.Errorf("query advisory lock: %w", err)
	}
	if !acquired {
		_ = conn.Close()
		return nil, false, nil
	}
	return databaseLease{conn: conn, lockID: lockID}, true, nil
}

func advisoryLockID(key string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(key))
	return int64(hash.Sum64())
}
