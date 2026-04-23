package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const agentHelp = `Usage: remote-agent agent <subcommand> [args...]

Manage custom agents defined on the remote ai-critic server.

Subcommands:
  list
      List custom agents.

  show <agent-id>
      Show one custom agent as JSON, including its system prompt when present.

  add [<agent-id>] [options...]
      Create a custom agent. If <agent-id> is omitted, the server derives one
      from --name when possible.

  delete <agent-id>
  remove <agent-id>
      Delete a custom agent.

  sessions <agent-id>
      List saved sessions for a custom agent.

  run <agent-id> [--project <dir>] [--resume <session-id|latest>] [--wait <duration>]
      Start a new agent session, or resume an existing saved session.
`

const agentAddHelp = `Usage: remote-agent agent add [<agent-id>] [options...]

Create a custom agent on the remote server.

Options:
  --name NAME                Agent display name.
  --description TEXT         Short agent description.
  --mode MODE                Agent mode: primary or subagent.
  --model MODEL              Default model override.
  --template ID              Built-in template to start from: build, plan,
                             refactor, debug.
  --tool NAME                Enable a tool. Repeat for multiple tools.
  --permission RULE          Tool permission rule like bash=deny or edit=ask.
                             Repeat for multiple rules.
  --prompt TEXT              Inline system prompt content.
  --prompt-file PATH         Load system prompt content from a local file.
  -h, --help                 Show this help message.

Examples:
  remote-agent agent add build-review --template build --name "Build Review"
  remote-agent agent add debug-helper --mode subagent --tool read --tool grep --tool bash
  remote-agent agent add plan-auditor --template plan --prompt-file ./SYSTEM_PROMPT.md
`

const agentDeleteHelp = `Usage: remote-agent agent delete <agent-id>
       remote-agent agent remove <agent-id>

Delete a custom agent from the remote server.
`

const agentSessionsHelp = `Usage: remote-agent agent sessions <agent-id>

List saved sessions for a custom agent.
`

const agentRunHelp = `Usage: remote-agent agent run <agent-id> [--project <dir>] [--resume <session-id|latest>] [--wait <duration>]

Start a remote custom-agent session. By default this creates a new session.
Use --resume to restart a saved session record instead.

Options:
  --project DIR              Project directory on the remote server. Required
                             for new sessions. Optional when resuming.
  --resume SESSION           Resume the given saved session ID. Use "latest"
                             to pick the most recent saved session automatically.
  --wait DURATION            Wait for the proxied opencode server to answer
                             before returning. Default: 15s. Use 0s to skip.
  -h, --help                 Show this help message.

Examples:
  remote-agent agent run build-review --project ~/work/repo
  remote-agent agent run build-review --resume latest
  remote-agent agent run build-review --resume build-review-1740000000000 --wait 30s
`

func runAgent(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(agentHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runAgentList(resolve, rest)
	case "show", "get":
		return runAgentShow(resolve, rest)
	case "add", "create":
		return runAgentAdd(resolve, rest)
	case "delete", "remove":
		return runAgentDelete(resolve, rest)
	case "sessions":
		return runAgentSessions(resolve, rest)
	case "run":
		return runAgentRun(resolve, rest)
	case "-h", "--help":
		fmt.Print(agentHelp)
		return nil
	default:
		return fmt.Errorf("unknown agent subcommand: %s", sub)
	}
}

func runAgentList(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(agentHelp)
			return nil
		}
		return fmt.Errorf("agent list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	agents, err := cli.ListCustomAgents()
	if err != nil {
		return err
	}
	if len(agents) == 0 {
		fmt.Println("No custom agents found.")
		return nil
	}

	fmt.Printf("%-24s  %-12s  %-18s  %s\n", "ID", "MODE", "MODEL", "NAME")
	for _, agent := range agents {
		fmt.Printf("%-24s  %-12s  %-18s  %s\n",
			agent.ID,
			displayOrDash(agent.Mode),
			displayOrDash(agent.Model),
			displayOrDash(agent.Name),
		)
		if agent.Description != "" {
			fmt.Printf("  %s\n", agent.Description)
		}
	}
	return nil
}

