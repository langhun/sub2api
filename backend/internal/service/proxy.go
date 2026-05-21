package service

import (
	"net"
	"net/url"
	"strconv"
	"time"
)

type Proxy struct {
	ID                      int64
	Name                    string
	Protocol                string
	Host                    string
	Port                    int
	Username                string
	Password                string
	Status                  string
	AutoFailoverPoolEnabled bool
	SubscriptionSourceID    *int64
	SubscriptionNodeID      *int64
	SubscriptionSourceName  string
	SubscriptionNodeType    string
	ManagedBySubscription   bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

func (p *Proxy) URL() string {
	u := &url.URL{
		Scheme: p.Protocol,
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
	}
	if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount           int64
	LatencyMs              *int64
	LatencyStatus          string
	LatencyMessage         string
	IPAddress              string
	Country                string
	CountryCode            string
	Region                 string
	City                   string
	QualityStatus          string
	QualityScore           *int
	QualityGrade           string
	QualitySummary         string
	QualityChecked         *int64
	HealthStatus           string
	CooldownUntilUnix      *int64
	LastFailReason         string
	LastFailAtUnix         *int64
	LastRecoveredAtUnix    *int64
	FailoverSwitchCount    *int64
	SubscriptionSourceName string
	SubscriptionNodeType   string
}

type ProxySubscriptionSource struct {
	ID                         int64
	Name                       string
	URL                        string
	SourceFormat               string
	Enabled                    bool
	RefreshIntervalHours       int
	TargetEntryCount           int
	AutoAddToPool              bool
	LastRefreshedAt            *time.Time
	LastSuccessAt              *time.Time
	LastError                  string
	LastNodeCount              int
	LastMaterializedProxyCount int
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type ProxySubscriptionNode struct {
	ID            int64
	SourceID      int64
	NodeKey       string
	DisplayName   string
	NodeType      string
	Server        string
	Port          int
	ConfigJSON    map[string]any
	LandingStatus string
	LastError     string
	LastSeenAt    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProxySubscriptionRefreshResult struct {
	SourceID               int64                    `json:"source_id"`
	RefreshedAt            time.Time                `json:"refreshed_at"`
	NodeCount              int                      `json:"node_count"`
	MaterializedProxyCount int                      `json:"materialized_proxy_count"`
	CreatedProxyCount      int                      `json:"created_proxy_count"`
	UpdatedProxyCount      int                      `json:"updated_proxy_count"`
	DisabledProxyCount     int                      `json:"disabled_proxy_count"`
	DeletedProxyCount      int                      `json:"deleted_proxy_count"`
	SkippedNodeCount       int                      `json:"skipped_node_count"`
	ConflictNodeCount      int                      `json:"conflict_node_count"`
	UnsupportedNodeCount   int                      `json:"unsupported_node_count"`
	Errors                 []ProxySubscriptionError `json:"errors"`
}

type ProxySubscriptionError struct {
	NodeKey string `json:"node_key,omitempty"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message"`
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}
