package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type proxySubscriptionSourceRepository struct {
	client *dbent.Client
	sql    *sql.DB
}

type proxySubscriptionNodeRepository struct {
	client *dbent.Client
	sql    *sql.DB
}

func NewProxySubscriptionSourceRepository(client *dbent.Client, db *sql.DB) service.ProxySubscriptionSourceRepository {
	return &proxySubscriptionSourceRepository{client: client, sql: db}
}

func NewProxySubscriptionNodeRepository(client *dbent.Client, db *sql.DB) service.ProxySubscriptionNodeRepository {
	return &proxySubscriptionNodeRepository{client: client, sql: db}
}

func (r *proxySubscriptionSourceRepository) Create(ctx context.Context, source *service.ProxySubscriptionSource) error {
	now := time.Now()
	query := `
		INSERT INTO proxy_subscription_sources
			(name, url, source_format, enabled, refresh_interval_hours, target_entry_count, auto_add_to_pool,
			 last_refreshed_at, last_success_at, last_error, last_node_count, last_materialized_proxy_count,
			 created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7,
			 $8, $9, $10, $11, $12,
			 $13, $14)
		RETURNING id, created_at, updated_at
	`
	return scanSingleRow(ctx, r.sql, query, []any{
		source.Name,
		source.URL,
		source.SourceFormat,
		source.Enabled,
		source.RefreshIntervalHours,
		source.TargetEntryCount,
		source.AutoAddToPool,
		source.LastRefreshedAt,
		source.LastSuccessAt,
		nullableString(source.LastError),
		source.LastNodeCount,
		source.LastMaterializedProxyCount,
		now,
		now,
	}, &source.ID, &source.CreatedAt, &source.UpdatedAt)
}

func (r *proxySubscriptionSourceRepository) GetByID(ctx context.Context, id int64) (*service.ProxySubscriptionSource, error) {
	query := `
		SELECT id, name, url, source_format, enabled, refresh_interval_hours, target_entry_count, auto_add_to_pool,
		       last_refreshed_at, last_success_at, last_error, last_node_count, last_materialized_proxy_count,
		       created_at, updated_at
		FROM proxy_subscription_sources
		WHERE id = $1 AND deleted_at IS NULL
	`
	source, err := scanProxySubscriptionSource(ctx, r.sql, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrProxyNotFound
		}
		return nil, err
	}
	return source, nil
}

