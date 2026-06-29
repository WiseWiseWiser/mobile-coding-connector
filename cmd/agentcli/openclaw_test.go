package agentcli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenClawConfigSetBodyIncludesSlackSecrets(t *testing.T) {
	args := []string{
		"--slack-enabled",
		"--slack-bot-token", "xoxb-test",
		"--slack-app-token", "xapp-test",
	}

	var (
		slackEnabled   bool
		slackBotToken  string
		slackAppToken  string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--slack-enabled":
			slackEnabled = true
		case "--slack-bot-token":
			slackBotToken = args[i+1]
			i++
		case "--slack-app-token":
			slackAppToken = args[i+1]
			i++
		}
	}

	body := map[string]any{
		"slack": map[string]any{
			"enabled":   slackEnabled,
			"bot_token": slackBotToken,
			"app_token": slackAppToken,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	payload := string(data)
	if !strings.Contains(payload, "xoxb-test") || !strings.Contains(payload, "xapp-test") {
		t.Fatalf("payload = %s, want slack tokens", payload)
	}
}

func TestOpenClawMapErrorToCLIAlreadyRunning(t *testing.T) {
	msg := openclawMapErrorToCLI(apiError{Code: "ALREADY_RUNNING", Message: "openclaw gateway is already running"})
	if !strings.Contains(msg, "openclaw stop") {
		t.Fatalf("message = %q, want stop hint", msg)
	}
}