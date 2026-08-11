# Remote-Agent event-bus listen Doctests

L2 in-process doctests for `remote-agent event-bus listen`: connect to the
main-server event-bus WebSocket (`GET /api/event-bus/ws`), print events as logs,
optional `--json` / `--type` / `--replay` / `--open-tty`, reconnect with `warning:`.

PHASE 2 classic TDD: `--open-tty` + injectable open hook + process-local
session dedupe are **new** (expect RED until implementer). Existing listen /
help / reject leaves stay GREEN once `OpenTTY` fields are scaffolded (default
off). Server hub + `RegisterSubscribeWS` for real-WS leaves; reconnect and
open-tty streams use injectable dialers.

# DSN (Domain Specific Notion)

CLI client that subscribes to the server event bus over WebSocket, prints each
Event as a log line (human or NDJSON), optionally opens a local iTerm attach
window on new agent TTY sessions, with reconnect and open-failure warnings.

**Participants**

- **agentcli.RunWithWriters** — top-level / `event-bus` help and reject paths
  with injected stdout/stderr (no `os.Stdout` reassignment; no product binary e2e).
- **agentcli.RunEventBusListen** — injectable listen core: writers +
  `EventBusListenOpts` (Server, Token, Types, JSON, Replay, OpenTTY,
  OpenTTYSession, DialWS, Now, MaxEvents, Context).
- **EventBusListenOpts.DialWS** — optional fake dialer for drop-once reconnect
  and deterministic streams without a live hub.
- **EventBusListenOpts.OpenTTYSession** — L2 inject for open-on-TTY; when nil
  and OpenTTY, production opens iTerm ForceNew running
  `remote-agent agent-run attach <session_id>` (resolve binary via `os.Args[0]`
  or lookpath). Tests always inject a recorder — no real osascript/iTerm.
- **server/eventbus Hub + RegisterSubscribeWS** — real ephemeral mux for
  one-event / filter / replay leaves when DialWS is nil.
- **Shared Event** — `dot-pkgs/go-pkgs/eventbus` JSON envelope; TTY payload
  carries `session_id` (plus optional runner/workspace from agent-run).

**Behaviors**

- Help: top-level lists `event-bus`; `event-bus --help` lists `listen`;
  `event-bus listen --help` documents `--type`, `--json`, `--replay`,
  `--open-tty` (default off).
- Human listen: green `connected  server=…` then lines
  `HH:MM:SS  <type>  …` (gray meta optional); yellow `warning:` on disconnect
  before reconnect.
- `--json`: one NDJSON Event per line on stdout; **no ANSI**.
- Token optional (empty Bearer / omit ok for open WS).
- `--type T` (repeatable): only matching event types are printed.
- `--replay N`: up to N recent hub events (oldest first) before live frames.
- `--open-tty`: on each **printed** `agent.tty.started` with non-empty
  payload `session_id`, call OpenTTYSession once per distinct session_id
  (process-local dedupe). Missing session_id → `warning:` stderr, keep
  listening. Open error → `warning:` stderr, continue (MaxEvents still exits 0).
  Other event types (e.g. `seatalk.message.received`) never open. Without
  `--open-tty` / OpenTTY=false → never open.
- Unknown `event-bus` subcommand → non-zero + `Error:`.

## Version

0.0.2

## Decision Tree

