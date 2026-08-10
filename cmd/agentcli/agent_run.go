package agentcli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/xhd2015/agent-pro/pkgs/agentstorage"
	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
	"golang.org/x/term"
)

const agentRunRootHelp = `Usage: remote-agent agent-run <subcommand> [args...]

Remote façade over agent-run session storage on the server.

Subcommands:
  sessions              List agent-run sessions on the remote server
  attach <session-id>   Attach this terminal to a remote agent-run session
  status [session-id]   Show remote agent-run home or multi-layer session status
  resume <session-id>   Resume a finished bound session (library path on server)
  send <session-id> …   Send a follow-up message to a live session
  msg status|cancel     Inspect or cancel a queued message
  snapshot <session-id> Print sanitized TTY snapshot text
  watch <session-id>    Stream readonly TTY output (short)
  kill [--dry-run] <id> Stop a live session TTY
  run [OPTIONS] [prompt]
                        Start or auto-send/resume a session (library path)

Local-only (not available via remote-agent):
  focus, web, assets, pty

Run 'remote-agent agent-run <subcommand> --help' for details.
`

const agentRunSessionsHelp = `Usage: remote-agent agent-run sessions [--json] [--limit N]

List agent-run sessions stored on the remote server (newest first).

Options:
  --json               list sessions as JSON
  --limit N            max sessions to show (default 10; 0 = all)
  -h, --help           show help
`

const agentRunAttachHelp = `Usage: remote-agent agent-run attach <session-id>

Attach this terminal to a live remote agent-run session TTY (binary relay).
Detaching keeps the remote agent child running (detach_keep).

Arguments:
  session-id           Agent-run session id to attach to

Options:
  -h, --help           show help
`

const agentRunStatusHelp = `Usage: remote-agent agent-run status [OPTIONS] [<session-id>]

Show remote agent-run home (bare) or multi-layer session status.

With no arguments, prints the remote agent-run home path.

With a session id, probes storage, process, terminal, runner, and resume
readiness (library path on the server).

Options:
  --json               output multi-layer status as JSON
  -h, --help           show help
`

const agentRunResumeHelp = `Usage: remote-agent agent-run resume [OPTIONS] <session-id> ["followup…"]

Resume a finished agent-run session on the remote server (library path —
no local agent-run binary exec). Requires the session to be bound
(runner_session_id) and the runner to have exited. If still live, use
send or attach instead.

Arguments:
  session-id           Agent-run session id to resume
  followup             optional follow-up prompt after resume

Options:
  --open               after successful resume, attach this terminal to the session
  -h, --help           show help
`

const agentRunSendHelp = `Usage: remote-agent agent-run send [OPTIONS] <session-id> "message"

Send a follow-up message to a live remote agent-run session.

Arguments:
  session-id           Agent-run session id
  message              Message text to enqueue

Options:
  --no-wait            do not wait for delivery
  --no-submit          inject without trailing Enter (when supported)
  -h, --help           show help
`

const agentRunMsgHelp = `Usage: remote-agent agent-run msg <status|cancel> <session-id>/<message-id>

Inspect or cancel a queued follow-up message.

Subcommands:
  status   Print message status (pending|delivered)
  cancel   Cancel a still-queued message

Arguments:
  session-id/message-id   Combined ref, e.g. sess-1/msg_1

Options:
  -h, --help           show help
`

const agentRunSnapshotHelp = `Usage: remote-agent agent-run snapshot <session-id>

Print a sanitized TTY snapshot for a live remote agent-run session.

Arguments:
  session-id           Agent-run session id

Options:
  -h, --help           show help
`

const agentRunWatchHelp = `Usage: remote-agent agent-run watch <session-id>

Stream readonly TTY output for a remote agent-run session (short stream).

Arguments:
  session-id           Agent-run session id

Options:
  -h, --help           show help
`

const agentRunKillHelp = `Usage: remote-agent agent-run kill [OPTIONS] <session-id>

Stop a live remote agent-run session TTY.

Arguments:
  session-id           Agent-run session id

Options:
  --dry-run            report what would be stopped without terminating
  -h, --help           show help
`

