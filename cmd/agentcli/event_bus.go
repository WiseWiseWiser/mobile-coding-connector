package agentcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/server/eventbus"
	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
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
	remain, err := flags.
		StringSlice("--type", &types).
		Bool("--json", &jsonOut).
		Int("--replay", &replay).
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
		Server: server,
		Token:  token,
		Types:  types,
		JSON:   jsonOut,
		Replay: replay,
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
	conn *websocket.Conn
	ctx  context.Context
}

func (c *wsEventBusConn) ReadJSON(v any) error {
	// Unblock on context cancel by closing the conn from a helper if needed.
	// DialContext already respects ctx for dial; for read we set a deadline loop.
	if c.ctx != nil {
		if err := c.ctx.Err(); err != nil {
			return err
		}
		// Short read deadline so we can notice cancellation.
		_ = c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	}
	for {
		err := c.conn.ReadJSON(v)
		if err == nil {
			return nil
		}
		if c.ctx != nil && c.ctx.Err() != nil {
			return c.ctx.Err()
		}
		// Retry on timeout to poll context.
		if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
			_ = c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			continue
		}
		return err
	}
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