func runAgentShow(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print("Usage: remote-agent agent show <agent-id>\n")
			return nil
		}
		return fmt.Errorf("agent show requires exactly 1 argument <agent-id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	agent, err := cli.GetCustomAgent(args[0])
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runAgentAdd(resolve func() (*client.Client, error), args []string) error {
	var (
		name        string
		description string
		mode        string
		model       string
		templateID  string
		prompt      string
		promptFile  string
		tools       []string
		permissions []string
	)

	args, err := flags.
		String("--name", &name).
		String("--description", &description).
		String("--mode", &mode).
		String("--model", &model).
		String("--template", &templateID).
		StringSlice("--tool", &tools).
		StringSlice("--permission", &permissions).
		String("--prompt", &prompt).
		String("--prompt-file", &promptFile).
		Help("-h,--help", agentAddHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("agent add takes at most 1 positional argument [agent-id], got %v", args)
	}
	if prompt != "" && promptFile != "" {
		return fmt.Errorf("use either --prompt or --prompt-file, not both")
	}
	if mode != "" && mode != "primary" && mode != "subagent" {
		return fmt.Errorf("invalid --mode %q: expected primary or subagent", mode)
	}

	agentID := firstArg(args)
	if agentID == "" && strings.TrimSpace(name) == "" {
		return fmt.Errorf("agent add requires either <agent-id> or --name")
	}

	systemPrompt := strings.TrimSpace(prompt)
	if promptFile != "" {
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return fmt.Errorf("read --prompt-file: %w", err)
		}
		systemPrompt = string(data)
	}

	req := client.CreateCustomAgentRequest{
		ID:           agentID,
		Name:         strings.TrimSpace(name),
		Description:  strings.TrimSpace(description),
		Mode:         mode,
		Model:        strings.TrimSpace(model),
		Template:     strings.TrimSpace(templateID),
		SystemPrompt: systemPrompt,
	}

	toolMap := buildToolMap(tools)
	if len(toolMap) > 0 {
		req.Tools = toolMap
	}
	permissionMap, err := parsePermissionRules(permissions)
	if err != nil {
		return err
	}
	if len(permissionMap) > 0 {
		req.Permissions = permissionMap
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	agent, err := cli.CreateCustomAgent(req)
	if err != nil {
		return err
	}

	fmt.Printf("Created agent %s (%s)\n", agent.ID, displayOrDash(agent.Name))
	return nil
}

func runAgentDelete(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print(agentDeleteHelp)
			return nil
		}
		return fmt.Errorf("agent delete requires exactly 1 argument <agent-id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	if err := cli.DeleteCustomAgent(args[0]); err != nil {
		return err
	}
	fmt.Printf("Deleted agent %s\n", args[0])
	return nil
}

func runAgentSessions(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print(agentSessionsHelp)
			return nil
		}
		return fmt.Errorf("agent sessions requires exactly 1 argument <agent-id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	sessions, err := cli.ListCustomAgentSessions(args[0])
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Printf("No saved sessions for agent %s.\n", args[0])
		return nil
	}

	fmt.Printf("%-28s  %-10s  %-19s  %-6s  %s\n", "SESSION ID", "STATUS", "CREATED", "PORT", "PROJECT")
	for _, session := range sessions {
		fmt.Printf("%-28s  %-10s  %-19s  %-6d  %s\n",
			session.ID,
			displayOrDash(session.Status),
			formatAgentTime(session.CreatedAt),
			session.Port,
			displayOrDash(session.ProjectDir),
		)
		if session.Error != "" {
			fmt.Printf("  error: %s\n", session.Error)
		}
	}
	return nil
}

func runAgentRun(resolve func() (*client.Client, error), args []string) error {
	var (
		projectDir string
		resume     string
		wait       time.Duration
	)

	wait = 15 * time.Second
	args, err := flags.
		String("--project", &projectDir).
		String("--resume", &resume).
		Duration("--wait", &wait).
		Help("-h,--help", agentRunHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("agent run requires exactly 1 argument <agent-id>")
	}
	agentID := args[0]
	projectDir = strings.TrimSpace(projectDir)

	cli, err := resolve()
	if err != nil {
		return err
	}

	resumeSessionID := strings.TrimSpace(resume)
	if resumeSessionID == "latest" {
		sessions, err := cli.ListCustomAgentSessions(agentID)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			return fmt.Errorf("agent %s has no saved sessions to resume", agentID)
		}
		resumeSessionID = sessions[0].ID
	}

	if resumeSessionID == "" && projectDir == "" {
		return fmt.Errorf("agent run requires --project for a new session")
	}

	res, err := cli.LaunchCustomAgent(agentID, projectDir, resumeSessionID)
	if err != nil {
		return err
	}

	if wait > 0 {
		if err := cli.WaitCustomAgentSessionReady(res.SessionID, wait); err != nil {
			return err
		}
	}

	action := "Started"
	if resumeSessionID != "" {
		action = "Resumed"
	}

	fmt.Printf("%s custom agent session\n", action)
	fmt.Printf("Session ID: %s\n", res.SessionID)
	fmt.Printf("Agent:      %s\n", agentID)
	if projectDir != "" {
		fmt.Printf("Project:    %s\n", projectDir)
	}
	fmt.Printf("Port:       %d\n", res.Port)
	fmt.Printf("URL:        %s\n", res.URL)
	fmt.Printf("Proxy:      %s\n", customAgentProxyBase(cli, res.SessionID))
	return nil
}

func buildToolMap(tools []string) map[string]bool {
	if len(tools) == 0 {
		return nil
	}
	m := make(map[string]bool, len(tools))
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		m[tool] = true
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func parsePermissionRules(rules []string) (map[string]string, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(rules))
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		key, value, ok := strings.Cut(rule, "=")
		if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("invalid --permission %q: expected tool=value", rule)
		}
		m[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}

func customAgentProxyBase(cli *client.Client, sessionID string) string {
	base := strings.TrimRight(cli.Server, "/")
	path := "/api/custom-agents/sessions/" + url.PathEscape(sessionID) + "/proxy"
	return base + path
}

func formatAgentTime(v string) string {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
