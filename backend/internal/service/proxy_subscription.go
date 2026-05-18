package service

import (
	"context"
	"time"
)

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

type ProxySubscriptionSidecarUpsertRequest struct {
	RuntimeID    string         `json:"runtime_id"`
	NodeType     string         `json:"node_type"`
	DisplayName  string         `json:"display_name"`
	Server       string         `json:"server"`
	Port         int            `json:"port"`
	Config       map[string]any `json:"config"`
	ListenerHost string         `json:"listener_host"`
}

type ProxySubscriptionSidecarUpsertResponse struct {
	RuntimeID    string `json:"runtime_id"`
	ListenerHost string `json:"listener_host"`
	ListenerPort int    `json:"listener_port"`
	Protocol     string `json:"protocol"`
}

type ProxySubscriptionSidecarClient interface {
	HealthCheck(ctx context.Context) error
	UpsertRuntime(ctx context.Context, req ProxySubscriptionSidecarUpsertRequest) (*ProxySubscriptionSidecarUpsertResponse, error)
	DeleteRuntime(ctx context.Context, runtimeID string) error
	CheckRuntime(ctx context.Context, runtimeID string) error
}

type CreateProxySubscriptionSourceInput struct {
	Name                 string
	URL                  string
	SourceFormat         string
	Enabled              bool
	RefreshIntervalHours int
	AutoAddToPool        bool
}

type UpdateProxySubscriptionSourceInput struct {
	Name                 *string
	URL                  *string
	SourceFormat         *string
	Enabled              *bool
	RefreshIntervalHours *int
	AutoAddToPool        *bool
}
