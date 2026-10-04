// Package channelmonitorv2activeprobes projects the existing V1 probe runner
// into V2 channel-status responses without changing passive aggregation tables.
package channelmonitorv2activeprobes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	settingKeyEnabled = "channel_monitor_v2_active_probes_enabled"
	extensionKey      = "channel_monitor_v2_active_probes"
)

type Settings interface {
	GetMultiple(context.Context, []string) (map[string]string, error)
	SetMultiple(context.Context, map[string]string) error
}

type Scheduler interface {
	TriggerScheduledChecks()
}

type RuntimeReader interface {
	GetChannelMonitorRuntime(context.Context) service.ChannelMonitorRuntime
}

type Module struct {
	db              *sql.DB
	settings        Settings
	scheduler       Scheduler
	runtime         RuntimeReader
	now             func() time.Time
	lastCleanupUnix atomic.Int64
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type probeFact struct {
	MonitorID   int64
	Platform    string
	Model       string
	Status      string
	LatencyMs   *int64
	Interval    int
	CheckedAt   time.Time
	MappedModel string
	InWindow    bool
	Fresh       bool
}

type probeSource struct {
	PassiveTraffic       bool       `json:"passive_traffic,omitempty"`
	ActiveProbe          bool       `json:"active_probe,omitempty"`
	CacheSampled         bool       `json:"cache_sampled"`
	LastActiveProbeAt    *time.Time `json:"last_active_probe_at,omitempty"`
	ActiveProbeLatencyMs *int64     `json:"active_probe_latency_ms,omitempty"`
	ActiveProbeStatus    string     `json:"active_probe_status,omitempty"`
}

func NewModule(db *sql.DB, settings Settings, scheduler Scheduler, runtime RuntimeReader) *Module {
	return &Module{db: db, settings: settings, scheduler: scheduler, runtime: runtime, now: func() time.Time { return time.Now().UTC() }}
}

// Start restores prompt probing after a process restart when the Overlay was
// already enabled. The core runner starts first, so its initial fire may occur
// before this module is mounted.
func (m *Module) Start(ctx context.Context) {
	if m == nil || m.scheduler == nil || m.runtime == nil {
		return
	}
	runtime := m.runtime.GetChannelMonitorRuntime(ctx)
	if !runtime.Enabled || runtime.Mode != service.ChannelMonitorModeV2 {
		return
	}
	enabled, err := m.enabled(ctx)
	if err != nil {
		slog.Warn("channel_monitor_v2_active_probes: restore startup schedule", "error", err)
		return
	}
	if enabled {
		m.scheduler.TriggerScheduledChecks()
	}
}

func (m *Module) RegisterRoutes(admin *gin.RouterGroup) {
	if m == nil || admin == nil {
		return
	}
	group := admin.Group("/channel-monitor-v2-active-probes")
	group.GET("/config", m.getConfig)
	group.PUT("/config", m.updateConfig)
}

func (m *Module) getConfig(c *gin.Context) {
	enabled, err := m.enabled(c.Request.Context())
	if err != nil {
		writeConfigError(c, err)
		return
	}
	response.Success(c, Config{Enabled: enabled})
}

func (m *Module) updateConfig(c *gin.Context) {
	var input Config
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("ACTIVE_PROBE_INVALID_CONFIG", "enabled is required"))
		return
	}
	if m == nil || m.settings == nil {
		writeConfigError(c, errors.New("active probe settings are unavailable"))
		return
	}
	previous, err := m.enabled(c.Request.Context())
	if err != nil {
		writeConfigError(c, err)
		return
	}
	value := "false"
	if input.Enabled {
		value = "true"
	}
	if err := m.settings.SetMultiple(c.Request.Context(), map[string]string{settingKeyEnabled: value}); err != nil {
		writeConfigError(c, err)
		return
	}
	if input.Enabled && !previous && m.scheduler != nil {
		m.scheduler.TriggerScheduledChecks()
	}
	response.Success(c, input)
}

func writeConfigError(c *gin.Context, err error) {
	response.ErrorFrom(c, infraerrors.InternalServer("ACTIVE_PROBE_CONFIG_UNAVAILABLE", "active probe settings are temporarily unavailable").WithCause(err))
}

