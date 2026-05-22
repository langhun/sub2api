package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name      string
		portRange string
		wantStart int
		wantEnd   int
	}{
		{
			name:      "valid range",
			portRange: "21080-21180",
			wantStart: 21080,
			wantEnd:   21180,
		},
		{
			name:      "single port",
			portRange: "8080-8080",
			wantStart: 8080,
			wantEnd:   8080,
		},
		{
			name:      "invalid format",
			portRange: "8080",
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "invalid numbers",
			portRange: "abc-def",
			wantStart: 0,
			wantEnd:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := parsePortRange(tt.portRange)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("parsePortRange(%q) = (%d, %d), want (%d, %d)",
					tt.portRange, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestValidateSubscriptionURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https url",
			url:     "https://example.com/subscription",
			wantErr: false,
		},
		{
			name:    "http url should fail",
			url:     "http://example.com/subscription",
			wantErr: true,
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
		{
			name:    "relative url",
			url:     "/subscription",
			wantErr: true,
		},
		{
			name:    "invalid url",
			url:     "not a url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubscriptionURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSubscriptionURL(%q) error = %v, wantErr %v",
					tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDataDir(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		dataDir string
		setup   func() string
		wantErr bool
	}{
		{
			name:    "valid writable directory",
			dataDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "empty path",
			dataDir: "",
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			dataDir: filepath.Join(tmpDir, "nonexistent"),
			wantErr: true,
		},
		{
			name: "file instead of directory",
			setup: func() string {
				f := filepath.Join(tmpDir, "testfile")
				if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
					t.Fatalf("write test file: %v", err)
				}
				return f
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := tt.dataDir
			if tt.setup != nil {
				dataDir = tt.setup()
			}
			err := validateDataDir(dataDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDataDir(%q) error = %v, wantErr %v",
					dataDir, err, tt.wantErr)
			}
		})
	}
}

func TestMihomoConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		config  ProxySubscriptionMihomoConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid mihomo config",
			config: ProxySubscriptionMihomoConfig{
				Enabled:           true,
				MihomoBin:         "embedded",
				DataDir:           tmpDir,
				ListenerHost:      "127.0.0.1",
				ListenerPortRange: "21080-21180",
			},
			wantErr: false,
		},
		{
			name: "invalid port range - start > end",
			config: ProxySubscriptionMihomoConfig{
				Enabled:           true,
				MihomoBin:         "embedded",
				DataDir:           tmpDir,
				ListenerHost:      "127.0.0.1",
				ListenerPortRange: "21180-21080",
			},
			wantErr: true,
			errMsg:  "listener_port_range invalid",
		},
		{
			name: "invalid port range - out of bounds",
			config: ProxySubscriptionMihomoConfig{
				Enabled:           true,
				MihomoBin:         "embedded",
				DataDir:           tmpDir,
				ListenerHost:      "127.0.0.1",
				ListenerPortRange: "21080-70000",
			},
			wantErr: true,
			errMsg:  "listener_port_range invalid",
		},
		{
			name: "missing listener host",
			config: ProxySubscriptionMihomoConfig{
				Enabled:           true,
				MihomoBin:         "embedded",
				DataDir:           tmpDir,
				ListenerHost:      "",
				ListenerPortRange: "21080-21180",
			},
			wantErr: true,
			errMsg:  "listener_host is required",
		},
		{
			name: "missing mihomo bin",
			config: ProxySubscriptionMihomoConfig{
				Enabled:           true,
				MihomoBin:         "",
				DataDir:           tmpDir,
				ListenerHost:      "127.0.0.1",
				ListenerPortRange: "21080-21180",
			},
			wantErr: true,
			errMsg:  "mihomo_bin is required",
		},
		{
			name: "narrow port range",
			config: ProxySubscriptionMihomoConfig{
				Enabled:           true,
				MihomoBin:         "embedded",
				DataDir:           tmpDir,
				ListenerHost:      "127.0.0.1",
				ListenerPortRange: "21080-21085",
			},
			wantErr: false, // 只是警告,不是错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMihomoConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMihomoConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateMihomoConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// validateMihomoConfig 提取的 Mihomo 配置验证逻辑
func validateMihomoConfig(c *ProxySubscriptionMihomoConfig) error {
	if !c.Enabled {
		return nil
	}

	// 1. 端口范围验证
	start, end := parsePortRange(c.ListenerPortRange)
	if start <= 0 || end < start || end > 65535 {
		return fmt.Errorf("proxy_subscription_mihomo.listener_port_range invalid: must be in format 'start-end' with valid port numbers (1-65535)")
	}
	// 窄端口范围只是警告,不返回错误

	// 2. 监听地址验证
	if strings.TrimSpace(c.ListenerHost) == "" {
		return fmt.Errorf("proxy_subscription_mihomo.listener_host is required when enabled")
	}

	// 3. 数据目录验证
	if strings.TrimSpace(c.DataDir) != "" {
		if err := validateDataDir(c.DataDir); err != nil {
			return fmt.Errorf("proxy_subscription_mihomo.data_dir: %w", err)
		}
	}

	// 4. Mihomo 二进制路径验证
	mihomoBin := strings.TrimSpace(c.MihomoBin)
	if mihomoBin == "" {
		return fmt.Errorf("proxy_subscription_mihomo.mihomo_bin is required when enabled")
	}
	if mihomoBin != "embedded" {
		// 验证自定义二进制文件是否存在且可执行
		info, err := os.Stat(mihomoBin)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("proxy_subscription_mihomo.mihomo_bin: file does not exist: %s", mihomoBin)
			}
			return fmt.Errorf("proxy_subscription_mihomo.mihomo_bin: cannot access file: %w", err)
		}
		if info.IsDir() {
			return fmt.Errorf("proxy_subscription_mihomo.mihomo_bin: path is a directory, not a file: %s", mihomoBin)
		}
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
