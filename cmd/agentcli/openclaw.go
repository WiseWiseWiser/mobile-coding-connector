package agentcli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const openclawHelp = `Usage: remote-agent openclaw <subcommand> [args...]

Manage the mocked OpenClaw gateway integration on the remote server.
Slack socket mode config is stored in .ai-critic/openclaw.json; the gateway
process and Slack connection are simulated until real integration lands.

Subcommands:
  start [--dry-run]
      Start the mocked OpenClaw gateway and write generated openclaw.json.

  stop
      Stop the mocked gateway.

  status
      Show gateway status (running, port, mock PID).

  config
      Show current configuration (secrets masked).

  config set [options...]
      Update configuration. Secrets are preserved when omitted.

  doctor
      Run prerequisite and mock integration health checks.

Options for config set:
  --enabled / --no-enabled
  --gateway-port PORT
  --workspace PATH
  --auto-start / --no-auto-start
  --model MODEL
  --slack-enabled / --no-slack-enabled
  --slack-bot-token TOKEN
  --slack-app-token TOKEN
  --slack-dm-policy POLICY
  --slack-require-mention / --no-slack-require-mention

Examples:
  remote-agent openclaw config set --slack-enabled --slack-bot-token xoxb-...
  remote-agent openclaw config set --slack-app-token xapp-...
  remote-agent openclaw start
  remote-agent openclaw status
  remote-agent openclaw doctor
`

func runOpenClaw(getClient func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(openclawHelp)
		return nil
	}

	switch args[0] {
	case "start":
		return openclawStart(getClient, args[1:])
	case "stop":
		return openclawStop(getClient, args[1:])
	case "status":
		return openclawStatus(getClient, args[1:])
	case "config":
		if len(args) > 1 && args[1] == "set" {
			return openclawConfigSet(getClient, args[2:])
		}
		return openclawConfigGet(getClient, args[1:])
	case "doctor":
		return openclawDoctor(getClient, args[1:])
	case "-h", "--help":
		fmt.Print(openclawHelp)
		return nil
	default:
		fmt.Print(openclawHelp)
		return nil
	}
}