func (m *Module) enabled(ctx context.Context) (bool, error) {
	if m == nil || m.settings == nil {
		return false, errors.New("active probe settings are unavailable")
	}
	values, err := m.settings.GetMultiple(ctx, []string{settingKeyEnabled})
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(values[settingKeyEnabled]), "true"), nil
}

// AllowChannelMonitorCheck opts the existing runner into V2 probing without
// changing V1 behavior. Pure quota targets stay outside this Overlay.
func (m *Module) AllowChannelMonitorCheck(ctx context.Context, runtime service.ChannelMonitorRuntime, checkMode string) bool {
	if m == nil || runtime.Mode != service.ChannelMonitorModeV2 {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(checkMode), service.MonitorCheckModeQuota) {
		return false
	}
	enabled, err := m.enabled(ctx)
	if err != nil {
		slog.Warn("channel_monitor_v2_active_probes: read runner switch", "error", err)
		return false
	}
	return enabled
}

// ObserveChannelMonitorCheck consumes completed core checks. It records only
// V2 probe/quota_probe results and never stores credentials or response bodies.
func (m *Module) ObserveChannelMonitorCheck(ctx context.Context, runtime service.ChannelMonitorRuntime, monitor *service.ChannelMonitor, results []*service.CheckResult) {
	if m == nil || m.db == nil || monitor == nil || len(results) == 0 {
		return
	}
	if runtime.Mode != service.ChannelMonitorModeV2 {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(monitor.CheckMode))
	if mode == service.MonitorCheckModeQuota {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	tx, err := m.db.BeginTx(writeCtx, nil)
	if err != nil {
		slog.Warn("channel_monitor_v2_active_probes: begin snapshot", "monitor_id", monitor.ID, "error", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	for _, result := range results {
		if result == nil || strings.TrimSpace(result.Model) == "" {
			continue
		}
		_, err = tx.ExecContext(writeCtx, `
			INSERT INTO custom_channel_monitor_v2_active_probe_results
				(monitor_id, platform, model, status, latency_ms, interval_seconds, checked_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (monitor_id, model, checked_at) DO NOTHING`,
			monitor.ID, strings.ToLower(strings.TrimSpace(monitor.Provider)), strings.TrimSpace(result.Model),
			result.Status, result.LatencyMs, monitor.IntervalSeconds, result.CheckedAt.UTC())
		if err != nil {
			slog.Warn("channel_monitor_v2_active_probes: persist snapshot", "monitor_id", monitor.ID, "error", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		slog.Warn("channel_monitor_v2_active_probes: commit snapshot", "monitor_id", monitor.ID, "error", err)
		return
	}
	m.cleanupExpiredFacts()
}

func (m *Module) cleanupExpiredFacts() {
	if m == nil || m.db == nil {
		return
	}
	now := m.now().UTC()
	previous := m.lastCleanupUnix.Load()
	if previous > 0 && now.Unix()-previous < int64(6*time.Hour/time.Second) {
		return
	}
	if !m.lastCleanupUnix.CompareAndSwap(previous, now.Unix()) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := m.db.ExecContext(ctx, `DELETE FROM custom_channel_monitor_v2_active_probe_results WHERE checked_at < NOW() - INTERVAL '31 days'`); err != nil {
		slog.Warn("channel_monitor_v2_active_probes: cleanup expired facts", "error", err)
	}
}

func (m *Module) loadFacts(ctx context.Context, filter service.ChannelMonitorV2Filter, cfg service.ChannelMonitorV2Config) ([]probeFact, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("active probe database is unavailable")
	}
	enabled, err := m.enabled(ctx)
	if err != nil {
		// Query extensions fail closed so a transient settings read cannot take
		// the upstream passive V2 page down.
		slog.Warn("channel_monitor_v2_active_probes: read switch", "error", err)
		return nil, nil
	}
	if !enabled {
		return nil, nil
	}
	rows, err := m.db.QueryContext(ctx, `
		SELECT monitor_id, platform, model, status, latency_ms, interval_seconds, checked_at
		FROM custom_channel_monitor_v2_active_probe_results
		WHERE checked_at < $2
		  AND (checked_at >= $1 OR checked_at >= NOW() - make_interval(secs => interval_seconds * 3))
		ORDER BY checked_at ASC`, filter.Start.UTC(), filter.End.UTC())
	if err != nil {
		return nil, fmt.Errorf("query active probe facts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	platforms := configuredPlatforms(cfg)
	facts := make([]probeFact, 0)
	for rows.Next() {
		var fact probeFact
		var latency sql.NullInt64
		if err := rows.Scan(&fact.MonitorID, &fact.Platform, &fact.Model, &fact.Status, &latency, &fact.Interval, &fact.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan active probe fact: %w", err)
		}
		fact.Platform = strings.ToLower(strings.TrimSpace(fact.Platform))
		fact.Model = strings.TrimSpace(fact.Model)
		fact.InWindow = !fact.CheckedAt.Before(filter.Start)
		fact.Fresh = fact.Interval > 0 && !fact.CheckedAt.Before(m.now().UTC().Add(-3*time.Duration(fact.Interval)*time.Second))
		models, ok := platforms[fact.Platform]
		if !ok || !matchesStringFilter(fact.Platform, filter.Platforms) {
			continue
		}
		fact.MappedModel = fact.Model
		if len(models) > 0 && !contains(models, fact.Model) {
			fact.MappedModel = service.ChannelMonitorV2OtherModel
		}
		if !matchesStringFilter(fact.MappedModel, filter.Models) {
			continue
		}
		if latency.Valid {
			value := latency.Int64
			fact.LatencyMs = &value
		}
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

func inWindowFacts(facts []probeFact) []probeFact {
	result := make([]probeFact, 0, len(facts))
	for _, fact := range facts {
		if fact.InWindow {
			result = append(result, fact)
		}
	}
	return result
}

func freshFacts(facts []probeFact) []probeFact {
	result := make([]probeFact, 0, len(facts))
	for _, fact := range facts {
		if fact.Fresh {
			result = append(result, fact)
		}
	}
	return result
}

func configuredPlatforms(cfg service.ChannelMonitorV2Config) map[string][]string {
	result := make(map[string][]string)
	for _, platform := range cfg.Platforms {
		if platform.Enabled {
			result[strings.ToLower(strings.TrimSpace(platform.Platform))] = platform.Models
		}
	}
	return result
}

func matchesStringFilter(value string, filter []string) bool {
	return len(filter) == 0 || contains(filter, value)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func successful(status string) bool {
	return status == service.MonitorStatusOperational || status == service.MonitorStatusDegraded
}

func metricForFacts(facts []probeFact, minutes float64, admin bool) service.ChannelMonitorV2Metric {
	metric := service.ChannelMonitorV2Metric{}
	var latencySum int64
	for _, fact := range facts {
		if successful(fact.Status) {
			metric.SuccessRequests++
		} else {
			metric.ErrorRequests++
		}
		if fact.LatencyMs != nil {
			latencySum += *fact.LatencyMs
			metric.Duration.SampleCount++
		}
	}
	metric.RequestCount = metric.SuccessRequests + metric.ErrorRequests
	if metric.RequestCount > 0 {
		metric.SuccessRate = float64(metric.SuccessRequests) / float64(metric.RequestCount)
		metric.ErrorRate = float64(metric.ErrorRequests) / float64(metric.RequestCount)
	}
	if minutes <= 0 {
		minutes = 1
	}
	metric.RPM = float64(metric.RequestCount) / minutes
	if metric.Duration.SampleCount > 0 {
		avg := float64(latencySum) / float64(metric.Duration.SampleCount)
		metric.Duration.AvgMs = &avg
		latencies := make([]int64, 0, metric.Duration.SampleCount)
		for _, fact := range facts {
			if fact.LatencyMs != nil {
				latencies = append(latencies, *fact.LatencyMs)
			}
		}
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		metric.Duration.P50Ms = percentile(latencies, .50)
		metric.Duration.P90Ms = percentile(latencies, .90)
		metric.Duration.P95Ms = percentile(latencies, .95)
	}
	if admin {
		zero := int64(0)
		metric.UpstreamAffectedRequests = &zero
		metric.UpstreamAttemptCount = &zero
	}
	return metric
}

func percentile(values []int64, p float64) *int64 {
	if len(values) == 0 {
		return nil
	}
	index := int(float64(len(values))*p + .999999)
	if index < 1 {
		index = 1
	}
	if index > len(values) {
		index = len(values)
	}
	value := values[index-1]
	return &value
}

func mergeMetric(target *service.ChannelMonitorV2Metric, added service.ChannelMonitorV2Metric, minutes float64, admin bool) {
	if target == nil || added.RequestCount == 0 {
		return
	}
	oldDurationCount := target.Duration.SampleCount
	oldDurationAvg := target.Duration.AvgMs
	target.SuccessRequests += added.SuccessRequests
	target.ErrorRequests += added.ErrorRequests
	target.RequestCount = target.SuccessRequests + target.ErrorRequests
	if target.RequestCount > 0 {
		target.SuccessRate = float64(target.SuccessRequests) / float64(target.RequestCount)
		target.ErrorRate = float64(target.ErrorRequests) / float64(target.RequestCount)
	}
	if minutes <= 0 {
		minutes = 1
	}
	target.RPM = float64(target.RequestCount) / minutes
	if added.Duration.SampleCount > 0 {
		newCount := oldDurationCount + added.Duration.SampleCount
		oldSum := float64(0)
		if oldDurationAvg != nil {
			oldSum = *oldDurationAvg * float64(oldDurationCount)
		}
		addedSum := float64(0)
		if added.Duration.AvgMs != nil {
			addedSum = *added.Duration.AvgMs * float64(added.Duration.SampleCount)
		}
		avg := (oldSum + addedSum) / float64(newCount)
		target.Duration.SampleCount, target.Duration.AvgMs = newCount, &avg
		if oldDurationCount == 0 {
			target.Duration.P50Ms, target.Duration.P90Ms, target.Duration.P95Ms = added.Duration.P50Ms, added.Duration.P90Ms, added.Duration.P95Ms
		}
	}
	if admin {
		if target.UpstreamAffectedRequests == nil {
			zero := int64(0)
			target.UpstreamAffectedRequests = &zero
		}
		if target.UpstreamAttemptCount == nil {
			zero := int64(0)
			target.UpstreamAttemptCount = &zero
		}
	}
}

func sourceForFacts(facts []probeFact, passive bool) probeSource {
	source := probeSource{PassiveTraffic: passive, ActiveProbe: len(facts) > 0}
	latestByTarget := make(map[string]probeFact)
	for _, fact := range facts {
		targetKey := fmt.Sprintf("%d\x00%s", fact.MonitorID, fact.MappedModel)
		current, ok := latestByTarget[targetKey]
		if !ok || fact.CheckedAt.After(current.CheckedAt) {
			latestByTarget[targetKey] = fact
		}
		if source.LastActiveProbeAt == nil || fact.CheckedAt.After(*source.LastActiveProbeAt) {
			checkedAt := fact.CheckedAt
			source.LastActiveProbeAt = &checkedAt
			source.ActiveProbeStatus = fact.Status
			if fact.LatencyMs != nil {
				latency := *fact.LatencyMs
				source.ActiveProbeLatencyMs = &latency
			} else {
				source.ActiveProbeLatencyMs = nil
			}
		}
	}
	anyOperational := false
	anyDegraded := false
	for _, fact := range latestByTarget {
		anyOperational = anyOperational || fact.Status == service.MonitorStatusOperational
		anyDegraded = anyDegraded || fact.Status == service.MonitorStatusDegraded
	}
	if len(latestByTarget) > 0 {
		switch {
		case anyOperational:
			source.ActiveProbeStatus = service.MonitorStatusOperational
		case anyDegraded:
			source.ActiveProbeStatus = service.MonitorStatusDegraded
		default:
			source.ActiveProbeStatus = service.MonitorStatusFailed
		}
	}
	return source
}

func activeOnlyHealth(source probeSource, thresholds service.ChannelMonitorV2HealthThresholds) service.ChannelMonitorV2Health {
	thresholds = service.NormalizeChannelMonitorV2HealthThresholds(thresholds)
	health := service.ChannelMonitorV2HealthForWithThresholds(service.ChannelMonitorV2Metric{}, thresholds)
	if !source.ActiveProbe {
		return health
	}
	switch source.ActiveProbeStatus {
	case service.MonitorStatusOperational:
		health.Overall, health.ErrorRate = "healthy", "healthy"
		score := float64(100)
		health.Score, health.ErrorRateScore = &score, &score
		if source.ActiveProbeLatencyMs != nil && *source.ActiveProbeLatencyMs >= thresholds.WarningTTFTMs {
			health.Overall = "warning"
			score = 50
			health.Score = &score
		}
	case service.MonitorStatusDegraded:
		health.Overall, health.ErrorRate = "warning", "warning"
		score := float64(50)
		health.Score, health.ErrorRateScore = &score, &score
	default:
		health.Overall, health.ErrorRate = "critical", "critical"
		score := float64(0)
		health.Score, health.ErrorRateScore = &score, &score
	}
	return health
}

// healthForSource keeps active probes useful when passive traffic is sparse.
// Probe responses intentionally do not contain provider cache usage, so a
// missing cache denominator is never treated as a cache miss or allowed to
// make an otherwise usable channel critical.
func healthForSource(source probeSource, metrics service.ChannelMonitorV2Metric, passive bool, thresholds service.ChannelMonitorV2HealthThresholds) service.ChannelMonitorV2Health {
	if !source.ActiveProbe {
		return service.ChannelMonitorV2HealthForWithThresholds(metrics, thresholds)
	}
	if !passive {
		return activeOnlyHealth(source, thresholds)
	}

	health := service.ChannelMonitorV2HealthForWithThresholds(metrics, thresholds)
	if metrics.CacheRateDenominator == 0 {
		health.Cache = "unknown"
		health.CacheScore = nil
	}

	// A mixed row can contain a small amount of historical passive traffic
	// without enough samples for an error/latency score. In that case the
	// fresh active probe is the only meaningful availability signal.
	if metrics.CacheRateDenominator == 0 &&
		health.ErrorRate == "unknown" &&
		health.TTFT == "unknown" {
		return activeOnlyHealth(source, thresholds)
	}
	return health
}

func setExtension(target *map[string]any, source probeSource) {
	if target == nil {
		return
	}
	if *target == nil {
		*target = make(map[string]any, 1)
	}
	(*target)[extensionKey] = source
}

func durationMinutes(filter service.ChannelMonitorV2Filter) float64 {
	minutes := filter.End.Sub(filter.Start).Minutes()
	if minutes <= 0 {
		return 1
	}
	return minutes
}

func (m *Module) ExtendChannelMonitorV2Dimensions(ctx context.Context, filter service.ChannelMonitorV2Filter, cfg service.ChannelMonitorV2Config, dimensions *service.ChannelMonitorV2Dimensions) error {
	if len(filter.GroupIDs) > 0 {
		return nil
	}
	facts, err := m.loadFacts(ctx, filter, cfg)
	if err != nil || dimensions == nil {
		return err
	}
	platformCounts := make(map[string]int64)
	modelCounts := make(map[string]int64)
	for _, fact := range facts {
		modelKey := fact.Platform + "\x00" + fact.MappedModel
		if _, exists := platformCounts[fact.Platform]; !exists {
			platformCounts[fact.Platform] = 0
		}
		if _, exists := modelCounts[modelKey]; !exists {
			modelCounts[modelKey] = 0
		}
		if fact.InWindow {
			platformCounts[fact.Platform]++
			modelCounts[modelKey]++
		}
	}
	for platform, count := range platformCounts {
		found := false
		for i := range dimensions.Platforms {
			if dimensions.Platforms[i].Value == platform {
				dimensions.Platforms[i].RequestCount += count
				found = true
			}
		}
		if !found {
			dimensions.Platforms = append(dimensions.Platforms, service.ChannelMonitorV2Dimension{Value: platform, Label: platform, RequestCount: count})
		}
	}
	for key, count := range modelCounts {
		parts := strings.SplitN(key, "\x00", 2)
		found := false
		for i := range dimensions.Models {
			if dimensions.Models[i].Platform == parts[0] && dimensions.Models[i].Value == parts[1] {
				dimensions.Models[i].RequestCount += count
				found = true
			}
		}
		if !found {
			dimensions.Models = append(dimensions.Models, service.ChannelMonitorV2Dimension{Value: parts[1], Label: parts[1], Platform: parts[0], RequestCount: count})
		}
	}
	return nil
}

func (m *Module) ExtendChannelMonitorV2Snapshot(ctx context.Context, filter service.ChannelMonitorV2Filter, cfg service.ChannelMonitorV2Config, snapshot *service.ChannelMonitorV2Snapshot, admin bool) error {
	if len(filter.GroupIDs) > 0 {
		return nil
	}
	facts, err := m.loadFacts(ctx, filter, cfg)
	if err != nil || snapshot == nil {
		return err
	}
	passive := snapshot.Metrics.RequestCount > 0
	windowFacts := inWindowFacts(facts)
	added := metricForFacts(windowFacts, durationMinutes(filter), admin)
	mergeMetric(&snapshot.Metrics, added, durationMinutes(filter), admin)
	source := sourceForFacts(freshFacts(facts), passive)
	source.CacheSampled = snapshot.Metrics.CacheRateDenominator > 0
	setExtension(&snapshot.Extensions, source)
	snapshot.Health = healthForSource(source, snapshot.Metrics, passive, cfg.HealthThresholds)
	byBucket := make(map[time.Time][]probeFact)
	for _, fact := range windowFacts {
		bucket := fact.CheckedAt.UTC().Truncate(filter.Bucket)
		byBucket[bucket] = append(byBucket[bucket], fact)
	}
	for bucket, bucketFacts := range byBucket {
		index := -1
		for i := range snapshot.Trend {
			if snapshot.Trend[i].BucketStart.Equal(bucket) {
				index = i
				break
			}
		}
		if index < 0 {
			snapshot.Trend = append(snapshot.Trend, service.ChannelMonitorV2TrendPoint{BucketStart: bucket})
			index = len(snapshot.Trend) - 1
		}
		point := &snapshot.Trend[index]
		passivePoint := point.Metrics.RequestCount > 0
		mergeMetric(&point.Metrics, metricForFacts(bucketFacts, filter.Bucket.Minutes(), admin), filter.Bucket.Minutes(), admin)
		pointSource := sourceForFacts(freshFacts(bucketFacts), passivePoint)
		pointSource.CacheSampled = point.Metrics.CacheRateDenominator > 0
		setExtension(&point.Extensions, pointSource)
		point.Health = healthForSource(pointSource, point.Metrics, passivePoint, cfg.HealthThresholds)
	}
	sort.Slice(snapshot.Trend, func(i, j int) bool { return snapshot.Trend[i].BucketStart.Before(snapshot.Trend[j].BucketStart) })
	return nil
}

func (m *Module) ExtendChannelMonitorV2Models(ctx context.Context, filter service.ChannelMonitorV2Filter, cfg service.ChannelMonitorV2Config, list *service.ChannelMonitorV2List[service.ChannelMonitorV2ModelRow], admin bool) error {
	if len(filter.GroupIDs) > 0 {
		return nil
	}
	facts, err := m.loadFacts(ctx, filter, cfg)
	if err != nil || list == nil {
		return err
	}
	grouped := make(map[string][]probeFact)
	for _, fact := range facts {
		grouped[fact.Platform+"\x00"+fact.MappedModel] = append(grouped[fact.Platform+"\x00"+fact.MappedModel], fact)
	}
	for key, group := range grouped {
		parts := strings.SplitN(key, "\x00", 2)
		index := -1
		for i := range list.Items {
			if list.Items[i].Platform == parts[0] && list.Items[i].Model == parts[1] {
				index = i
				break
			}
		}
		if index < 0 {
			list.Items = append(list.Items, service.ChannelMonitorV2ModelRow{Platform: parts[0], Model: parts[1]})
			index = len(list.Items) - 1
		}
		row := &list.Items[index]
		passive := row.Metrics.RequestCount > 0
		mergeMetric(&row.Metrics, metricForFacts(inWindowFacts(group), durationMinutes(filter), admin), durationMinutes(filter), admin)
		rowSource := sourceForFacts(freshFacts(group), passive)
		rowSource.CacheSampled = row.Metrics.CacheRateDenominator > 0
		setExtension(&row.Extensions, rowSource)
		row.Health = healthForSource(rowSource, row.Metrics, passive, cfg.HealthThresholds)
	}
	sort.Slice(list.Items, func(i, j int) bool {
		if list.Items[i].Platform == list.Items[j].Platform {
			return list.Items[i].Model < list.Items[j].Model
		}
		return list.Items[i].Platform < list.Items[j].Platform
	})
	return nil
}

func (m *Module) ExtendChannelMonitorV2Matrix(ctx context.Context, filter service.ChannelMonitorV2Filter, cfg service.ChannelMonitorV2Config, groupBy service.ChannelMonitorV2GroupBy, matrix *service.ChannelMonitorV2Matrix, admin bool) error {
	if len(filter.GroupIDs) > 0 || matrix == nil || groupBy == service.ChannelMonitorV2GroupByPlatformGroup || groupBy == service.ChannelMonitorV2GroupByPlatformGroupModel {
		return nil
	}
	facts, err := m.loadFacts(ctx, filter, cfg)
	if err != nil {
		return err
	}
	grouped := make(map[string][]probeFact)
	for _, fact := range facts {
		key := fact.Platform
		if groupBy == service.ChannelMonitorV2GroupByPlatformModel || groupBy == service.ChannelMonitorV2GroupByPlatformGroupModel {
			key += "\x00" + fact.MappedModel
		}
		grouped[key] = append(grouped[key], fact)
	}
	for key, group := range grouped {
		parts := strings.SplitN(key, "\x00", 2)
		model := ""
		if len(parts) == 2 {
			model = parts[1]
		}
		index := -1
		for i := range matrix.Items {
			if matrix.Items[i].Platform == parts[0] && matrix.Items[i].GroupID == nil && matrix.Items[i].Model == model {
				index = i
				break
			}
		}
		if index < 0 {
			matrix.Items = append(matrix.Items, service.ChannelMonitorV2MatrixRow{Platform: parts[0], Model: model, Buckets: []service.ChannelMonitorV2TrendPoint{}})
			index = len(matrix.Items) - 1
		}
		row := &matrix.Items[index]
		passive := row.Metrics.RequestCount > 0
		windowGroup := inWindowFacts(group)
		mergeMetric(&row.Metrics, metricForFacts(windowGroup, durationMinutes(filter), admin), durationMinutes(filter), admin)
		rowSource := sourceForFacts(freshFacts(group), passive)
		rowSource.CacheSampled = row.Metrics.CacheRateDenominator > 0
		setExtension(&row.Extensions, rowSource)
		row.Health = healthForSource(rowSource, row.Metrics, passive, cfg.HealthThresholds)
		bucketFacts := make(map[time.Time][]probeFact)
		for _, fact := range windowGroup {
			bucket := fact.CheckedAt.UTC().Truncate(filter.Bucket)
			bucketFacts[bucket] = append(bucketFacts[bucket], fact)
		}
		for bucket, factsInBucket := range bucketFacts {
			bucketIndex := -1
			for i := range row.Buckets {
				if row.Buckets[i].BucketStart.Equal(bucket) {
					bucketIndex = i
					break
				}
			}
			if bucketIndex < 0 {
				row.Buckets = append(row.Buckets, service.ChannelMonitorV2TrendPoint{BucketStart: bucket})
				bucketIndex = len(row.Buckets) - 1
			}
			point := &row.Buckets[bucketIndex]
			passivePoint := point.Metrics.RequestCount > 0
			mergeMetric(&point.Metrics, metricForFacts(factsInBucket, filter.Bucket.Minutes(), admin), filter.Bucket.Minutes(), admin)
			pointSource := sourceForFacts(freshFacts(factsInBucket), passivePoint)
			pointSource.CacheSampled = point.Metrics.CacheRateDenominator > 0
			setExtension(&point.Extensions, pointSource)
			point.Health = healthForSource(pointSource, point.Metrics, passivePoint, cfg.HealthThresholds)
		}
		sort.Slice(row.Buckets, func(i, j int) bool { return row.Buckets[i].BucketStart.Before(row.Buckets[j].BucketStart) })
	}
	return nil
}

var _ service.ChannelMonitorCheckExtension = (*Module)(nil)
var _ service.ChannelMonitorV2DataExtension = (*Module)(nil)
