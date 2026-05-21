package service

import (
	"context"
	"time"
)

type proxySubscriptionAccountMover interface {
	BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error)
}

type ProxySubscriptionSourceRepository interface {
	Create(ctx context.Context, source *ProxySubscriptionSource) error
	GetByID(ctx context.Context, id int64) (*ProxySubscriptionSource, error)
	List(ctx context.Context, page, pageSize int, search string, enabled *bool) ([]ProxySubscriptionSource, int64, error)
	ListEnabled(ctx context.Context) ([]ProxySubscriptionSource, error)
	ListDueForRefresh(ctx context.Context, now time.Time, limit int) ([]ProxySubscriptionSource, error)
	Update(ctx context.Context, source *ProxySubscriptionSource) error
	Delete(ctx context.Context, id int64) error
}

type ProxySubscriptionNodeRepository interface {
	Create(ctx context.Context, node *ProxySubscriptionNode) error
	Update(ctx context.Context, node *ProxySubscriptionNode) error
	GetByID(ctx context.Context, id int64) (*ProxySubscriptionNode, error)
	ListBySourceID(ctx context.Context, sourceID int64) ([]ProxySubscriptionNode, error)
	GetBySourceAndNodeKey(ctx context.Context, sourceID int64, nodeKey string) (*ProxySubscriptionNode, error)
	SoftDeleteMissingBySourceID(ctx context.Context, sourceID int64, activeNodeKeys []string, now time.Time) error
}

type ProxySubscriptionRuntimeUpsertRequest struct {
	RuntimeID       string         `json:"runtime_id"`
	SourceName      string         `json:"source_name"`
	SourceFormat    string         `json:"source_format"`
	Subscription    string         `json:"subscription"`
	ProviderContent string         `json:"provider_content"`
	EntryIndex      int            `json:"entry_index"`
	NodeType        string         `json:"node_type"`
	DisplayName     string         `json:"display_name"`
	Server          string         `json:"server"`
	Port            int            `json:"port"`
	Config          map[string]any `json:"config"`
	ListenerHost    string         `json:"listener_host"`
	ListenerPort    int            `json:"listener_port"`
}

type ProxySubscriptionRuntimeUpsertResponse struct {
	RuntimeID    string `json:"runtime_id"`
	ListenerHost string `json:"listener_host"`
	ListenerPort int    `json:"listener_port"`
	Protocol     string `json:"protocol"`
}

type ProxySubscriptionRuntimeManager interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	DeleteRuntime(ctx context.Context, runtimeID string) error
	UpsertRuntime(ctx context.Context, req ProxySubscriptionRuntimeUpsertRequest) (*ProxySubscriptionRuntimeUpsertResponse, error)
	CheckRuntime(ctx context.Context, runtimeID string) error
}

type CreateProxySubscriptionSourceInput struct {
	Name                 string
	URL                  string
	SourceFormat         string
	Enabled              bool
	RefreshIntervalHours int
	TargetEntryCount     int
	AutoAddToPool        bool
}

type UpdateProxySubscriptionSourceInput struct {
	Name                 *string
	URL                  *string
	SourceFormat         *string
	Enabled              *bool
	RefreshIntervalHours *int
	TargetEntryCount     *int
	AutoAddToPool        *bool
}