func openclawStart(getClient func() (*client.Client, error), args []string) error {
	var dryRun bool
	_, err := flags.
		Bool("--dry-run", &dryRun).
		Help("-h,--help", openclawHelp).
		Parse(args)
	if err != nil {
		return err
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	url := "/api/openclaw/start"
	if dryRun {
		url += "?dry_run=true"
	}

	req, err := c.NewRequest("POST", url, strings.NewReader(""))
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start openclaw: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("%s", openclawMapErrorToCLI(*apiErr))
	}

	if dryRun {
		var result struct {
			DryRun *struct {
				GatewayPort       int      `json:"gateway_port"`
				SlackEnabled      bool     `json:"slack_enabled"`
				SlackMode         string   `json:"slack_mode"`
				NodeInstalled     bool     `json:"node_installed"`
				OpenClawInstalled bool     `json:"openclaw_installed"`
				Checks            []string `json:"checks"`
				Issues            []string `json:"issues"`
				Mocked            bool     `json:"mocked"`
			} `json:"dry_run"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}
		if result.DryRun == nil {
			return fmt.Errorf("server did not return dry_run result")
		}
		dr := result.DryRun
		fmt.Println("=== OpenClaw Dry Run ===")
		fmt.Printf("Gateway port:      %d\n", dr.GatewayPort)
		fmt.Printf("Slack enabled:     %v\n", dr.SlackEnabled)
		if dr.SlackMode != "" {
			fmt.Printf("Slack mode:         %s\n", dr.SlackMode)
		}
		fmt.Printf("Node installed:    %v\n", dr.NodeInstalled)
		fmt.Printf("OpenClaw installed:%v\n", dr.OpenClawInstalled)
		fmt.Printf("Mocked:            %v\n", dr.Mocked)
		for _, check := range dr.Checks {
			fmt.Printf("  check: %s\n", check)
		}
		for _, issue := range dr.Issues {
			fmt.Printf("  issue: %s\n", issue)
		}
		return nil
	}

	var result struct {
		Running         bool   `json:"running"`
		Mocked          bool   `json:"mocked"`
		MockPID         int    `json:"mock_pid"`
		GatewayPort     int    `json:"gateway_port"`
		GeneratedConfig string `json:"generated_config"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	fmt.Println("OpenClaw gateway started (mocked).")
	fmt.Printf("Running:           %v\n", result.Running)
	fmt.Printf("Mock PID:          %d\n", result.MockPID)
	fmt.Printf("Gateway port:      %d\n", result.GatewayPort)
	if result.GeneratedConfig != "" {
		fmt.Printf("Generated config:  %s\n", result.GeneratedConfig)
	}
	return nil
}

func openclawStop(getClient func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(openclawHelp)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("openclaw stop takes no arguments, got %v", args)
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("POST", "/api/openclaw/stop", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop openclaw: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to stop: %s", apiErr.Message)
	}

	fmt.Println("OpenClaw gateway stopped.")
	return nil
}

func openclawStatus(getClient func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(openclawHelp)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("openclaw status takes no arguments, got %v", args)
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("GET", "/api/openclaw/status", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to get status: %s", apiErr.Message)
	}

	var status struct {
		Running         bool   `json:"running"`
		GatewayPort     int    `json:"gateway_port"`
		Mocked          bool   `json:"mocked"`
		MockPID         int    `json:"mock_pid"`
		StartedAt       string `json:"started_at"`
		GeneratedConfig string `json:"generated_config"`
		SlackEnabled    bool   `json:"slack_enabled"`
		SlackMode       string `json:"slack_mode"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	fmt.Printf("Running:           %v\n", status.Running)
	fmt.Printf("Mocked:            %v\n", status.Mocked)
	fmt.Printf("Gateway port:      %d\n", status.GatewayPort)
	if status.MockPID > 0 {
		fmt.Printf("Mock PID:          %d\n", status.MockPID)
	}
	if status.StartedAt != "" {
		fmt.Printf("Started at:        %s\n", status.StartedAt)
	}
	fmt.Printf("Slack enabled:     %v\n", status.SlackEnabled)
	if status.SlackMode != "" {
		fmt.Printf("Slack mode:         %s\n", status.SlackMode)
	}
	if status.GeneratedConfig != "" {
		fmt.Printf("Generated config:  %s\n", status.GeneratedConfig)
	}
	return nil
}

func openclawConfigGet(getClient func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(openclawHelp)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("openclaw config takes no arguments, got %v", args)
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("GET", "/api/openclaw/config", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to get config: %s", apiErr.Message)
	}

	fmt.Println(string(data))
	return nil
}

func openclawConfigSet(getClient func() (*client.Client, error), args []string) error {
	var (
		enabled            bool
		noEnabled          bool
		gatewayPort        int
		workspace          string
		autoStart          bool
		noAutoStart        bool
		model              string
		slackEnabled       bool
		noSlackEnabled     bool
		slackBotToken      string
		slackAppToken      string
		slackDMPolicy      string
		slackRequireMention bool
		noSlackRequireMention bool
	)

	_, err := flags.
		Bool("--enabled", &enabled).
		Bool("--no-enabled", &noEnabled).
		Int("--gateway-port", &gatewayPort).
		String("--workspace", &workspace).
		Bool("--auto-start", &autoStart).
		Bool("--no-auto-start", &noAutoStart).
		String("--model", &model).
		Bool("--slack-enabled", &slackEnabled).
		Bool("--no-slack-enabled", &noSlackEnabled).
		String("--slack-bot-token", &slackBotToken).
		String("--slack-app-token", &slackAppToken).
		String("--slack-dm-policy", &slackDMPolicy).
		Bool("--slack-require-mention", &slackRequireMention).
		Bool("--no-slack-require-mention", &noSlackRequireMention).
		Help("-h,--help", openclawHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("openclaw config set takes no positional arguments, got %v", args)
	}

	body := map[string]any{}
	if argsHave("--enabled", args) {
		body["enabled"] = enabled
	}
	if argsHave("--no-enabled", args) {
		body["enabled"] = false
	}
	if gatewayPort > 0 {
		body["gateway_port"] = gatewayPort
	}
	if workspace != "" {
		body["workspace"] = workspace
	}
	if argsHave("--auto-start", args) {
		body["auto_start"] = autoStart
	}
	if argsHave("--no-auto-start", args) {
		body["auto_start"] = false
	}
	if model != "" {
		body["model"] = model
	}

	slack := map[string]any{}
	slackTouched := false
	if argsHave("--slack-enabled", args) {
		slack["enabled"] = slackEnabled
		slackTouched = true
	}
	if argsHave("--no-slack-enabled", args) {
		slack["enabled"] = false
		slackTouched = true
	}
	if slackBotToken != "" {
		slack["bot_token"] = slackBotToken
		slackTouched = true
	}
	if slackAppToken != "" {
		slack["app_token"] = slackAppToken
		slackTouched = true
	}
	if slackDMPolicy != "" {
		slack["dm_policy"] = slackDMPolicy
		slackTouched = true
	}
	if argsHave("--slack-require-mention", args) {
		slack["require_mention"] = slackRequireMention
		slackTouched = true
	}
	if argsHave("--no-slack-require-mention", args) {
		slack["require_mention"] = false
		slackTouched = true
	}
	if slackTouched {
		body["slack"] = slack
	}

	if len(body) == 0 {
		return fmt.Errorf("openclaw config set requires at least one option")
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	bodyData, _ := json.Marshal(body)
	req, err := c.NewRequest("PUT", "/api/openclaw/config", strings.NewReader(string(bodyData)))
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to update config: %s", apiErr.Message)
	}

	fmt.Println(string(data))
	return nil
}

func openclawDoctor(getClient func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(openclawHelp)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("openclaw doctor takes no arguments, got %v", args)
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("GET", "/api/openclaw/doctor", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to run doctor: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to run doctor: %s", apiErr.Message)
	}

	var report struct {
		Healthy bool `json:"healthy"`
		Mocked  bool `json:"mocked"`
		Checks  []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Detail string `json:"detail"`
			Hint   string `json:"hint"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}

	fmt.Printf("Healthy: %v\n", report.Healthy)
	fmt.Printf("Mocked:  %v\n", report.Mocked)
	for _, check := range report.Checks {
		line := fmt.Sprintf("[%s] %s", check.Status, check.Name)
		if check.Detail != "" {
			line += ": " + check.Detail
		}
		fmt.Println(line)
		if check.Hint != "" {
			fmt.Printf("  hint: %s\n", check.Hint)
		}
	}
	if !report.Healthy {
		return fmt.Errorf("openclaw doctor found failing checks")
	}
	return nil
}

func openclawMapErrorToCLI(e apiError) string {
	switch e.Code {
	case "ALREADY_RUNNING":
		return fmt.Sprintf("%s\n  Run: remote-agent openclaw stop", e.Message)
	case "BAD_REQUEST":
		return e.Message
	default:
		return e.Message
	}
}