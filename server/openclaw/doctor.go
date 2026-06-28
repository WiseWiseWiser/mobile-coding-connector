package openclaw

import (
	"strings"

	"github.com/xhd2015/agent-pro/agent/exec/tool_resolve"
	"github.com/xhd2015/ai-critic/server/tools"
)

type DoctorCheckStatus string

const (
	DoctorOK   DoctorCheckStatus = "ok"
	DoctorWarn DoctorCheckStatus = "warn"
	DoctorFail DoctorCheckStatus = "fail"
	DoctorSkip DoctorCheckStatus = "skip"
)

type DoctorCheck struct {
	ID     string            `json:"id"`
	Layer  string            `json:"layer"`
	Name   string            `json:"name"`
	Status DoctorCheckStatus `json:"status"`
	Detail string            `json:"detail,omitempty"`
	Hint   string            `json:"hint,omitempty"`
}

type DoctorReport struct {
	Healthy bool          `json:"healthy"`
	Mocked  bool          `json:"mocked"`
	Status  *Status       `json:"status,omitempty"`
	Checks  []DoctorCheck `json:"checks"`
}

func (m *Manager) Doctor() *DoctorReport {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = defaultConfig()
	}

	report := &DoctorReport{
		Mocked: true,
		Status: m.Status(),
	}
	report.Checks = append(report.Checks, m.serverDoctorChecks(cfg)...)

	report.Healthy = true
	for _, check := range report.Checks {
		if check.Status == DoctorFail {
			report.Healthy = false
			break
		}
	}
	return report
}

func (m *Manager) serverDoctorChecks(cfg *Config) []DoctorCheck {
	checks := []DoctorCheck{
		{
			ID: "mock_mode", Layer: "server", Name: "integration mode",
			Status: DoctorWarn, Detail: "openclaw gateway and slack are mocked",
			Hint:   "real subprocess and socket mode wiring will replace this scaffold",
		},
	}

	if tool_resolve.IsAvailable("node") {
		checks = append(checks, DoctorCheck{
			ID: "node", Layer: "server", Name: "Node.js",
			Status: DoctorOK, Detail: "node is on PATH",
		})
	} else {
		hint := tools.GetInstallHint("node")
		if hint == "" {
			hint = "install Node.js 22+"
		}
		checks = append(checks, DoctorCheck{
			ID: "node", Layer: "server", Name: "Node.js",
			Status: DoctorFail, Detail: "node is not on PATH", Hint: hint,
		})
	}

	if tool_resolve.IsAvailable("openclaw") {
		checks = append(checks, DoctorCheck{
			ID: "openclaw_cli", Layer: "server", Name: "OpenClaw CLI",
			Status: DoctorOK, Detail: "openclaw is on PATH",
		})
	} else {
		hint := tools.GetInstallHint("openclaw")
		if hint == "" {
			hint = "npm install -g openclaw@latest"
		}
		checks = append(checks, DoctorCheck{
			ID: "openclaw_cli", Layer: "server", Name: "OpenClaw CLI",
			Status: DoctorFail, Detail: "openclaw is not on PATH", Hint: hint,
		})
	}

	if cfg.Slack != nil && cfg.Slack.Enabled {
		if strings.TrimSpace(cfg.Slack.BotToken) == "" || strings.TrimSpace(cfg.Slack.AppToken) == "" {
			checks = append(checks, DoctorCheck{
				ID: "slack_tokens", Layer: "server", Name: "Slack tokens",
				Status: DoctorFail, Detail: "slack is enabled but bot/app tokens are missing",
				Hint:   "remote-agent openclaw config set --slack-bot-token xoxb-... --slack-app-token xapp-...",
			})
		} else {
			checks = append(checks, DoctorCheck{
				ID: "slack_tokens", Layer: "server", Name: "Slack tokens",
				Status: DoctorOK, Detail: "bot and app tokens are configured (stored in openclaw.json)",
			})
		}

		checks = append(checks, DoctorCheck{
			ID: "slack_plugin", Layer: "server", Name: "Slack plugin",
			Status: DoctorWarn, Detail: "plugin install check is mocked",
			Hint:   "future: openclaw plugins install @openclaw/slack",
		})

		checks = append(checks, DoctorCheck{
			ID: "slack_socket", Layer: "server", Name: "Slack socket mode",
			Status: DoctorWarn, Detail: "socket mode connection is mocked",
		})
	} else {
		checks = append(checks, DoctorCheck{
			ID: "slack_enabled", Layer: "server", Name: "Slack channel",
			Status: DoctorSkip, Detail: "slack is disabled",
		})
	}

	state, _ := LoadState()
	if state != nil && state.Running {
		checks = append(checks, DoctorCheck{
			ID: "gateway_running", Layer: "server", Name: "Gateway process",
			Status: DoctorOK, Detail: "mock gateway is running",
		})
	} else {
		checks = append(checks, DoctorCheck{
			ID: "gateway_running", Layer: "server", Name: "Gateway process",
			Status: DoctorWarn, Detail: "mock gateway is not running",
			Hint:   "remote-agent openclaw start",
		})
	}

	generated := generatedConfigPath()
	if _, err := LoadConfig(); err == nil {
		if data, readErr := readGeneratedConfig(generated); readErr == nil && len(data) > 0 {
			checks = append(checks, DoctorCheck{
				ID: "generated_config", Layer: "server", Name: "Generated openclaw.json",
				Status: DoctorOK, Detail: generated,
			})
		} else if state != nil && state.Running {
			checks = append(checks, DoctorCheck{
				ID: "generated_config", Layer: "server", Name: "Generated openclaw.json",
				Status: DoctorWarn, Detail: "generated config is missing",
				Hint:   "remote-agent openclaw start",
			})
		}
	}

	return checks
}

func readGeneratedConfig(path string) ([]byte, error) {
	return readFileIfExists(path)
}