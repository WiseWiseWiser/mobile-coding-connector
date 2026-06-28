package openclaw

import (
	"encoding/json"
	"testing"
)

func TestRenderGatewayConfigSocketMode(t *testing.T) {
	requireMention := true
	cfg := &Config{
		GatewayPort: 18789,
		Workspace:   "~/.openclaw/workspace",
		Model:       "anthropic/claude-sonnet-4-6",
		Slack: &SlackConfig{
			Enabled:        true,
			Mode:           slackModeSocket,
			DMPolicy:       "pairing",
			AllowFrom:      []string{"U123"},
			RequireMention: &requireMention,
		},
	}

	data, err := RenderGatewayConfig(cfg)
	if err != nil {
		t.Fatalf("RenderGatewayConfig() error = %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(data, &rendered); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	gateway, ok := rendered["gateway"].(map[string]any)
	if !ok || int(gateway["port"].(float64)) != 18789 {
		t.Fatalf("gateway.port missing or wrong: %v", rendered["gateway"])
	}

	channels := rendered["channels"].(map[string]any)
	slack := channels["slack"].(map[string]any)
	if slack["enabled"] != true || slack["mode"] != slackModeSocket {
		t.Fatalf("slack config = %v", slack)
	}

	appToken := slack["appToken"].(map[string]any)
	if appToken["id"] != "SLACK_APP_TOKEN" {
		t.Fatalf("appToken = %v", appToken)
	}
}