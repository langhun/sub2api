// Package upstreamcost resolves a fresh per-Key upstream billing multiplier.
package upstreamcost

import (
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	probeExtraKey        = "upstream_billing_probe"
	probeEnabledExtraKey = "upstream_billing_probe_enabled"
)

// Resolver reads the sanitized response written by the upstream billing probe.
// It deliberately has no write path: every usage log freezes the value resolved
// at request time, while the account's manually configured multiplier remains a
// fallback owned by core.
type Resolver struct{}

func NewResolver() *Resolver { return &Resolver{} }

func (r *Resolver) ResolveAccountUsageMultiplier(account *service.Account, now time.Time) (float64, bool) {
	if account == nil || !account.IsOpenAIApiKey() || account.Extra == nil {
		return 0, false
	}
	if enabled, ok := account.Extra[probeEnabledExtraKey].(bool); !ok || !enabled {
		return 0, false
	}

	snapshot, ok := account.Extra[probeExtraKey].(map[string]any)
	if !ok || snapshot == nil {
		return 0, false
	}
	status, _ := snapshot["status"].(string)
	if status != service.UpstreamBillingProbeStatusOK && status != service.UpstreamBillingProbeStatusFailed {
		return 0, false
	}
	freshUntil, ok := parseTime(snapshot["fresh_until"])
	if !ok || !now.Before(freshUntil) {
		return 0, false
	}
	data, ok := snapshot["data"].(map[string]any)
	if !ok || data == nil || stringValue(data["billing_scope"]) != "token" {
		return 0, false
	}

	base, ok := finiteNonNegativeNumber(data["resolved_rate_multiplier"])
	if !ok {
		return 0, false
	}
	peakEnabled, ok := data["peak_rate_enabled"].(bool)
	if !ok || !peakEnabled {
		return base, ok
	}
	peakMultiplier, ok := finiteNonNegativeNumber(data["peak_rate_multiplier"])
	if !ok || !isInPeakWindow(now, stringValue(data["peak_start"]), stringValue(data["peak_end"]), stringValue(data["timezone"])) {
		return base, true
	}
	value := base * peakMultiplier
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func parseTime(value any) (time.Time, bool) {
	raw, ok := value.(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	return parsed, err == nil && !parsed.IsZero()
}

func stringValue(value any) string {
	valueString, _ := value.(string)
	return strings.TrimSpace(valueString)
}

func finiteNonNegativeNumber(value any) (float64, bool) {
	valueFloat, ok := value.(float64)
	return valueFloat, ok && valueFloat >= 0 && !math.IsNaN(valueFloat) && !math.IsInf(valueFloat, 0)
}

func isInPeakWindow(now time.Time, start, end, timezone string) bool {
	location, err := time.LoadLocation(timezone)
	if err != nil || start == "" || end == "" {
		return false
	}
	startMinute, startOK := parseMinute(start)
	endMinute, endOK := parseMinute(end)
	if !startOK || !endOK || startMinute == endMinute {
		return false
	}
	local := now.In(location)
	minute := local.Hour()*60 + local.Minute()
	if startMinute < endMinute {
		return minute >= startMinute && minute < endMinute
	}
	return minute >= startMinute || minute < endMinute
}

func parseMinute(value string) (int, bool) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}
