package agentcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/server/eventbus"
	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	"github.com/xhd2015/less-gen/flags"
)

// EventBusConn is a WebSocket-like connection that delivers JSON Event frames.
type EventBusConn interface {
	ReadJSON(v any) error
	Close() error
}

// EventBusDialFunc dials the event-bus WebSocket (injectable for tests).
type EventBusDialFunc func(ctx context.Context, wsURL string, header http.Header) (EventBusConn, error)

// EventBusListenOpts configures RunEventBusListen.
type EventBusListenOpts struct {
	Server    string
	Token     string
	Types     []string
	JSON      bool
	Replay    int
	// OpenTTY enables opening a local iTerm attach window on agent.tty.started
	// and agent.tty.restarted. Default false (--open-tty flag).
	OpenTTY bool
	// OpenTTYSession is called once per distinct session_id when OpenTTY is set.
	// nil → production iTerm ForceNew running `remote-agent agent-run attach <id>`.
	OpenTTYSession func(sessionID string) error
	DialWS    EventBusDialFunc
	Now       func() time.Time
	MaxEvents int
	Context   context.Context
}

const (
	eventBusRootHelp = `Usage: remote-agent event-bus <command> [args...]

Subscribe to the main-server event notification bus.

Commands:
  listen    Connect to /api/event-bus/ws and print events as logs

Run 'remote-agent event-bus listen --help' for listen flags.
`

	eventBusListenHelp = `Usage: remote-agent event-bus listen [flags]

Connect to the server event-bus WebSocket and print each Event as a log line.

Flags:
  --type T       Only print events of type T (repeatable)
  --json         Emit one NDJSON Event object per line (no ANSI)
  --replay N     Print up to N recent hub events (oldest first) before live
  --open-tty     On agent.tty.started|restarted, open iTerm attach (default: off)

Global options (before event-bus):
  --server URL   Server base URL
  --token TOKEN  Optional Bearer token (empty is ok for open WS)
`
)

// runEventBus dispatches the event-bus subcommand.
func runEventBus(stdout, stderr io.Writer, server, token string, tokenSpecified bool, args []string) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) {
		fmt.Fprint(stdout, strings.TrimRight(eventBusRootHelp, "\n")+"\n")
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "listen":
		return runEventBusListenCLI(stdout, stderr, server, token, tokenSpecified, rest)
	case "-h", "--help":
		fmt.Fprint(stdout, strings.TrimRight(eventBusRootHelp, "\n")+"\n")
		return nil
	default:
		return fmt.Errorf("unknown event-bus subcommand: %s", sub)
	}
}

func runEventBusListenCLI(stdout, stderr io.Writer, server, token string, tokenSpecified bool, args []string) error {
	if wantsHelp(args) {
		fmt.Fprint(stdout, strings.TrimRight(eventBusListenHelp, "\n")+"\n")
		return nil
	}

	var types []string
	var jsonOut bool
	var replay int
	var openTTY bool
	remain, err := flags.
		StringSlice("--type", &types).
		Bool("--json", &jsonOut).
		Int("--replay", &replay).
		Bool("--open-tty", &openTTY).
		HelpFunc("-h,--help", func() {
			fmt.Fprint(stdout, strings.TrimRight(eventBusListenHelp, "\n")+"\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if err == flags.ErrHelp {
			return nil
		}
		return err
	}
	if len(remain) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(remain, " "))
	}

	// Resolve server/token from config when not fully specified (same as other commands).
	if server == "" || (!tokenSpecified && token == "") {
		cli, resolveErr := resolveClient(server, 0, token, tokenSpecified)
		if resolveErr == nil && cli != nil {
			if server == "" {
				server = cli.Server
			}
			if !tokenSpecified && token == "" {
				token = cli.Token
			}
		} else if server == "" {
			return fmt.Errorf("no server specified and no default domain configured. " +
				"Pass --server, or run 'remote-agent config' to add a domain and mark it as default.")
		}
	}

	return RunEventBusListen(stdout, stderr, EventBusListenOpts{
		Server:  server,
		Token:   token,
		Types:   types,
		JSON:    jsonOut,
		Replay:  replay,
		OpenTTY: openTTY,
	})
}

