package openclaw

import (
	"fmt"
	"sync"
	"time"

	"github.com/xhd2015/agent-pro/agent/exec/tool_resolve"
	"github.com/xhd2015/ai-critic/server/tools"
)

const mockPID = 4242

type Status struct {
	Running          bool   `json:"running"`
	GatewayPort      int    `json:"gateway_port"`
	Mocked           bool   `json:"mocked"`
	MockPID          int    `json:"mock_pid,omitempty"`
	StartedAt        string `json:"started_at,omitempty"`
	GeneratedConfig  string `json:"generated_config,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	SlackEnabled     bool   `json:"slack_enabled"`
	SlackMode        string `json:"slack_mode,omitempty"`
}

type DryRunResult struct {
	GatewayPort     int      `json:"gateway_port"`
	Workspace       string   `json:"workspace,omitempty"`
	Model           string   `json:"model,omitempty"`
	SlackEnabled    bool     `json:"slack_enabled"`
	SlackMode       string   `json:"slack_mode,omitempty"`
	NodeInstalled   bool     `json:"node_installed"`
	OpenClawInstalled bool   `json:"openclaw_installed"`
	Checks          []string `json:"checks"`
	Issues          []string `json:"issues,omitempty"`
	Mocked          bool     `json:"mocked"`
}

var (
	inst     *Manager
	instOnce sync.Once
)

type Manager struct {
	mu sync.Mutex
}

func GetManager() *Manager {
	instOnce.Do(func() {
		inst = &Manager{}
	})
	return inst
}

func (m *Manager) Status() *Status {
	cfg, _ := LoadConfig()
	state, _ := LoadState()
	if cfg == nil {
		cfg = defaultConfig()
	}
	if state == nil {
		state = &RuntimeState{}
	}

	status := &Status{
		Running:     state.Running,
		GatewayPort: cfg.GatewayPort,
		Mocked:      true,
		MockPID:     state.MockPID,
		StartedAt:   state.StartedAt,
		LastError:   state.LastError,
	}
	if state.Running {
		status.GeneratedConfig = generatedConfigPath()
	}
	if cfg.Slack != nil {
		status.SlackEnabled = cfg.Slack.Enabled
		status.SlackMode = cfg.Slack.Mode
	}
	return status
}

func (m *Manager) DryRun() (*DryRunResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dryRunLocked()
}

func (m *Manager) dryRunLocked() (*DryRunResult, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = defaultConfig()
	}

	dr := &DryRunResult{
		GatewayPort: cfg.GatewayPort,
		Workspace:   cfg.Workspace,
		Model:       cfg.Model,
		Mocked:      true,
	}
	if cfg.Slack != nil {
		dr.SlackEnabled = cfg.Slack.Enabled
		dr.SlackMode = cfg.Slack.Mode
	}

	dr.NodeInstalled = tool_resolve.IsAvailable("node")
	if dr.NodeInstalled {
		dr.Checks = append(dr.Checks, "node is installed")
	} else {
		dr.Issues = append(dr.Issues, "node is not installed")
	}

	dr.OpenClawInstalled = tool_resolve.IsAvailable("openclaw")
	if dr.OpenClawInstalled {
		dr.Checks = append(dr.Checks, "openclaw CLI is installed")
	} else if hint := tools.GetInstallHint("openclaw"); hint != "" {
		dr.Issues = append(dr.Issues, "openclaw CLI is not installed")
		dr.Checks = append(dr.Checks, "install hint: "+hint)
	} else {
		dr.Issues = append(dr.Issues, "openclaw CLI is not installed")
	}

	if err := ValidateStartConfig(cfg); err != nil {
		dr.Issues = append(dr.Issues, err.Error())
	} else {
		dr.Checks = append(dr.Checks, "slack configuration is valid")
	}

	dr.Checks = append(dr.Checks, "gateway integration is mocked (no real openclaw process)")
	if dr.SlackEnabled {
		dr.Checks = append(dr.Checks, "slack socket mode connection is mocked")
	}

	return dr, nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startLocked()
}

func (m *Manager) startLocked() error {
	state, err := LoadState()
	if err != nil {
		return err
	}
	if state.Running {
		return newError(ErrAlreadyRunning, "openclaw gateway is already running")
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	if err := ValidateStartConfig(cfg); err != nil {
		return newError(ErrBadRequest, err.Error())
	}

	generatedPath, err := WriteGeneratedConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to write generated config: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state = &RuntimeState{
		Running:   true,
		StartedAt: now,
		MockPID:   mockPID,
		Mocked:    true,
	}
	if err := SaveState(state); err != nil {
		return err
	}

	_ = generatedPath
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := LoadState()
	if err != nil {
		return err
	}
	if !state.Running {
		return nil
	}

	state.Running = false
	state.MockPID = 0
	state.StartedAt = ""
	state.Mocked = true
	return SaveState(state)
}