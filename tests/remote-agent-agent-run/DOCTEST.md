# Remote-Agent Agent-Run (Full Remote Façade)

Classic TDD doctests for `remote-agent agent-run`:

- **P1** — `sessions` list mode
- **P2** — `attach` + keepalive
- **P3** — `status` + `resume`
- **P4** — `send`, `msg`, `snapshot`, `watch`, `kill`
- **P5** — `run` (library) + local-only command errors + full help tree polish

Remote façade over agent-run session storage via `agentstorage` and HTTP under
`/api/agent-run/...`. **No exec of local `agent-run` binary** — agent-pro
libraries + inject hooks.

**Out of scope:** iTerm open, menubar, perfect flag parity, real grok.

All leaves are **L2 in-process**:

- Prior phases as before
- **P5:** `Options.RunSession` inject; CLI rejects `--new-terminal` and
  `focus|web|assets|pty` as local-only
- Store home constructor-injected; no `t.Setenv` / `t.Chdir`
- CLI capture via `RunWithWriters` (harness does not reassign `os.Stdout`)

# DSN (Domain Specific Notion)

Remote operator UX matching local `agent-run` command tree — remote-capable
commands hit the server library path; local-only commands fail clearly.

**Participants**

- **CLI (`agentcli.RunWithWriters`)** — full `agent-run` subcommand tree against
  ephemeral mux + bearer auth.
- **server/agentrun** — inject hooks through `RegisterAPIWithOptions` including
  `RunSession` for P5.
- **agentrunapi / agenttty / agentsend / ttywatch** — production when injects nil.
- **Isolated store home** — per-`Run` temp dir; parallel-safe.

**Behaviors (P1–P4)** — sealed prior leaves.

**Behaviors (P5 run + polish)**

- `run --help` documents core flags (`--session-id`, `--dir`, `--open`,
  `--detach`, `--json`, `--auto-send-or-resume`, …).
- `run` success via inject (prefer `--detach` path: prints session/terminal ids).
- Validation when required inputs missing (prompt/session as product designs).
- `--new-terminal` → clear error (not available via remote-agent).
- `focus|web|assets|pty` → clear **local-only** errors.
- Top-level `agent-run --help` lists the remote command tree (sessions, attach,
  status, resume, send, msg, snapshot, watch, kill, run) and/or notes local-only.

## Version

0.0.2

## Decision Tree

```
[remote-agent agent-run — P1 … P5]
 |
 +-- help|sessions|api|unknown/              (P1 sealed)
 |    +-- help/full-command-tree/            (LEAF) P5 full help listing
 +-- attach/                                 (P2 sealed)
 +-- status/|resume/                         (P3 sealed)
 +-- send|msg|snapshot|watch|kill/           (P4 sealed)
 |
 +-- run/                                    (GROUP) P5 run
 |    +-- help/                              (LEAF)
 |    +-- success-detach/                    (LEAF)  inject detach success
 |    +-- validation/requires-input/         (LEAF)  missing prompt/session
 |    +-- new-terminal-rejected/             (LEAF)  --new-terminal error
 |
 +-- local-only/                             (GROUP) P5 local-only rejects
      +-- focus/                             (LEAF)
      +-- web/                               (LEAF)
      +-- assets/                            (LEAF)
      +-- pty/                               (LEAF)
```

## Test Index

