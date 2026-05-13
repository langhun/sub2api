package service

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGetOpsAdvancedSettings_DefaultHidesOpenAITokenStats(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if cfg.DisplayOpenAITokenStats {
		t.Fatalf("DisplayOpenAITokenStats = true, want false by default")
	}
	if !cfg.DisplayAlertEvents {
		t.Fatalf("DisplayAlertEvents = false, want true by default")
	}
	if repo.setCalls != 1 {
		t.Fatalf("expected defaults to be persisted once, got %d", repo.setCalls)
	}
}

func TestUpdateOpsAdvancedSettings_PersistsOpenAITokenStatsVisibility(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg := defaultOpsAdvancedSettings()
	cfg.DisplayOpenAITokenStats = true
	cfg.DisplayAlertEvents = false

	updated, err := svc.UpdateOpsAdvancedSettings(context.Background(), cfg)
	if err != nil {
		t.Fatalf("UpdateOpsAdvancedSettings() error = %v", err)
	}
	if !updated.DisplayOpenAITokenStats {
		t.Fatalf("DisplayOpenAITokenStats = false, want true")
	}
	if updated.DisplayAlertEvents {
		t.Fatalf("DisplayAlertEvents = true, want false")
	}

	reloaded, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() after update error = %v", err)
	}
	if !reloaded.DisplayOpenAITokenStats {
		t.Fatalf("reloaded DisplayOpenAITokenStats = false, want true")
	}
	if reloaded.DisplayAlertEvents {
		t.Fatalf("reloaded DisplayAlertEvents = true, want false")
	}
}

func TestGetOpsAdvancedSettings_BackfillsNewDisplayFlagsFromDefaults(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	legacyCfg := map[string]any{
		"data_retention": map[string]any{
			"cleanup_enabled":               false,
			"cleanup_schedule":              "0 2 * * *",
			"error_log_retention_days":      30,
			"minute_metrics_retention_days": 30,
			"hourly_metrics_retention_days": 30,
		},
		"aggregation": map[string]any{
			"aggregation_enabled": false,
		},
		"ignore_count_tokens_errors":    true,
		"ignore_context_canceled":       true,
		"ignore_no_available_accounts":  false,
		"ignore_invalid_api_key_errors": false,
		"auto_refresh_enabled":          false,
		"auto_refresh_interval_seconds": 30,
	}
	raw, err := json.Marshal(legacyCfg)
	if err != nil {
		t.Fatalf("marshal legacy config: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if cfg.DisplayOpenAITokenStats {
		t.Fatalf("DisplayOpenAITokenStats = true, want false default backfill")
	}
	if !cfg.DisplayAlertEvents {
		t.Fatalf("DisplayAlertEvents = false, want true default backfill")
	}
}

func TestGetOpsAdvancedSettings_BackfillsSlowTailLatencyThresholdsFromDefaults(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	legacyCfg := map[string]any{
		"slow_tail_isolation": map[string]any{
			"enabled":               true,
			"window_minutes":        10,
			"min_requests":          3,
			"ttft_p95_ms_threshold": 12000,
			"temp_unsched_minutes":  120,
			"platforms":             []string{"openai"},
			"models":                []string{},
			"group_ids":             []int64{},
			"max_accounts_per_run":  3,
		},
	}
	raw, err := json.Marshal(legacyCfg)
	if err != nil {
		t.Fatalf("marshal legacy config: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if cfg.SlowTailIsolation.DurationP95MsThreshold != 0 {
		t.Fatalf("DurationP95MsThreshold = %d, want 0", cfg.SlowTailIsolation.DurationP95MsThreshold)
	}
	if cfg.SlowTailIsolation.ResponseLatencyP95MsThreshold != 0 {
		t.Fatalf("ResponseLatencyP95MsThreshold = %d, want 0", cfg.SlowTailIsolation.ResponseLatencyP95MsThreshold)
	}
}

func TestUpdateOpsAdvancedSettings_PersistsSlowTailLatencyThresholds(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg := defaultOpsAdvancedSettings()
	cfg.SlowTailIsolation.DurationP95MsThreshold = 90000
	cfg.SlowTailIsolation.ResponseLatencyP95MsThreshold = 80000

	updated, err := svc.UpdateOpsAdvancedSettings(context.Background(), cfg)
	if err != nil {
		t.Fatalf("UpdateOpsAdvancedSettings() error = %v", err)
	}
	if updated.SlowTailIsolation.DurationP95MsThreshold != 90000 {
		t.Fatalf("DurationP95MsThreshold = %d, want 90000", updated.SlowTailIsolation.DurationP95MsThreshold)
	}
	if updated.SlowTailIsolation.ResponseLatencyP95MsThreshold != 80000 {
		t.Fatalf("ResponseLatencyP95MsThreshold = %d, want 80000", updated.SlowTailIsolation.ResponseLatencyP95MsThreshold)
	}

	reloaded, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() after update error = %v", err)
	}
	if reloaded.SlowTailIsolation.DurationP95MsThreshold != 90000 {
		t.Fatalf("reloaded DurationP95MsThreshold = %d, want 90000", reloaded.SlowTailIsolation.DurationP95MsThreshold)
	}
	if reloaded.SlowTailIsolation.ResponseLatencyP95MsThreshold != 80000 {
		t.Fatalf("reloaded ResponseLatencyP95MsThreshold = %d, want 80000", reloaded.SlowTailIsolation.ResponseLatencyP95MsThreshold)
	}
}