func (r *proxySubscriptionSourceRepository) List(ctx context.Context, page, pageSize int, search string, enabled *bool) ([]service.ProxySubscriptionSource, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	clauses := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 4)
	if search = strings.TrimSpace(search); search != "" {
		clauses = append(clauses, "(name ILIKE $"+itoa(len(args)+1)+" OR url ILIKE $"+itoa(len(args)+1)+")")
		args = append(args, "%"+search+"%")
	}
	if enabled != nil {
		clauses = append(clauses, "enabled = $"+itoa(len(args)+1))
		args = append(args, *enabled)
	}
	where := strings.Join(clauses, " AND ")

	var total int64
	countQuery := "SELECT COUNT(*) FROM proxy_subscription_sources WHERE " + where
	if err := scanSingleRow(ctx, r.sql, countQuery, args, &total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), pageSize, offset)
	query := `
		SELECT id, name, url, source_format, enabled, refresh_interval_hours, target_entry_count, auto_add_to_pool,
		       last_refreshed_at, last_success_at, last_error, last_node_count, last_materialized_proxy_count,
		       created_at, updated_at
		FROM proxy_subscription_sources
		WHERE ` + where + `
		ORDER BY id DESC
		LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	rows, err := r.sql.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ProxySubscriptionSource, 0)
	for rows.Next() {
		item, err := scanProxySubscriptionSourceRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *proxySubscriptionSourceRepository) ListEnabled(ctx context.Context) ([]service.ProxySubscriptionSource, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, url, source_format, enabled, refresh_interval_hours, target_entry_count, auto_add_to_pool,
		       last_refreshed_at, last_success_at, last_error, last_node_count, last_materialized_proxy_count,
		       created_at, updated_at
		FROM proxy_subscription_sources
		WHERE deleted_at IS NULL AND enabled = TRUE
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ProxySubscriptionSource, 0)
	for rows.Next() {
		item, err := scanProxySubscriptionSourceRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *proxySubscriptionSourceRepository) ListDueForRefresh(ctx context.Context, now time.Time, limit int) ([]service.ProxySubscriptionSource, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, url, source_format, enabled, refresh_interval_hours, target_entry_count, auto_add_to_pool,
		       last_refreshed_at, last_success_at, last_error, last_node_count, last_materialized_proxy_count,
		       created_at, updated_at
		FROM proxy_subscription_sources
		WHERE deleted_at IS NULL
		  AND enabled = TRUE
		  AND (
		      last_refreshed_at IS NULL OR
		      last_refreshed_at <= $1 - make_interval(hours => refresh_interval_hours)
		  )
		ORDER BY COALESCE(last_refreshed_at, TIMESTAMPTZ '1970-01-01') ASC, id ASC
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ProxySubscriptionSource, 0)
	for rows.Next() {
		item, err := scanProxySubscriptionSourceRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *proxySubscriptionSourceRepository) Update(ctx context.Context, source *service.ProxySubscriptionSource) error {
	now := time.Now()
	_, err := r.sql.ExecContext(ctx, `
		UPDATE proxy_subscription_sources
		SET name = $2,
		    url = $3,
		    source_format = $4,
		    enabled = $5,
		    refresh_interval_hours = $6,
		    target_entry_count = $7,
		    auto_add_to_pool = $8,
		    last_refreshed_at = $9,
		    last_success_at = $10,
		    last_error = $11,
		    last_node_count = $12,
		    last_materialized_proxy_count = $13,
		    updated_at = $14
		WHERE id = $1 AND deleted_at IS NULL
	`, source.ID, source.Name, source.URL, source.SourceFormat, source.Enabled, source.RefreshIntervalHours,
		source.TargetEntryCount, source.AutoAddToPool, source.LastRefreshedAt, source.LastSuccessAt, nullableString(source.LastError),
		source.LastNodeCount, source.LastMaterializedProxyCount, now)
	if err != nil {
		return err
	}
	source.UpdatedAt = now
	return nil
}

func (r *proxySubscriptionSourceRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE proxy_subscription_sources
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

func (r *proxySubscriptionNodeRepository) Create(ctx context.Context, node *service.ProxySubscriptionNode) error {
	now := time.Now()
	rawConfig, err := json.Marshal(node.ConfigJSON)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO proxy_subscription_nodes
			(source_id, node_key, display_name, node_type, server, port, config_json,
			 landing_status, last_error, last_seen_at, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7::jsonb,
			 $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`
	return scanSingleRow(ctx, r.sql, query, []any{
		node.SourceID,
		node.NodeKey,
		nullableString(node.DisplayName),
		node.NodeType,
		node.Server,
		node.Port,
		string(rawConfig),
		node.LandingStatus,
		nullableString(node.LastError),
		node.LastSeenAt,
		now,
		now,
	}, &node.ID, &node.CreatedAt, &node.UpdatedAt)
}

func (r *proxySubscriptionNodeRepository) Update(ctx context.Context, node *service.ProxySubscriptionNode) error {
	now := time.Now()
	rawConfig, err := json.Marshal(node.ConfigJSON)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `
		UPDATE proxy_subscription_nodes
		SET display_name = $2,
		    node_type = $3,
		    server = $4,
		    port = $5,
		    config_json = $6::jsonb,
		    landing_status = $7,
		    last_error = $8,
		    last_seen_at = $9,
		    updated_at = $10,
		    deleted_at = NULL
		WHERE id = $1
	`, node.ID, nullableString(node.DisplayName), node.NodeType, node.Server, node.Port, string(rawConfig),
		node.LandingStatus, nullableString(node.LastError), node.LastSeenAt, now)
	if err != nil {
		return err
	}
	node.UpdatedAt = now
	return nil
}

