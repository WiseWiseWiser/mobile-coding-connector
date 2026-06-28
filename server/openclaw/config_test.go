package openclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func useTempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetTestDataDir(dir)
	t.Cleanup(func() { SetTestDataDir("") })
	return dir
}

func TestLoadConfigDefaults(t *testing.T) {
	useTempDataDir(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.GatewayPort != defaultGatewayPort {
		t.Fatalf("GatewayPort = %d, want %d", cfg.GatewayPort, defaultGatewayPort)
	}
}

func TestSaveAndLoadConfigRoundTrip(t *testing.T) {
	useTempDataDir(t)

	enabled := true
	requireMention := false
	cfg := &Config{
		Enabled:     true,
		GatewayPort: 19001,
		Workspace:   "/tmp/workspace",
		AutoStart:   true,
		Model:       "anthropic/claude-sonnet-4-6",
		Slack: &SlackConfig{
			Enabled:        enabled,
			Mode:           slackModeSocket,
			BotToken:       "xoxb-test",
			AppToken:       "xapp-test",
			DMPolicy:       "allowlist",
			AllowFrom:      []string{"U123"},
			RequireMention: &requireMention,
		},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got.GatewayPort != 19001 {
		t.Fatalf("GatewayPort = %d, want 19001", got.GatewayPort)
	}
	if got.Slack == nil || got.Slack.BotToken != "xoxb-test" {
		t.Fatalf("slack bot token not preserved: %+v", got.Slack)
	}
}

func TestMaskConfigSecrets(t *testing.T) {
	cfg := &Config{
		Slack: &SlackConfig{
			BotToken: "xoxb-secret",
			AppToken: "xapp-secret",
		},
	}
	masked := MaskConfig(cfg)
	if masked.Slack.BotToken != "***" || masked.Slack.AppToken != "***" {
		t.Fatalf("MaskConfig() = %+v, want masked secrets", masked.Slack)
	}
}

func TestMergeConfigPreservesSecrets(t *testing.T) {
	base := &Config{
		Slack: &SlackConfig{
			Enabled:  true,
			BotToken: "xoxb-keep",
			AppToken: "xapp-keep",
		},
	}
	enabled := true
	merged := MergeConfig(base, ConfigUpdate{
		Slack: &SlackUpdate{Enabled: &enabled, DMPolicy: "pairing"},
	})
	if merged.Slack.BotToken != "xoxb-keep" || merged.Slack.AppToken != "xapp-keep" {
		t.Fatalf("MergeConfig() dropped secrets: %+v", merged.Slack)
	}
	if merged.Slack.DMPolicy != "pairing" {
		t.Fatalf("DMPolicy = %q, want pairing", merged.Slack.DMPolicy)
	}
}

func TestValidateStartConfigSlackRequirements(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "slack disabled",
			cfg:  &Config{GatewayPort: 18789},
		},
		{
			name: "missing bot token",
			cfg: &Config{
				Slack: &SlackConfig{Enabled: true, AppToken: "xapp-1"},
			},
			wantErr: "slack bot token required",
		},
		{
			name: "missing app token",
			cfg: &Config{
				Slack: &SlackConfig{Enabled: true, BotToken: "xoxb-1"},
			},
			wantErr: "slack app token required",
		},
		{
			name: "unsupported mode",
			cfg: &Config{
				Slack: &SlackConfig{
					Enabled: true, BotToken: "xoxb-1", AppToken: "xapp-1", Mode: "http",
				},
			},
			wantErr: "only socket mode is supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStartConfig(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateStartConfig() error = %v", err)
				}
				return
			}
			if err == nil || !contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateStartConfig() = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexSubstring(s, sub))
}

func indexSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestGeneratedConfigWrittenOnStart(t *testing.T) {
	dir := useTempDataDir(t)
	cfg := &Config{
		GatewayPort: 18789,
		Slack: &SlackConfig{
			Enabled:  true,
			BotToken: "xoxb-test",
			AppToken: "xapp-test",
		},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	m := &Manager{}
	if err := m.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	path := filepath.Join(dir, "openclaw", "openclaw.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(data, &rendered); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	channels, ok := rendered["channels"].(map[string]any)
	if !ok {
		t.Fatalf("channels missing in rendered config: %s", string(data))
	}
	slack, ok := channels["slack"].(map[string]any)
	if !ok {
		t.Fatalf("slack missing in rendered config: %s", string(data))
	}
	if slack["mode"] != slackModeSocket {
		t.Fatalf("slack mode = %v, want socket", slack["mode"])
	}
}