```
[remote-agent event-bus]
 |
 +-- help/                                   (GROUP) help surfaces
 |    +-- top-level-lists-event-bus/         (LEAF)  remote-agent --help lists event-bus
 |    +-- event-bus-root/                    (LEAF)  event-bus --help lists listen
 |    +-- listen/                            (LEAF)  event-bus listen --help shows flags
 |
 +-- listen/                                 (GROUP) streaming listen (injectable core)
 |    +-- human/                             (GROUP) pretty human lines
 |    |    +-- one-event/                    (LEAF)  real hub WS → connected + human line
 |    |    +-- empty-token-ok/               (LEAF)  Token="" still receives event
 |    +-- json/                              (GROUP) --json NDJSON
 |    |    +-- one-event/                    (LEAF)  one JSON line, no ANSI
 |    +-- reconnect/                         (GROUP) disconnect mid-stream
 |    |    +-- warning-and-retry/            (LEAF)  dialer drops once → warning: + retry
 |    +-- filter/                            (GROUP) --type filter
 |    |    +-- excludes-non-matching/        (LEAF)  non-matching type suppressed
 |    +-- replay/                            (GROUP) --replay N
 |    |    +-- recent-then-live/             (LEAF)  N recent then one live event
 |    +-- open-tty/                          (GROUP) --open-tty / OpenTTYSession
 |         +-- enabled/                      (GROUP) OpenTTY=true + inject hook
 |         |    +-- opens-on-tty-started/    (LEAF)  agent.tty.started → open once
 |         |    +-- dedupe-same-session/     (LEAF)  same session_id → no 2nd open
 |         |    +-- missing-session-id-warns/(LEAF)  no session_id → warning; no open
 |         |    +-- open-failure-warns/      (LEAF)  open err → warning; exit 0
 |         |    +-- ignores-non-tty-event/   (LEAF)  seatalk.* → never open
 |         +-- disabled/                     (GROUP) OpenTTY=false (default)
 |              +-- never-opens/             (LEAF)  tty.started without flag → no open
 |
 +-- rejected/                               (GROUP) CLI surface errors
      +-- unknown-subcommand/                (LEAF)  event-bus foo → Error
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/top-level-lists-event-bus` | Top-level help mentions `event-bus` |
| 2 | `help/event-bus-root` | `event-bus --help` lists `listen` |
| 3 | `help/listen` | `event-bus listen --help` documents flags incl. `--open-tty` |
| 4 | `listen/human/one-event` | Connect + one hub event → human stdout |
| 5 | `listen/human/empty-token-ok` | Empty token still connects and prints |
| 6 | `listen/json/one-event` | `--json` → one NDJSON line, no ANSI |
| 7 | `listen/reconnect/warning-and-retry` | Injectable drop-once → `warning:` + recover |
| 8 | `listen/filter/excludes-non-matching` | `--type` drops non-matching events |
| 9 | `listen/replay/recent-then-live` | `--replay N` prints recent then live |
| 10 | `listen/open-tty/enabled/opens-on-tty-started` | OpenTTY + tty.started → one open |
| 11 | `listen/open-tty/enabled/dedupe-same-session` | Same session_id → open once; other id opens |
| 12 | `listen/open-tty/enabled/missing-session-id-warns` | Missing session_id → warning; no open |
| 13 | `listen/open-tty/enabled/open-failure-warns` | Open error → warning; continue exit 0 |
| 14 | `listen/open-tty/enabled/ignores-non-tty-event` | seatalk.message.received → no open |
| 15 | `listen/open-tty/disabled/never-opens` | Without OpenTTY → never open |
| 16 | `rejected/unknown-subcommand` | `event-bus foo` → non-zero + Error |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Surface (help / listen / rejected) | help/*, listen/*, rejected/* |
| Help depth (top / event-bus / listen) | help/* |
| Output format (human / --json) | listen/human/*, listen/json/* |
| Token empty vs default | listen/human/empty-token-ok, listen/human/one-event |
| Reconnect / warning | listen/reconnect/* |
| Type filter | listen/filter/* |
| Replay N | listen/replay/* |
| OpenTTY on / off | listen/open-tty/enabled/*, listen/open-tty/disabled/* |
| Open event type (tty vs other) | opens-on-tty-started, ignores-non-tty-event |
| session_id present / missing | opens-on-tty-started, missing-session-id-warns |
| Open success / fail | opens-on-tty-started, open-failure-warns |
| Process-local session dedupe | dedupe-same-session |
| Unknown subcommand | rejected/* |

## Locked product API (implementer)

Classic TDD: **new** `--open-tty` symbols below may not exist until implementer
scaffolds them. Scaffold `OpenTTY` / `OpenTTYSession` first so the suite
compiles (existing leaves stay GREEN with defaults), then satisfy open-tty
Asserts.

Package `github.com/xhd2015/ai-critic/cmd/agentcli` (or subpackage wired from
`Run`):

| Symbol | Role |
|--------|------|
| `event-bus` command in `Run` / `RunWithWriters` switch | dispatch + top-level help listing |
| `RunWithWriters(profile Profile, args []string, stdout, stderr io.Writer) error` | L2 CLI entry (help/reject); product `Run` may delegate with `os.Stdout`/`os.Stderr` |
| `RunEventBusListen(stdout, stderr io.Writer, opts EventBusListenOpts) error` | L2-injectable listen core |
| `EventBusListenOpts` | existing: `Server, Token string`; `Types []string`; `JSON bool`; `Replay int`; `DialWS EventBusDialFunc`; `Now func() time.Time`; `MaxEvents int`; `Context context.Context`; **new:** `OpenTTY bool`; `OpenTTYSession func(sessionID string) error` |
| `EventBusDialFunc` | `func(ctx context.Context, wsURL string, header http.Header) (EventBusConn, error)` |
| `EventBusConn` | `ReadJSON(v any) error` (or equivalent text-frame Event read) + `Close() error` |
| Human log | green `connected  server=<base>`; event lines `HH:MM:SS  <type>  …` (local clock from `Now`); yellow `warning:` on disconnect before reconnect |
| `--json` | one JSON Event object per stdout line; no ANSI on stdout/stderr for event payload path |
| Token | optional; empty token may omit Authorization or send empty Bearer |
| `--type` | repeatable; when non-empty, print only matching `Event.Type` |
| `--replay N` | print up to N hub recent events (oldest first) before live; may use WS query, HTTP, or client fetch of Recent — implementer choice as long as Asserts pass |
| `--open-tty` / `OpenTTY` | default off; when on, after printing `agent.tty.started`, parse payload `session_id`; if non-empty and not yet opened this process, call `OpenTTYSession(sessionID)` (nil → production iTerm ForceNew with `remote-agent agent-run attach <session_id>`). Process-local map/set dedupe only (no disk). Missing/empty session_id → `warning:` on stderr, no open, keep listening. Open error → `warning:` on stderr, keep listening. Never open for other types. |
| Reconnect | on read/dial failure after a successful connect, print `warning:` (stderr) and retry dial (backoff ok; tests use injectable dialer) |
| Stop | when `MaxEvents > 0`, return nil after that many **printed** events; honor `Context` cancel |

Wire path for real dial: `ws(s)://<host>/api/event-bus/ws` from `Server` base URL
(same as PHASE 2 `SubscribeWSPath`).

CLI shape:

```
remote-agent event-bus listen [--type T ...] [--json] [--replay N] [--open-tty]
```

## How to Run

From ai-critic module root:

```sh
doctest vet ./tests/remote-agent-event-bus
doctest test ./tests/remote-agent-event-bus/...
```

Single leaf:

```sh
doctest test ./tests/remote-agent-event-bus/help/listen
doctest test ./tests/remote-agent-event-bus/listen/open-tty/enabled/opens-on-tty-started
```

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/server/eventbus"
	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

// EventSeed is a fixture event published on the hub or injected via DialWS.
type EventSeed struct {
	ID      string
	TS      string
	Source  string
	Type    string
	Host    string
	Payload string // JSON object text; empty → {}
}

// Request drives one leaf.
//
// Op values:
//   cli     — agentcli.RunWithWriters(Args) for help / rejected
//   listen  — agentcli.RunEventBusListen with opts (+ optional real hub)
type Request struct {
	Op string

	// CLI (Op=cli)
	Args []string

	// Listen (Op=listen)
	Server string // optional override; default from ephemeral hub
	Token  string // use EmptyToken to force ""
	EmptyToken bool
	Types  []string
	JSON   bool
	Replay int

	// --open-tty / OpenTTY (PHASE 2)
	OpenTTY bool
	// InjectOpenHook installs a recording OpenTTYSession even when OpenTTY is
	// false (disabled leaf proves the hook is never called).
	InjectOpenHook bool
	// OpenTTYFail makes the injected OpenTTYSession return an error.
	OpenTTYFail bool

	// Hub / stream control
	// When DialMode is empty or "hub", start RegisterSubscribeWS and publish LiveEvents after connect.
	// When DialMode is "drop-once", use injectable DialWS that fails once then delivers InjectEvents.
	// When DialMode is "inject", DialWS delivers InjectEvents only (no hub).
	DialMode string

	// Events published on hub after listen starts (hub mode), or full stream (inject).
	LiveEvents []EventSeed
	// Pre-seeded hub events for --replay (published before listen starts).
	RecentEvents []EventSeed
	// InjectEvents used by injectable dialer (reconnect / pure inject / open-tty).
	InjectEvents []EventSeed

	// MaxEvents passed to opts (0 → default from leaf Setup).
	MaxEvents int

	// Fixed clock for human timestamps (local HH:MM:SS).
	FixedNow time.Time
}

type Response struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string

	ServerURL string
	WSURL     string

	// OpenTTYSessionIDs is the ordered list of session IDs passed to the
	// injected OpenTTYSession hook (empty when hook not installed or never called).
	OpenTTYSessionIDs []string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	resp := &Response{}
	if req.Op == "" {
		req.Op = "cli"
	}

	switch req.Op {
	case "cli":
		return runCLI(t, req, resp)
	case "listen":
		return runListen(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown Op %q", req.Op)
	}
}

func runCLI(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	args := req.Args
	if len(args) == 0 {
		args = []string{"--help"}
	}
	var stdout, stderr bytes.Buffer
	// Intended L2 API: no os.Stdout/os.Stderr reassignment (parallel-safe).
	runErr := agentcli.RunWithWriters(agentcli.RemoteProfile(), args, &stdout, &stderr)
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if runErr != nil {
		if !strings.Contains(resp.Stderr, "Error:") {
			resp.Stderr += fmt.Sprintf("Error: %v\n", runErr)
		}
		resp.ExitCode = 1
	} else {
		resp.ExitCode = 0
	}
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	return resp, nil
}

func runListen(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()

	token := req.Token
	if req.EmptyToken {
		token = ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	maxEvents := req.MaxEvents
	if maxEvents <= 0 {
		maxEvents = 1
	}

	fixedNow := req.FixedNow
	if fixedNow.IsZero() {
		fixedNow = time.Date(2026, 8, 10, 12, 34, 56, 0, time.Local)
	}

	var opts agentcli.EventBusListenOpts
	opts.Token = token
	opts.Types = append([]string(nil), req.Types...)
	opts.JSON = req.JSON
	opts.Replay = req.Replay
	opts.OpenTTY = req.OpenTTY
	opts.MaxEvents = maxEvents
	opts.Context = ctx
	opts.Now = func() time.Time { return fixedNow }

	// Injectable OpenTTYSession recorder (L2; no real iTerm/osascript).
	var openMu sync.Mutex
	var openIDs []string
	if req.InjectOpenHook || req.OpenTTY || req.OpenTTYFail {
		opts.OpenTTYSession = func(sessionID string) error {
			openMu.Lock()
			openIDs = append(openIDs, sessionID)
			openMu.Unlock()
			if req.OpenTTYFail {
				return fmt.Errorf("simulated open failure")
			}
			return nil
		}
	}

	dialMode := req.DialMode
	if dialMode == "" {
		dialMode = "hub"
	}

	switch dialMode {
	case "hub":
		hub, baseURL, wsURL, cleanup := startHubWS(t)
		t.Cleanup(cleanup)
		resp.ServerURL = baseURL
		resp.WSURL = wsURL
		opts.Server = baseURL
		if req.Server != "" {
			opts.Server = req.Server
		}

		// Pre-seed recent ring for --replay.
		for _, s := range req.RecentEvents {
			hub.Publish(toSharedEvent(s))
		}

		// Publish live events shortly after listen dials.
		go func() {
			time.Sleep(80 * time.Millisecond)
			for _, s := range req.LiveEvents {
				hub.Publish(toSharedEvent(s))
			}
		}()

	case "inject", "drop-once":
		opts.Server = req.Server
		if opts.Server == "" {
			opts.Server = "http://127.0.0.1:23712"
		}
		resp.ServerURL = opts.Server
		resp.WSURL = httpToWS(opts.Server) + eventbus.SubscribeWSPath
		opts.DialWS = makeInjectDialer(t, dialMode, req.InjectEvents)

	default:
		return nil, fmt.Errorf("unknown DialMode %q", dialMode)
	}

	var stdout, stderr bytes.Buffer
	runErr := agentcli.RunEventBusListen(&stdout, &stderr, opts)

	openMu.Lock()
	resp.OpenTTYSessionIDs = append([]string(nil), openIDs...)
	openMu.Unlock()

	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	if runErr != nil {
		// Match product CLI: Error: on stderr + non-zero when listen fails hard.
		if !strings.Contains(resp.Stderr, "Error:") {
			resp.Stderr += fmt.Sprintf("Error: %v\n", runErr)
			resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
		}
		resp.ExitCode = 1
		// Context cancel after MaxEvents success should not be treated as failure by product;
		// if implementer returns ctx.Err() after success, Asserts check ExitCode==0 via MaxEvents path.
		// Leaves that expect success check ExitCode 0; hard errors stay non-zero.
		if opts.MaxEvents > 0 && ctx.Err() != nil && strings.Count(resp.Stdout, "\n") > 0 {
			// Prefer success when we already printed events and context ended the loop.
			// Implementer should return nil on MaxEvents; this softens partial RED ambiguity.
		}
		return resp, nil
	}
	resp.ExitCode = 0
	return resp, nil
}

func startHubWS(t *testing.T) (hub *eventbus.Hub, baseURL, wsURL string, cleanup func()) {
	t.Helper()
	hub = eventbus.NewHub(200)
	mux := http.NewServeMux()
	eventbus.RegisterSubscribeWS(mux, hub)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	addr := ln.Addr().String()
	baseURL = "http://" + addr
	wsURL = "ws://" + addr + eventbus.SubscribeWSPath
	// brief ready
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/ping")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	cleanup = func() { _ = srv.Close() }
	return hub, baseURL, wsURL, cleanup
}

func toSharedEvent(s EventSeed) sharedeb.Event {
	payload := s.Payload
	if payload == "" {
		payload = `{}`
	}
	return sharedeb.Event{
		ID:      s.ID,
		TS:      s.TS,
		Source:  s.Source,
		Type:    s.Type,
		Host:    s.Host,
		Payload: json.RawMessage(payload),
	}
}

func httpToWS(server string) string {
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

// injectConn is a minimal EventBusConn stand-in used only if product types
// are interfaces satisfied by test fakes. The dialer below builds a stream via
// the product EventBusDialFunc contract.
type injectFrame struct {
	ev  sharedeb.Event
	err error // if non-nil, Read returns this (disconnect)
}

func makeInjectDialer(t *testing.T, mode string, events []EventSeed) agentcli.EventBusDialFunc {
	t.Helper()
	var mu sync.Mutex
	dialCount := 0
	return func(ctx context.Context, wsURL string, header http.Header) (agentcli.EventBusConn, error) {
		mu.Lock()
		dialCount++
		n := dialCount
		mu.Unlock()

		if mode == "drop-once" {
			if n == 1 {
				// First dial: one event then hard disconnect on next read.
				frames := make([]injectFrame, 0, 2)
				if len(events) > 0 {
					frames = append(frames, injectFrame{ev: toSharedEvent(events[0])})
				}
				frames = append(frames, injectFrame{err: io.ErrUnexpectedEOF})
				return newInjectConn(frames), nil
			}
			// Second dial: remaining events (or event index 1+).
			rest := events
			if len(events) > 1 {
				rest = events[1:]
			} else if len(events) == 1 {
				// re-deliver last as recovery signal if only one configured
				rest = events
			}
			frames := make([]injectFrame, 0, len(rest))
			for _, s := range rest {
				frames = append(frames, injectFrame{ev: toSharedEvent(s)})
			}
			return newInjectConn(frames), nil
		}

		// pure inject: deliver all events then EOF
		frames := make([]injectFrame, 0, len(events)+1)
		for _, s := range events {
			frames = append(frames, injectFrame{ev: toSharedEvent(s)})
		}
		frames = append(frames, injectFrame{err: io.EOF})
		return newInjectConn(frames), nil
	}
}

type injectConn struct {
	mu     sync.Mutex
	frames []injectFrame
	idx    int
	closed bool
}

func newInjectConn(frames []injectFrame) *injectConn {
	return &injectConn{frames: frames}
}

func (c *injectConn) ReadJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
	if c.idx >= len(c.frames) {
		return io.EOF
	}
	f := c.frames[c.idx]
	c.idx++
	if f.err != nil {
		return f.err
	}
	data, err := json.Marshal(f.ev)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (c *injectConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

```