const agentRunRunHelp = `Usage: remote-agent agent-run run [OPTIONS] ["prompt"]

Start a new agent-run session or auto-send/resume an existing one on the
remote server (library path — no local agent-run binary exec).

Arguments:
  prompt               Initial prompt (required unless resuming with --session-id
                       in modes that allow empty prompt)

Options:
  --session-id ID      existing session id (for auto-send-or-resume / resume path)
  --dir DIR            workspace directory on the remote host
  --open               open keep-alive TTY and attach interactively after start
  --detach             start keep-alive TTY and exit after registry (prints ids)
  --json               request JSON/NDJSON event stream mode when supported
  --auto-send-or-resume
                       classify live vs resume vs run automatically
  --agent-runner NAME  runner override (e.g. grok, codex)
  --model MODEL        model name
  --new-terminal       NOT available via remote-agent (rejected)
  -h, --help           show help
`

const defaultAgentRunSessionsLimit = 10

func localOnlyError(cmd string) error {
	return fmt.Errorf("%s is not available via remote-agent (local-only: use agent-run %s on the host)", cmd, cmd)
}

// runAgentRunRoot handles top-level `remote-agent agent-run …` (not `agent run`).
func runAgentRunRoot(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(agentRunRootHelp)
		return nil
	}
	switch args[0] {
	case "sessions":
		return runAgentRunSessions(resolve, args[1:])
	case "attach":
		return runAgentRunAttach(resolve, args[1:])
	case "status":
		return runAgentRunStatus(resolve, args[1:])
	case "resume":
		return runAgentRunResume(resolve, args[1:])
	case "send":
		return runAgentRunSend(resolve, args[1:])
	case "msg":
		return runAgentRunMsg(resolve, args[1:])
	case "snapshot":
		return runAgentRunSnapshot(resolve, args[1:])
	case "watch":
		return runAgentRunWatch(resolve, args[1:])
	case "kill":
		return runAgentRunKill(resolve, args[1:])
	case "run":
		return runAgentRunRun(resolve, args[1:])
	case "focus", "web", "assets", "pty":
		return localOnlyError(args[0])
	case "-h", "--help":
		fmt.Print(agentRunRootHelp)
		return nil
	default:
		return fmt.Errorf("unknown agent-run subcommand: %s", args[0])
	}
}

func runAgentRunRun(resolve func() (*client.Client, error), args []string) error {
	// Reject --new-terminal early with a clear remote-only message.
	for _, a := range args {
		if a == "--new-terminal" {
			return fmt.Errorf("--new-terminal is not available via remote-agent (unsupported / local-only iTerm path)")
		}
	}

	var (
		sessionID, dir, agentRunner, model string
		openFlag, detachFlag, jsonFlag     bool
		autoSendOrResume                   bool
	)
	remaining, err := flags.
		String("--session-id", &sessionID).
		String("--dir", &dir).
		String("--agent-runner", &agentRunner).
		String("--model", &model).
		Bool("--open", &openFlag).
		Bool("--detach", &detachFlag).
		Bool("--json", &jsonFlag).
		Bool("--auto-send-or-resume", &autoSendOrResume).
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(agentRunRunHelp, "\n") + "\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		// less-gen may surface unrecognized --new-terminal if not stripped above.
		if strings.Contains(err.Error(), "new-terminal") {
			return fmt.Errorf("--new-terminal is not available via remote-agent (unsupported / local-only)")
		}
		return err
	}

	prompt := strings.TrimSpace(strings.Join(remaining, " "))
	sessionID = strings.TrimSpace(sessionID)
	if prompt == "" && sessionID == "" && !detachFlag {
		return fmt.Errorf("agent-run run requires a prompt or --session-id (see --help)")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	result, err := cli.RunAgentRunSession(client.AgentRunRunOpts{
		SessionID:        sessionID,
		Prompt:           prompt,
		Dir:              strings.TrimSpace(dir),
		Open:             openFlag,
		Detach:           detachFlag,
		JSON:             jsonFlag,
		AutoSendOrResume: autoSendOrResume,
		AgentRunner:      strings.TrimSpace(agentRunner),
		Model:            strings.TrimSpace(model),
	})
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("run returned empty result")
	}
	fmt.Printf("session-id: %s\n", result.SessionID)
	if result.TerminalID != "" {
		fmt.Printf("terminal-id: %s\n", result.TerminalID)
	}

	if openFlag && !detachFlag {
		if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
			return fmt.Errorf("remote-agent agent-run run --open requires an interactive terminal on stdin/stdout to attach")
		}
		return cli.AttachAgentRunSession(result.SessionID, os.Stdin, os.Stdout, nil)
	}
	return nil
}