func (r *proxySubscriptionNodeRepository) GetByID(ctx context.Context, id int64) (*service.ProxySubscriptionNode, error) {
	return scanProxySubscriptionNode(ctx, r.sql, `
		SELECT id, source_id, node_key, display_name, node_type, server, port, config_json,
		       landing_status, last_error, last_seen_at, created_at, updated_at
		FROM proxy_subscription_nodes
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
}

func (r *proxySubscriptionNodeRepository) ListBySourceID(ctx context.Context, sourceID int64) ([]service.ProxySubscriptionNode, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, source_id, node_key, display_name, node_type, server, port, config_json,
		       landing_status, last_error, last_seen_at, created_at, updated_at
		FROM proxy_subscription_nodes
		WHERE source_id = $1 AND deleted_at IS NULL
		ORDER BY id ASC
	`, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ProxySubscriptionNode, 0)
	for rows.Next() {
		item, err := scanProxySubscriptionNodeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *proxySubscriptionNodeRepository) GetBySourceAndNodeKey(ctx context.Context, sourceID int64, nodeKey string) (*service.ProxySubscriptionNode, error) {
	return scanProxySubscriptionNode(ctx, r.sql, `
		SELECT id, source_id, node_key, display_name, node_type, server, port, config_json,
		       landing_status, last_error, last_seen_at, created_at, updated_at
		FROM proxy_subscription_nodes
		WHERE source_id = $1 AND node_key = $2 AND deleted_at IS NULL
	`, sourceID, nodeKey)
}

func (r *proxySubscriptionNodeRepository) SoftDeleteMissingBySourceID(ctx context.Context, sourceID int64, activeNodeKeys []string, now time.Time) error {
	if len(activeNodeKeys) == 0 {
		_, err := r.sql.ExecContext(ctx, `
			UPDATE proxy_subscription_nodes
			SET landing_status = $2, deleted_at = $3, updated_at = $3
			WHERE source_id = $1 AND deleted_at IS NULL
		`, sourceID, service.ProxySubscriptionLandingStatusStale, now)
		return err
	}

	args := []any{sourceID, service.ProxySubscriptionLandingStatusStale, now}
	placeholders := make([]string, 0, len(activeNodeKeys))
	for _, item := range activeNodeKeys {
		args = append(args, item)
		placeholders = append(placeholders, "$"+strconv.Itoa(len(args)))
	}
	_, err := r.sql.ExecContext(ctx, `
		UPDATE proxy_subscription_nodes
		SET landing_status = $2, deleted_at = $3, updated_at = $3
		WHERE source_id = $1
		  AND deleted_at IS NULL
		  AND node_key NOT IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	return err
}

func scanProxySubscriptionSource(ctx context.Context, db *sql.DB, query string, args ...any) (*service.ProxySubscriptionSource, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	return scanProxySubscriptionSourceRow(rows)
}

func scanProxySubscriptionSourceRow(scanner interface{ Scan(dest ...any) error }) (*service.ProxySubscriptionSource, error) {
	var (
		item            service.ProxySubscriptionSource
		lastRefreshedAt sql.NullTime
		lastSuccessAt   sql.NullTime
		lastError       sql.NullString
	)
	if err := scanner.Scan(
		&item.ID, &item.Name, &item.URL, &item.SourceFormat, &item.Enabled, &item.RefreshIntervalHours,
		&item.TargetEntryCount, &item.AutoAddToPool, &lastRefreshedAt, &lastSuccessAt, &lastError, &item.LastNodeCount,
		&item.LastMaterializedProxyCount, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if lastRefreshedAt.Valid {
		item.LastRefreshedAt = &lastRefreshedAt.Time
	}
	if lastSuccessAt.Valid {
		item.LastSuccessAt = &lastSuccessAt.Time
	}
	if lastError.Valid {
		item.LastError = lastError.String
	}
	return &item, nil
}

func scanProxySubscriptionNode(ctx context.Context, db *sql.DB, query string, args ...any) (*service.ProxySubscriptionNode, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	return scanProxySubscriptionNodeRow(rows)
}

func scanProxySubscriptionNodeRow(scanner interface{ Scan(dest ...any) error }) (*service.ProxySubscriptionNode, error) {
	var (
		item        service.ProxySubscriptionNode
		displayName sql.NullString
		lastError   sql.NullString
		rawConfig   []byte
	)
	if err := scanner.Scan(
		&item.ID, &item.SourceID, &item.NodeKey, &displayName, &item.NodeType, &item.Server, &item.Port,
		&rawConfig, &item.LandingStatus, &lastError, &item.LastSeenAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if displayName.Valid {
		item.DisplayName = displayName.String
	}
	if lastError.Valid {
		item.LastError = lastError.String
	}
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &item.ConfigJSON); err != nil {
			return nil, err
		}
	}
	if item.ConfigJSON == nil {
		item.ConfigJSON = map[string]any{}
	}
	return &item, nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