| # | Leaf | Phase | Description |
|---|------|-------|-------------|
| 1–49 | P1–P4 (sealed) | P1–P4 | prior tree |
| 50 | `run/help` | P5 | run --help core flags |
| 51 | `run/success-detach` | P5 | inject run --detach success ids |
| 52 | `run/validation/requires-input` | P5 | missing required prompt/session |
| 53 | `run/new-terminal-rejected` | P5 | --new-terminal clear error |
| 54 | `local-only/focus` | P5 | focus → local-only error |
| 55 | `local-only/web` | P5 | web → local-only error |
| 56 | `local-only/assets` | P5 | assets → local-only error |
| 57 | `local-only/pty` | P5 | pty → local-only error |
| 58 | `help/full-command-tree` | P5 | agent-run --help lists full remote tree |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| P1–P4 sealed | prior paths |
| run help / success / validation / new-terminal | run/* |
| local-only subcommands | local-only/* |
| full top-level help | help/full-command-tree |

## How to Run

```sh
doctest vet ./tests/remote-agent-agent-run
doctest test ./tests/remote-agent-agent-run/...
```

Classic TDD: **P5 leaves RED** until run + local-only errors exist.
**P1–P4 ASSERTs sealed**. No L3 / `label: e2e`.

### Implementer Run() / API contract (P1–P5)

```text
server/agentrun Options (extend; prior fields keep working):

  // P2–P4 as before: ResolveTTY, FakeAttach, ProbeSession, ResumeSession,
  // SendMessage, MsgStatus, MsgCancel, Snapshot, Watch, Kill, …

  // P5
  RunSession(opts RunSessionOpts) (RunSessionResult, error)
  // nil → agentrunapi.AutoSendOrResume / production run path (no agent-run exec)

  type RunSessionOpts struct {
    SessionID         string
    Prompt            string
    Dir               string
    Open              bool
    Detach            bool
    JSON              bool
    AutoSendOrResume  bool
    AgentRunner       string
    // env / prepend / model as product supports
  }
  type RunSessionResult struct {
    SessionID  string
    TerminalID string
  }

  Route: POST /api/agent-run/run  body mirrors RunSessionOpts
         → {session_id, terminal_id, …}

CLI:
  agent-run run [OPTIONS] ["prompt"]
    --session-id, --dir, --open, --detach, --json, --auto-send-or-resume,
    --agent-runner, …
    --new-terminal → Error: not available via remote-agent (or clear reject)

  agent-run focus|web|assets|pty …
    → Error: not available via remote-agent (local-only: …)

  agent-run --help lists: sessions, attach, status, resume, send, msg,
    snapshot, watch, kill, run; may list local-only with note
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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/agentrun"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli profile swap (if global).
var agentcliInProcessMu sync.Mutex

// SessionSeed is one sessions/<id>/meta.json written before Run.
type SessionSeed struct {
	SessionID         string
	Runner            string
	Status            string
	CreatedAt         string // RFC3339; optional
	UpdatedAt         string // RFC3339; optional — drives newest-first order
	TerminalSessionID string // optional — P2 attach resolve
	RunnerSessionID   string // optional — P3 resume bind (provider session)
	Workspace         string // optional — status workspace layer
}

// StatusProbeSeed is an injectable multi-layer probe result (P3).
type StatusProbeSeed struct {
	Session        string
	Status         string
	Workspace      string
	ProcessStatus  string // alive|dead|unknown
	TerminalStatus string // reachable|unreachable|missing
	RunnerStatus   string // bound|unbound|binding
	RunnerExited   *bool  // nil unknown; true exited; false live
	ResumeReady    bool
	ResumeReason   string
}

// Request configures one leaf.
//
// Op values:
//   cli       — agentcli.RunWithWriters(Args) (P1–P4 CLI paths)
//   api       — GET /api/agent-run/sessions (optional ?limit=)  [P1]
//   attach    — dial attach WS (missing / unreachable / live / disconnect)
//   keepalive — dial attach WS idle; count Ping frames under injected interval
type Request struct {
	Op string

	// CLI argv after global --server/--token (e.g. agent-run sessions --json).
	Args []string

	// APILimit: nil omits query param (server default limit); non-nil sends ?limit=N.
	// 0 means all sessions (match local agent-run sessions --limit 0).
	APILimit *int

	// Seeds written under the temp agent-run home before list/attach/status.
	Seeds []SessionSeed

	// --- P2 attach ---

	// AttachSessionID is the agent-run session id for Op=attach|keepalive.
	AttachSessionID string

	// TTYMode selects injectable server TTY behavior for attach ops:
	//   "" / "missing" — no resolve success (use empty Seeds or wrong id)
	//   "unreachable"  — ResolveTTY returns reachable=false
	//   "live"         — FakeAttach echo (or hold) backend
	//   "unclean"      — FakeAttach holds then closes abnormally
	//   "hold"         — FakeAttach hold (keepalive / open-then-attach)
	TTYMode string

	// AttachInput is binary payload written on the attach WS (live-roundtrip).
	AttachInput []byte

	// AttachHold is how long the client keeps the attach open before closing
	// (live / unclean). Zero → short default in Run.
	AttachHold time.Duration

	// PingInterval injects server/client hop ping period (keepalive). Zero on
	// product default; tests set e.g. 50ms.
	PingInterval time.Duration

	// WaitForPing is how long keepalive waits to observe at least one ping.
	WaitForPing time.Duration

	// --- P3 status / resume inject ---

	// ProbeInject, when non-nil, wires Options.ProbeSession to return this
	// report for the seeded session id (and error for other ids).
	ProbeInject *StatusProbeSeed

	// ResumeInject selects Options.ResumeSession behavior:
	//   ""              — no inject (product default / library)
	//   "error-live"    — session still live; suggest send/attach
	//   "error-unbound" — missing runner bind
	//   "ok"            — success (exited+bound path)
	ResumeInject string

	// WantAttachOnResume wires FakeAttach and records AttachInvoked when
	// resume --open causes an attach (L2 inject observation).
	WantAttachOnResume bool

	// --- P4 live-control inject ---

	// SendInject: "" | "ok" | "error-unreachable"
	SendInject string
	// SendResultMsgID returned on ok (default msg_1).
	SendResultMsgID string

	// MsgStatusInject: "" | "pending" | "delivered" | "error"
	MsgStatusInject string
	// MsgCancelInject: "" | "ok" | "error"
	MsgCancelInject string

	// SnapshotInject: "" | "ok" | "error"
	SnapshotInject string
	SnapshotText   string // body when ok

	// WatchInject: "" | "ok" | "error"
	WatchInject string
	WatchLines  []string // short stream lines when ok

	// KillInject: "" | "ok" | "error"
	KillInject string

	// --- P5 run inject ---

	// RunInject: "" | "ok" | "error"
	RunInject string
	// Result ids returned on ok (defaults sess-run-1 / term-run-1).
	RunResultSessionID  string
	RunResultTerminalID string
}

// SessionItem is the list DTO (API JSON and --json CLI).
type SessionItem struct {
	SessionID string `json:"session_id"`
	Runner    string `json:"runner"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Response struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string

	// API list
	HTTPStatus int
	Sessions   []SessionItem
	Body       string

	// Attach / keepalive
	AttachErr      string
	ReceivedOutput string
	PingCount      int
	TermRestored   bool
	WSHTTPStatus   int // non-101 upgrade failures

	// P3 status / resume inject observation
	ResumeCalled  bool
	ResumeOpen    bool // ResumeOpts.Open observed
	AttachInvoked bool // FakeAttach entered (e.g. resume --open)

	// P4 live-control observation
	SendCalled      bool
	SendMsgID       string
	MsgStatusCalled bool
	MsgStatus       string
	MsgCancelCalled bool
	SnapshotCalled  bool
	SnapshotOut     string
	WatchCalled     bool
	WatchOut        string
	KillCalled      bool
	KillDryRun      bool
	KillReport      string

	// P5 run observation
	RunCalled     bool
	RunSessionID  string
	RunTerminalID string
	RunDetach     bool
	RunOpen       bool

	// Paths
	ServerURL string
	StoreHome string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	if req.Op == "" {
		req.Op = "cli"
	}

	storeHome, err := os.MkdirTemp("", "agent-run-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = os.RemoveAll(storeHome) })
	resp.StoreHome = storeHome

	if err := seedSessionMetas(storeHome, req.Seeds); err != nil {
		return nil, fmt.Errorf("seed sessions: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if err := registerAgentRun(mux, storeHome, req, resp); err != nil {
		return nil, err
	}

	handler := withBearerAuth(lib.TestPassword, mux)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	port := ln.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	resp.ServerURL = serverURL

	if err := waitHTTPReady(serverURL+"/ping", 10*time.Second); err != nil {
		return nil, err
	}

	switch req.Op {
	case "cli":
		return runCLI(t, req, resp, serverURL)
	case "api":
		return runAPI(t, req, resp, serverURL)
	case "attach":
		return runAttachWS(t, req, resp, serverURL)
	case "keepalive":
		return runKeepaliveWS(t, req, resp, serverURL)
	default:
		return nil, fmt.Errorf("unknown Op %q", req.Op)
	}
}

// registerAgentRun mounts list (+ attach/status/resume inject when needed).
// Plain RegisterAPI when no inject hooks; otherwise RegisterAPIWithOptions.
func registerAgentRun(mux *http.ServeMux, home string, req *Request, resp *Response) error {
	if !needsOptionsHooks(req) {
		return agentrun.RegisterAPI(mux, home)
	}

	opts := agentrun.Options{
		Home:         home,
		PingInterval: req.PingInterval,
		OnLocalRestore: func() {
			resp.TermRestored = true
		},
	}
	switch strings.ToLower(strings.TrimSpace(req.TTYMode)) {
	case "unreachable":
		opts.ResolveTTY = func(sessionID string) (string, string, bool, error) {
			return "term-" + sessionID, "127.0.0.1:1", false, nil
		}
	case "live":
		opts.FakeAttach = wrapFakeAttach(resp, fakeAttachEcho)
	case "unclean":
		hold := req.AttachHold
		if hold <= 0 {
			hold = 80 * time.Millisecond
		}
		opts.FakeAttach = wrapFakeAttach(resp, fakeAttachUncleanClose(hold))
	case "hold", "keepalive":
		opts.FakeAttach = wrapFakeAttach(resp, fakeAttachHold)
	default:
		if req.WantAttachOnResume {
			// resume --open: observe FakeAttach without requiring live echo.
			opts.FakeAttach = wrapFakeAttach(resp, fakeAttachHold)
		}
	}

	// P3: ProbeSession inject
	if req.ProbeInject != nil {
		seed := *req.ProbeInject
		opts.ProbeSession = func(sessionID string) (agentrun.StatusReport, error) {
			// Match any non-empty id to the inject (leaves use one session).
			if strings.TrimSpace(sessionID) == "" {
				return agentrun.StatusReport{}, fmt.Errorf("session not found: empty")
			}
			return agentrun.StatusReport{
				Session:   firstNonEmpty(seed.Session, sessionID),
				Status:    seed.Status,
				Workspace: seed.Workspace,
				Process:   agentrun.ProcessLayer{Status: seed.ProcessStatus},
				Terminal:  agentrun.TerminalLayer{Status: seed.TerminalStatus},
				Runner: agentrun.RunnerLayer{
					Status:    seed.RunnerStatus,
					SessionID: sessionID,
					Exited:    seed.RunnerExited,
				},
				Resume: agentrun.ResumeLayer{
					Ready:  seed.ResumeReady,
					Reason: seed.ResumeReason,
				},
			}, nil
		}
	}

	// P3: ResumeSession inject
	if inj := strings.ToLower(strings.TrimSpace(req.ResumeInject)); inj != "" {
		opts.ResumeSession = func(sessionID string, ro agentrun.ResumeOpts) error {
			resp.ResumeCalled = true
			resp.ResumeOpen = ro.Open
			switch inj {
			case "error-live":
				return fmt.Errorf("session %s is live; use send or attach instead of resume", sessionID)
			case "error-unbound":
				return fmt.Errorf("session %s has no runner_session_id bind; cannot resume", sessionID)
			case "ok":
				return nil
			default:
				return fmt.Errorf("unknown ResumeInject %q", inj)
			}
		}
	}

	// P4: live-control injects
	wireP4Injects(req, resp, &opts)
	// P5: run inject
	wireP5RunInject(req, resp, &opts)

	return agentrun.RegisterAPIWithOptions(mux, opts)
}

func wireP5RunInject(req *Request, resp *Response, opts *agentrun.Options) {
	inj := strings.ToLower(strings.TrimSpace(req.RunInject))
	if inj == "" {
		return
	}
	sessID := strings.TrimSpace(req.RunResultSessionID)
	if sessID == "" {
		sessID = "sess-run-1"
	}
	termID := strings.TrimSpace(req.RunResultTerminalID)
	if termID == "" {
		termID = "term-run-1"
	}
	opts.RunSession = func(ro agentrun.RunSessionOpts) (agentrun.RunSessionResult, error) {
		resp.RunCalled = true
		resp.RunDetach = ro.Detach
		resp.RunOpen = ro.Open
		switch inj {
		case "ok":
			// Prefer detach success path for L2 (no interactive attach).
			outSess := sessID
			if strings.TrimSpace(ro.SessionID) != "" {
				outSess = ro.SessionID
			}
			resp.RunSessionID = outSess
			resp.RunTerminalID = termID
			return agentrun.RunSessionResult{
				SessionID:  outSess,
				TerminalID: termID,
			}, nil
		case "error":
			return agentrun.RunSessionResult{}, fmt.Errorf("run failed: inject error")
		default:
			return agentrun.RunSessionResult{}, fmt.Errorf("unknown RunInject %q", inj)
		}
	}
}

func wireP4Injects(req *Request, resp *Response, opts *agentrun.Options) {
	if inj := strings.ToLower(strings.TrimSpace(req.SendInject)); inj != "" {
		msgID := strings.TrimSpace(req.SendResultMsgID)
		if msgID == "" {
			msgID = "msg_1"
		}
		opts.SendMessage = func(sessionID, message string, so agentrun.SendOpts) (string, error) {
			resp.SendCalled = true
			_ = so
			if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(message) == "" {
				return "", fmt.Errorf("send: requires session-id and message")
			}
			switch inj {
			case "ok":
				resp.SendMsgID = msgID
				return msgID, nil
			case "error-unreachable":
				return "", fmt.Errorf("terminal unreachable for session %s", sessionID)
			default:
				return "", fmt.Errorf("unknown SendInject %q", inj)
			}
		}
	}
	if inj := strings.ToLower(strings.TrimSpace(req.MsgStatusInject)); inj != "" {
		opts.MsgStatus = func(sessionID, msgID string) (string, error) {
			resp.MsgStatusCalled = true
			if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(msgID) == "" {
				return "", fmt.Errorf("msg status: requires session-id/message-id")
			}
			switch inj {
			case "pending", "delivered":
				resp.MsgStatus = inj
				return inj, nil
			case "error":
				return "", fmt.Errorf("message not found: %s/%s", sessionID, msgID)
			default:
				return "", fmt.Errorf("unknown MsgStatusInject %q", inj)
			}
		}
	}
	if inj := strings.ToLower(strings.TrimSpace(req.MsgCancelInject)); inj != "" {
		opts.MsgCancel = func(sessionID, msgID string) error {
			resp.MsgCancelCalled = true
			if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(msgID) == "" {
				return fmt.Errorf("msg cancel: requires session-id/message-id")
			}
			switch inj {
			case "ok":
				return nil
			case "error":
				return fmt.Errorf("cannot cancel message %s/%s", sessionID, msgID)
			default:
				return fmt.Errorf("unknown MsgCancelInject %q", inj)
			}
		}
	}
	if inj := strings.ToLower(strings.TrimSpace(req.SnapshotInject)); inj != "" {
		text := req.SnapshotText
		if text == "" {
			text = "snapshot: hello from inject\n"
		}
		opts.Snapshot = func(sessionID string) (string, error) {
			resp.SnapshotCalled = true
			if strings.TrimSpace(sessionID) == "" {
				return "", fmt.Errorf("snapshot: requires session-id")
			}
			switch inj {
			case "ok":
				resp.SnapshotOut = text
				return text, nil
			case "error":
				return "", fmt.Errorf("terminal unreachable for snapshot %s", sessionID)
			default:
				return "", fmt.Errorf("unknown SnapshotInject %q", inj)
			}
		}
	}
	if inj := strings.ToLower(strings.TrimSpace(req.WatchInject)); inj != "" {
		lines := req.WatchLines
		if len(lines) == 0 {
			lines = []string{"watch-line-1", "watch-line-2"}
		}
		opts.Watch = func(sessionID string, w io.Writer, stop <-chan struct{}) error {
			resp.WatchCalled = true
			_ = stop
			if strings.TrimSpace(sessionID) == "" {
				return fmt.Errorf("watch: requires session-id")
			}
			switch inj {
			case "ok":
				var b strings.Builder
				for _, ln := range lines {
					fmt.Fprintln(w, ln)
					b.WriteString(ln)
					b.WriteByte('\n')
				}
				resp.WatchOut = b.String()
				return nil
			case "error":
				return fmt.Errorf("session not found: %s", sessionID)
			default:
				return fmt.Errorf("unknown WatchInject %q", inj)
			}
		}
	}
	if inj := strings.ToLower(strings.TrimSpace(req.KillInject)); inj != "" {
		opts.Kill = func(sessionID string, dryRun bool) (string, error) {
			resp.KillCalled = true
			resp.KillDryRun = dryRun
			if strings.TrimSpace(sessionID) == "" {
				return "", fmt.Errorf("kill: requires session-id")
			}
			switch inj {
			case "ok":
				report := "would stop " + sessionID
				if !dryRun {
					report = "stopped " + sessionID
				}
				resp.KillReport = report
				return report, nil
			case "error":
				return "", fmt.Errorf("session not found: %s", sessionID)
			default:
				return "", fmt.Errorf("unknown KillInject %q", inj)
			}
		}
	}
}

func wrapFakeAttach(resp *Response, fn func(*websocket.Conn, string) error) func(*websocket.Conn, string) error {
	return func(conn *websocket.Conn, sessionID string) error {
		resp.AttachInvoked = true
		if fn == nil {
			return nil
		}
		return fn(conn, sessionID)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func needsOptionsHooks(req *Request) bool {
	if req == nil {
		return false
	}
	switch req.Op {
	case "attach", "keepalive":
		return true
	}
	if req.ProbeInject != nil || strings.TrimSpace(req.ResumeInject) != "" || req.WantAttachOnResume {
		return true
	}
	// P4 injects
	if strings.TrimSpace(req.SendInject) != "" ||
		strings.TrimSpace(req.MsgStatusInject) != "" ||
		strings.TrimSpace(req.MsgCancelInject) != "" ||
		strings.TrimSpace(req.SnapshotInject) != "" ||
		strings.TrimSpace(req.WatchInject) != "" ||
		strings.TrimSpace(req.KillInject) != "" {
		return true
	}
	// P5 run inject
	if strings.TrimSpace(req.RunInject) != "" {
		return true
	}
	return false
}

func fakeAttachEcho(conn *websocket.Conn, sessionID string) error {
	_ = sessionID
	defer conn.Close()
	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
			if err := conn.WriteMessage(mt, data); err != nil {
				return nil
			}
		}
	}
}

func fakeAttachHold(conn *websocket.Conn, sessionID string) error {
	_ = sessionID
	defer conn.Close()
	// Block until client closes or context via read error.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return nil
		}
	}
}

func fakeAttachUncleanClose(hold time.Duration) func(*websocket.Conn, string) error {
	return func(conn *websocket.Conn, sessionID string) error {
		_ = sessionID
		time.Sleep(hold)
		// Abnormal close (not normal/going-away).
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "tty lost"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil
	}
}

func runCLI(t *testing.T, req *Request, resp *Response, serverURL string) (*Response, error) {
	t.Helper()
	args := req.Args
	if len(args) == 0 {
		args = []string{"agent-run", "sessions"}
	}
	argv := append([]string{"--server", serverURL, "--token", lib.TestPassword}, args...)

	exitCode, stdout, stderr, runErr := runAgentInProcess(argv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)

	// Best-effort parse --json bodies for leaves that care.
	if containsJSONFlag(args) && exitCode == 0 {
		var payload struct {
			Sessions []SessionItem `json:"sessions"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err == nil {
			resp.Sessions = payload.Sessions
			if resp.Sessions == nil {
				resp.Sessions = []SessionItem{}
			}
		}
	}
	return resp, nil
}

func runAPI(t *testing.T, req *Request, resp *Response, serverURL string) (*Response, error) {
	t.Helper()
	path := "/api/agent-run/sessions"
	if req.APILimit != nil {
		path += "?limit=" + strconv.Itoa(*req.APILimit)
	}
	httpReq, err := http.NewRequest(http.MethodGet, serverURL+path, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+lib.TestPassword)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp.HTTPStatus = httpResp.StatusCode
	resp.Body = string(body)
	resp.ExitCode = 0
	if httpResp.StatusCode != http.StatusOK {
		resp.ExitCode = 1
		resp.Stderr = string(body)
		resp.Combined = string(body)
		return resp, nil
	}

	var payload struct {
		Sessions []SessionItem `json:"sessions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode sessions JSON: %w; body=%s", err, body)
	}
	resp.Sessions = payload.Sessions
	if resp.Sessions == nil {
		resp.Sessions = []SessionItem{}
	}
	resp.Stdout = string(body)
	resp.Combined = string(body)
	return resp, nil
}

func runAttachWS(t *testing.T, req *Request, resp *Response, serverURL string) (*Response, error) {
	t.Helper()
	sid := strings.TrimSpace(req.AttachSessionID)
	if sid == "" {
		sid = "sess-missing"
	}
	wsURL := toWSURL(serverURL) + "/api/agent-run/sessions/" + url.PathEscape(sid) + "/attach"

	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+lib.TestPassword)
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	conn, httpResp, err := dialer.Dial(wsURL, hdr)
	if httpResp != nil {
		resp.WSHTTPStatus = httpResp.StatusCode
	}
	if err != nil {
		resp.ExitCode = 1
		resp.AttachErr = err.Error()
		if httpResp != nil && httpResp.Body != nil {
			b, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			if len(b) > 0 {
				resp.Body = string(b)
				resp.AttachErr = strings.TrimSpace(resp.AttachErr + ": " + string(b))
			}
		}
		resp.Combined = resp.AttachErr
		resp.Stderr = resp.AttachErr
		return resp, nil
	}
	defer conn.Close()

	hold := req.AttachHold
	if hold <= 0 {
		hold = 200 * time.Millisecond
	}

	// Optional write input (live-roundtrip).
	if len(req.AttachInput) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, req.AttachInput); err != nil {
			resp.ExitCode = 1
			resp.AttachErr = err.Error()
			resp.Combined = resp.AttachErr
			return resp, nil
		}
	}

	// Read until hold or error.
	deadline := time.Now().Add(hold)
	_ = conn.SetReadDeadline(deadline)
	var out bytes.Buffer
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		mt, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			// unclean or timeout
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			if ce, ok := err.(*websocket.CloseError); ok {
				resp.AttachErr = ce.Error()
				resp.ExitCode = 1
				break
			}
			if resp.AttachErr == "" && !strings.Contains(err.Error(), "timeout") {
				resp.AttachErr = err.Error()
				resp.ExitCode = 1
			}
			break
		}
		if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
			out.Write(data)
		}
	}
	resp.ReceivedOutput = out.String()
	if resp.ExitCode == 0 && resp.AttachErr == "" {
		resp.ExitCode = 0
	}
	resp.Combined = strings.TrimSpace(resp.ReceivedOutput + "\n" + resp.AttachErr)
	// Best-effort: product may call OnLocalRestore on client path; for raw WS
	// leaves TermRestored may stay false unless implementer wires server-side.
	return resp, nil
}

func runKeepaliveWS(t *testing.T, req *Request, resp *Response, serverURL string) (*Response, error) {
	t.Helper()
	sid := strings.TrimSpace(req.AttachSessionID)
	if sid == "" {
		sid = "sess-live"
	}
	wsURL := toWSURL(serverURL) + "/api/agent-run/sessions/" + url.PathEscape(sid) + "/attach"

	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+lib.TestPassword)
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	conn, httpResp, err := dialer.Dial(wsURL, hdr)
	if httpResp != nil {
		resp.WSHTTPStatus = httpResp.StatusCode
	}
	if err != nil {
		resp.ExitCode = 1
		resp.AttachErr = err.Error()
		resp.Combined = resp.AttachErr
		resp.Stderr = resp.AttachErr
		return resp, nil
	}
	defer conn.Close()

	var pings atomic.Int64
	conn.SetPingHandler(func(appData string) error {
		pings.Add(1)
		// Default pong response.
		deadline := time.Now().Add(time.Second)
		return conn.WriteControl(websocket.PongMessage, []byte(appData), deadline)
	})

	wait := req.WaitForPing
	if wait <= 0 {
		wait = 400 * time.Millisecond
	}
	// Drain reads so ping handlers fire (gorilla delivers control during Read).
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(40 * time.Millisecond))
		_, _, err := conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// connection closed
			break
		}
	}
	resp.PingCount = int(pings.Load())
	resp.ExitCode = 0
	resp.Combined = fmt.Sprintf("pings=%d", resp.PingCount)
	return resp, nil
}

func toWSURL(httpURL string) string {
	u, err := url.Parse(httpURL)
	if err != nil {
		return strings.Replace(httpURL, "http://", "ws://", 1)
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	return u.String()
}

func seedSessionMetas(home string, seeds []SessionSeed) error {
	for _, s := range seeds {
		id := strings.TrimSpace(s.SessionID)
		if id == "" {
			return fmt.Errorf("seed missing session_id")
		}
		runner := s.Runner
		if runner == "" {
			runner = "grok"
		}
		status := s.Status
		if status == "" {
			status = "finished"
		}
		created := s.CreatedAt
		if created == "" {
			created = "2024-01-01T00:00:00Z"
		}
		updated := s.UpdatedAt
		if updated == "" {
			updated = created
		}
		dir := filepath.Join(home, "sessions", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		meta := map[string]string{
			"session_id": id,
			"runner":     runner,
			"status":     status,
			"created_at": created,
			"updated_at": updated,
		}
		if ts := strings.TrimSpace(s.TerminalSessionID); ts != "" {
			meta["terminal_session_id"] = ts
		}
		if rs := strings.TrimSpace(s.RunnerSessionID); rs != "" {
			meta["runner_session_id"] = rs
		}
		if ws := strings.TrimSpace(s.Workspace); ws != "" {
			meta["workspace"] = ws
		}
		data, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "meta.json"), append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func containsJSONFlag(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func withBearerAuth(validToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != validToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("server not ready: %s", url)
}

func runAgentInProcess(argv []string) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	// Parallel-safe: inject writers — never reassign os.Stdout/os.Stderr in harness.
	var stdoutBuf, stderrBuf bytes.Buffer
	runErr := agentcli.RunWithWriters(agentcli.RemoteProfile(), argv, &stdoutBuf, &stderrBuf)

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	exitCode := 0
	if runErr != nil {
		if !strings.Contains(stderr, "Error:") {
			stderr += fmt.Sprintf("Error: %v\n", runErr)
		}
		exitCode = 1
	}
	return exitCode, stdout, stderr, nil
}

// silence unused imports in generated harnesses that share this block.
var (
	_ = time.Second
	_ = io.Discard
	_ = context.Background
)
```