func runAgentRunSend(resolve func() (*client.Client, error), args []string) error {
	var noWait, noSubmit bool
	remaining, err := flags.Bool("--no-wait", &noWait).
		Bool("--no-submit", &noSubmit).
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(agentRunSendHelp, "\n") + "\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		return err
	}
	if len(remaining) < 2 {
		return fmt.Errorf("agent-run send requires <session-id> and message")
	}
	sessionID := strings.TrimSpace(remaining[0])
	message := strings.TrimSpace(strings.Join(remaining[1:], " "))
	if sessionID == "" || message == "" {
		return fmt.Errorf("agent-run send requires session-id and message")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	msgID, err := cli.SendAgentRunMessage(sessionID, message, client.AgentRunSendOpts{
		NoWait:   noWait,
		NoSubmit: noSubmit,
	})
	if err != nil {
		return err
	}
	fmt.Println(msgID)
	return nil
}

func runAgentRunMsg(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(agentRunMsgHelp)
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		if len(rest) != 1 {
			return fmt.Errorf("msg status requires <session-id>/<message-id>")
		}
		sid, mid, err := parseSessionMsgRef(rest[0])
		if err != nil {
			return err
		}
		cli, err := resolve()
		if err != nil {
			return err
		}
		st, err := cli.AgentRunMsgStatus(sid, mid)
		if err != nil {
			return err
		}
		fmt.Println(st)
		return nil
	case "cancel":
		if len(rest) != 1 {
			return fmt.Errorf("msg cancel requires <session-id>/<message-id>")
		}
		sid, mid, err := parseSessionMsgRef(rest[0])
		if err != nil {
			return err
		}
		cli, err := resolve()
		if err != nil {
			return err
		}
		if err := cli.AgentRunMsgCancel(sid, mid); err != nil {
			return err
		}
		fmt.Printf("cancelled %s/%s\n", sid, mid)
		return nil
	case "-h", "--help":
		fmt.Print(agentRunMsgHelp)
		return nil
	default:
		return fmt.Errorf("unknown msg subcommand: %s", sub)
	}
}

func parseSessionMsgRef(ref string) (sessionID, msgID string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" || !strings.Contains(ref, "/") {
		return "", "", fmt.Errorf("invalid message ref %q: require session-id/message-id format", ref)
	}
	i := strings.Index(ref, "/")
	sessionID = strings.TrimSpace(ref[:i])
	msgID = strings.TrimSpace(ref[i+1:])
	if sessionID == "" || msgID == "" || strings.Contains(msgID, "/") {
		return "", "", fmt.Errorf("invalid message ref %q: require session-id/message-id format", ref)
	}
	return sessionID, msgID, nil
}