// RunEventBusListen connects to the event-bus WebSocket and prints events until
// MaxEvents is reached (when > 0) or Context is cancelled.
func RunEventBusListen(stdout, stderr io.Writer, opts EventBusListenOpts) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	dial := opts.DialWS
	if dial == nil {
		dial = defaultEventBusDial
	}

	server := strings.TrimRight(strings.TrimSpace(opts.Server), "/")
	if server == "" {
		return fmt.Errorf("event-bus listen: server is required")
	}

	wsURL := eventBusWSURL(server, opts.Replay)
	header := http.Header{}
	if opts.Token != "" {
		header.Set("Authorization", "Bearer "+opts.Token)
	}

	typeFilter := makeTypeFilter(opts.Types)
	printed := 0
	everConnected := false
	backoff := 50 * time.Millisecond

	// Process-local dedupe for --open-tty (no disk).
	openedSessions := make(map[string]struct{})
	openTTYSession := opts.OpenTTYSession
	if opts.OpenTTY && openTTYSession == nil {
		openTTYSession = defaultOpenTTYSession
	}

	for {
		if err := ctx.Err(); err != nil {
			if printed > 0 && opts.MaxEvents > 0 && printed >= opts.MaxEvents {
				return nil
			}
			// Context cancelled after successful partial work is not a hard failure
			// when we already hit MaxEvents; otherwise surface cancel.
			if opts.MaxEvents > 0 && printed >= opts.MaxEvents {
				return nil
			}
			if everConnected && printed > 0 {
				// Prefer nil when the caller cancelled after receiving events and no MaxEvents.
				// Still return ctx.Err for hard cancel with zero prints after connect.
			}
			return err
		}

		conn, err := dial(ctx, wsURL, header)
		if err != nil {
			if ctx.Err() != nil {
				if opts.MaxEvents > 0 && printed >= opts.MaxEvents {
					return nil
				}
				return ctx.Err()
			}
			if everConnected {
				writeWarning(stderr, opts.JSON, fmt.Sprintf("warning: dial failed: %v; reconnecting…", err))
				if err := sleepCtx(ctx, backoff); err != nil {
					return nilOrCtx(opts, printed, err)
				}
				continue
			}
			return fmt.Errorf("event-bus dial: %w", err)
		}
		everConnected = true

		if !opts.JSON {
			line := fmt.Sprintf("connected  server=%s", server)
			fmt.Fprintln(stdout, colorGreen(line))
		}

		// Read loop for this connection.
		connErr := readEventBusConn(ctx, conn, func(ev sharedeb.Event) error {
			if typeFilter != nil && !typeFilter[ev.Type] {
				return nil
			}
			if err := printEvent(stdout, opts.JSON, nowFn, ev); err != nil {
				return err
			}
			// Best-effort open-tty after a printed event; never aborts the listen loop.
			if opts.OpenTTY {
				maybeOpenTTY(stderr, opts.JSON, openTTYSession, openedSessions, ev)
			}
			printed++
			if opts.MaxEvents > 0 && printed >= opts.MaxEvents {
				return errMaxEvents
			}
			return nil
		})
		_ = conn.Close()

		if connErr == errMaxEvents {
			return nil
		}
		if connErr != nil && ctx.Err() != nil {
			if opts.MaxEvents > 0 && printed >= opts.MaxEvents {
				return nil
			}
			// Context cancelled: treat as normal stop after any progress.
			if printed > 0 {
				return nil
			}
			return ctx.Err()
		}
		if connErr != nil {
			writeWarning(stderr, opts.JSON, fmt.Sprintf("warning: disconnected: %v; reconnecting…", connErr))
			if err := sleepCtx(ctx, backoff); err != nil {
				if opts.MaxEvents > 0 && printed >= opts.MaxEvents {
					return nil
				}
				if printed > 0 {
					return nil
				}
				return err
			}
			continue
		}
		// Clean EOF without error — still reconnect unless context done.
		writeWarning(stderr, opts.JSON, "warning: disconnected; reconnecting…")
		if err := sleepCtx(ctx, backoff); err != nil {
			if printed > 0 {
				return nil
			}
			return err
		}
	}
}

var errMaxEvents = fmt.Errorf("max events reached")

func readEventBusConn(ctx context.Context, conn EventBusConn, onEvent func(sharedeb.Event) error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var ev sharedeb.Event
		if err := conn.ReadJSON(&ev); err != nil {
			return err
		}
		if err := onEvent(ev); err != nil {
			return err
		}
	}
}

func printEvent(stdout io.Writer, asJSON bool, nowFn func() time.Time, ev sharedeb.Event) error {
	if asJSON {
		data, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}

	ts := nowFn().Format("15:04:05")
	payload := strings.TrimSpace(string(ev.Payload))
	if payload == "" {
		payload = "{}"
	}
	// Human: HH:MM:SS  <type>  id=<id>  <payload>
	meta := ""
	if ev.ID != "" {
		meta = colorGray("  id="+ev.ID)
	}
	line := fmt.Sprintf("%s  %s%s  %s", ts, ev.Type, meta, payload)
	_, err := fmt.Fprintln(stdout, line)
	return err
}

func writeWarning(stderr io.Writer, jsonMode bool, msg string) {
	if jsonMode {
		fmt.Fprintln(stderr, msg)
		return
	}
	fmt.Fprintln(stderr, colorYellow(msg))
}

// maybeOpenTTY opens a local attach window for agent.tty.started|restarted.
// Best-effort: warnings only; never returns an error to the listen loop.
// started: process-local dedupe per session_id (first open only).
// restarted: always open (user closed window + follow-up should re-open).
func maybeOpenTTY(stderr io.Writer, jsonMode bool, openFn func(string) error, opened map[string]struct{}, ev sharedeb.Event) {
	if openFn == nil {
		return
	}
	switch ev.Type {
	case sharedeb.TypeAgentTTYStarted, sharedeb.TypeAgentTTYRestarted:
	default:
		return
	}
	sessionID := parseEventSessionID(ev.Payload)
	if sessionID == "" {
		writeWarning(stderr, jsonMode, fmt.Sprintf("warning: %s missing session_id; skipping open", ev.Type))
		return
	}
	if ev.Type == sharedeb.TypeAgentTTYStarted {
		if _, seen := opened[sessionID]; seen {
			return
		}
		// Mark before call so concurrent started events share one open attempt.
		opened[sessionID] = struct{}{}
	}
	if err := openFn(sessionID); err != nil {
		writeWarning(stderr, jsonMode, fmt.Sprintf("warning: open tty session %s: %v", sessionID, err))
	}
}

// parseEventSessionID extracts payload.session_id from a JSON Event payload.
func parseEventSessionID(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return ""
	}
	v, ok := m["session_id"]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	default:
		return strings.TrimSpace(fmt.Sprint(s))
	}
}

// defaultOpenTTYSession opens a new iTerm window running remote-agent agent-run attach.
func defaultOpenTTYSession(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("empty session_id")
	}
	bin := resolveRemoteAgentBinary()
	// Quote for shell write-text in iTerm.
	cmdLine := shellQuote(bin) + " agent-run attach " + shellQuote(sessionID)
	// Prefer $HOME so the new window is not stuck in listen's cwd.
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		dir = "/"
	}
	return iterm2.OpenConfig(dir, &iterm2.Config{
		Mode:             iterm2.ModeForceNew,
		FollowUpCommands: []string{cmdLine},
		SafeInputIgnore:  true,
	})
}

// resolveRemoteAgentBinary is the binary the user is already running.
func resolveRemoteAgentBinary() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	if p, err := exec.LookPath("remote-agent"); err == nil {
		return p
	}
	return "remote-agent"
}

// shellQuote returns a single-quoted shell-safe form of s (POSIX).
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if isShellSafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isShellSafe(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '/' || r == '_' || r == '-' || r == '.' || r == ':' || r == '+' || r == '=' {
			continue
		}
		return false
	}
	return s != ""
}

func colorYellow(s string) string {
	return "\033[33m" + s + "\033[0m"
}

func makeTypeFilter(types []string) map[string]bool {
	if len(types) == 0 {
		return nil
	}
	m := make(map[string]bool, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t != "" {
			m[t] = true
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func eventBusWSURL(server string, replay int) string {
	base := httpToWSBase(server)
	u := base + eventbus.SubscribeWSPath
	if replay > 0 {
		return u + "?replay=" + url.QueryEscape(fmt.Sprintf("%d", replay))
	}
	return u
}

func httpToWSBase(server string) string {
	s := strings.TrimRight(strings.TrimSpace(server), "/")
	switch {
	case strings.HasPrefix(s, "https://"):
		return "wss://" + strings.TrimPrefix(s, "https://")
	case strings.HasPrefix(s, "http://"):
		return "ws://" + strings.TrimPrefix(s, "http://")
	default:
		return "ws://" + s
	}
}

func defaultEventBusDial(ctx context.Context, wsURL string, header http.Header) (EventBusConn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, err
	}
	return &wsEventBusConn{conn: conn, ctx: ctx}, nil
}

type wsEventBusConn struct {
	conn       *websocket.Conn
	ctx        context.Context
	cancelOnce sync.Once
}

func (c *wsEventBusConn) ReadJSON(v any) (err error) {
	// Recover gorilla/websocket panic on repeated read after a failed frame.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("websocket read: %v", r)
		}
	}()
	// Do not use short read deadlines: gorilla/websocket marks the connection
	// failed after some deadline errors, which races with Cloudflare idle
	// proxies and causes reconnect thrash. Cancel is handled by a watcher that
	// closes the conn when ctx ends.
	if c.ctx != nil {
		if err := c.ctx.Err(); err != nil {
			return err
		}
		c.startCancelWatcher()
	}
	// Clear any prior deadline so the read blocks until a frame or close.
	_ = c.conn.SetReadDeadline(time.Time{})
	return c.conn.ReadJSON(v)
}

func (c *wsEventBusConn) startCancelWatcher() {
	c.cancelOnce.Do(func() {
		if c.ctx == nil || c.conn == nil {
			return
		}
		go func() {
			<-c.ctx.Done()
			_ = c.conn.Close()
		}()
	})
}

func (c *wsEventBusConn) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func nilOrCtx(opts EventBusListenOpts, printed int, err error) error {
	if opts.MaxEvents > 0 && printed >= opts.MaxEvents {
		return nil
	}
	if printed > 0 {
		return nil
	}
	return err
}