func runAgentRunSnapshot(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(agentRunSnapshotHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("agent-run snapshot requires exactly 1 argument <session-id>")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" {
		return fmt.Errorf("agent-run snapshot requires a session-id")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	text, err := cli.AgentRunSnapshot(sessionID)
	if err != nil {
		return err
	}
	fmt.Print(text)
	if !strings.HasSuffix(text, "\n") {
		fmt.Println()
	}
	return nil
}

func runAgentRunWatch(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(agentRunWatchHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("agent-run watch requires exactly 1 argument <session-id>")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" {
		return fmt.Errorf("agent-run watch requires a session-id")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	return cli.AgentRunWatch(sessionID, osStdout())
}

func runAgentRunKill(resolve func() (*client.Client, error), args []string) error {
	var dryRun bool
	remaining, err := flags.Bool("--dry-run", &dryRun).
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(agentRunKillHelp, "\n") + "\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		return err
	}
	if len(remaining) != 1 {
		return fmt.Errorf("agent-run kill requires a session-id")
	}
	sessionID := strings.TrimSpace(remaining[0])
	if sessionID == "" {
		return fmt.Errorf("agent-run kill requires a session-id")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	report, err := cli.AgentRunKill(sessionID, dryRun)
	if err != nil {
		return err
	}
	fmt.Println(report)
	return nil
}

func runAgentRunStatus(resolve func() (*client.Client, error), args []string) error {
	var jsonFlag bool
	remaining, err := flags.Bool("--json", &jsonFlag).
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(agentRunStatusHelp, "\n") + "\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		return err
	}
	if len(remaining) > 1 {
		return fmt.Errorf("status accepts at most one positional <session-id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	if len(remaining) == 0 {
		home, err := cli.GetAgentRunHome()
		if err != nil {
			return err
		}
		fmt.Printf("home: %s\n", home)
		return nil
	}

	sessionID := strings.TrimSpace(remaining[0])
	if sessionID == "" {
		return fmt.Errorf("status requires a non-empty session-id")
	}
	report, err := cli.StatusAgentRunSession(sessionID)
	if err != nil {
		return err
	}
	if jsonFlag {
		enc := json.NewEncoder(osStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	printAgentRunStatusHuman(report)
	return nil
}

func printAgentRunStatusHuman(r *client.AgentRunStatusReport) {
	if r == nil {
		return
	}
	fmt.Printf("session:   %s\n", r.Session)
	fmt.Printf("status:    %s\n", r.Status)
	if r.Workspace != "" {
		fmt.Printf("workspace: %s\n", r.Workspace)
	}
	fmt.Printf("process:   %s\n", r.Process.Status)
	fmt.Printf("terminal:  %s\n", r.Terminal.Status)
	fmt.Printf("runner:    %s\n", r.Runner.Status)
	fmt.Printf("resume:    ready=%v %s\n", r.Resume.Ready, r.Resume.Reason)
}

func runAgentRunResume(resolve func() (*client.Client, error), args []string) error {
	var openFlag bool
	remaining, err := flags.Bool("--open", &openFlag).
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(agentRunResumeHelp, "\n") + "\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("agent-run resume requires a session-id")
	}
	sessionID := strings.TrimSpace(remaining[0])
	if sessionID == "" {
		return fmt.Errorf("agent-run resume requires a session-id")
	}
	prompt := ""
	if len(remaining) > 1 {
		prompt = strings.TrimSpace(strings.Join(remaining[1:], " "))
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	if err := cli.ResumeAgentRunSession(sessionID, client.AgentRunResumeOpts{
		Open:   openFlag,
		Prompt: prompt,
	}); err != nil {
		return err
	}
	fmt.Printf("resumed session %s\n", sessionID)

	if openFlag {
		// After successful resume, attach interactively (same gate as agent-run attach).
		if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
			return fmt.Errorf("remote-agent agent-run resume --open requires an interactive terminal on stdin/stdout to attach")
		}
		return cli.AttachAgentRunSession(sessionID, os.Stdin, os.Stdout, nil)
	}
	return nil
}

func runAgentRunAttach(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(agentRunAttachHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("agent-run attach requires exactly 1 argument <session-id>")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" {
		return fmt.Errorf("agent-run attach requires a session-id")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("remote-agent agent-run attach requires an interactive terminal on stdin/stdout")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	return cli.AttachAgentRunSession(sessionID, os.Stdin, os.Stdout, nil)
}

func runAgentRunSessions(resolve func() (*client.Client, error), args []string) error {
	var jsonFlag bool
	limit := defaultAgentRunSessionsLimit
	remaining, err := flags.Bool("--json", &jsonFlag).
		Int("--limit", &limit).
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(agentRunSessionsHelp, "\n") + "\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		return err
	}
	if len(remaining) > 0 {
		return fmt.Errorf("agent-run sessions takes no positional arguments")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	sessions, total, err := cli.ListAgentRunSessions(limit)
	if err != nil {
		return err
	}
	if sessions == nil {
		sessions = []client.AgentRunSession{}
	}

	if jsonFlag {
		out := map[string]any{"sessions": sessions}
		enc := json.NewEncoder(osStdout())
		return enc.Encode(out)
	}
	return printAgentRunSessionsHuman(sessions, total, limit)
}

func printAgentRunSessionsHuman(list []client.AgentRunSession, total, limit int) error {
	now := time.Now()
	tw := tabwriter.NewWriter(osStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SESSION_ID\tRUNNER\tSTATUS\tUPDATED")
	for _, s := range list {
		ts := s.UpdatedAt
		if ts == "" {
			ts = s.CreatedAt
		}
		updated := agentstorage.FormatRelativeAge(now, parseAgentRunSessionTime(ts))
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.SessionID, s.Runner, s.Status, updated)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if limit > 0 && total > limit {
		fmt.Fprintf(osStdout(), "(showing %d of %d; use --limit N or --limit 0 for all)\n", limit, total)
	}
	return nil
}

func parseAgentRunSessionTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
