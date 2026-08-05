# remote-agent ssh — L2 doctests (P2: session store + local relay + serve)

Plan phase **P2** (classic TDD): on-disk session persistence, localhost TCP
relay splice, and `ServeService.Start(ctx)` lifecycle with injectable Dial —
**without** a live remote ai-critic server or Cloudflare.

Sibling of sealed P1 tree `cmd/agentcli/sshcmd/tests` (12 leaves; leave untouched).

Package under test (extends P1 surface; new symbols **RED** until implementer):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd
```

Out of scope: real agent WebSocket to remote host; public exposure; OpenSSH
client login (P3); agentcli top-level switch (P3).

# DSN (Domain Specific Notion)

**remote-agent ssh P2** materializes the serve-side tunnel local half: session
JSON on disk, a loopback TCP relay, and a blocking Start that owns lifecycle.

**Participants**

- **FileSessionStore** — `{Root}/ssh-sessions/{profileID}.json`; Load / Save / Clear.
- **Session** — LocalPort, User, ConfigDir, ServePID, ProfileID, Alive (P1 type).
- **LocalRelay** — listen `127.0.0.1:0`; each Accept dials remote via **DialFunc**,
  bidirectional copy; Close stops listener and active conns.
- **DialFunc** — injectable remote side of the tunnel (`func() (net.Conn, error)`).
- **ServeService** — Start(ctx): listen + relay + Save Alive session; on cancel:
  Clear session + close relay.

**Behaviors**

- Save then Load round-trips session fields; missing file → nil session, nil error.
- Clear removes profile session; subsequent Load is nil.
- Corrupt JSON on Load → error.
- Load with Alive + non-zero ServePID of a dead process → session not Alive.
- Relay: client dials LocalPort, payload echoed via DialFunc side.
- Relay Close → further dials fail.
- Serve Start with Dial: session file Alive + LocalPort > 0; client can echo;
  cancel ctx → session cleared, listen port closed.
- Serve Start without Dial → clear configuration error.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/sshcmd/session_tests/     [Request{Scenario, Root, …}]
│                                      Run dispatches by Scenario (L2 library)
├── session-store/                     # FileSessionStore
│   ├── save-load/                     # Save then Load → equal fields
│   ├── load-missing/                  # no file → nil, nil
│   ├── clear-then-load/               # Clear then Load → nil
│   ├── dead-pid-not-alive/            # dead ServePID → !Alive
│   └── corrupt-json/                  # bad JSON → Load error
├── local-relay/                       # LocalRelay + DialFunc
│   ├── echo/                          # write "hi" → read "hi"
│   └── close-rejects/                 # Close → further dial fails
└── serve/                             # ServeService.Start(ctx)
    ├── start-cancel-lifecycle/        # start → echo → cancel → clear + close
    └── missing-dial/                  # no Dial → configuration error
```

**Significance order:** surface (session-store | local-relay | serve) →
operation / lifecycle outcome within surface.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `session-store/save-load` | Save then Load same profile → equal fields |
| 2 | `session-store/load-missing` | Load missing file → nil session, nil error |
| 3 | `session-store/clear-then-load` | Clear after Save → Load nil |
| 4 | `session-store/dead-pid-not-alive` | Alive + dead ServePID → not Alive |
| 5 | `session-store/corrupt-json` | Corrupt JSON → Load error |
| 6 | `local-relay/echo` | Client dials port; DialFunc echo side returns payload |
| 7 | `local-relay/close-rejects` | After Close, dial to former port fails |
| 8 | `serve/start-cancel-lifecycle` | Start writes Alive session; echo works; cancel clears + closes |
| 9 | `serve/missing-dial` | Start without Dial → clear error |

## Exported APIs (implementer contract)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd` (same as P1):

| Symbol | Role |
|--------|------|
| `FileSessionStore` | `struct { Root string }` — session JSON under `{Root}/ssh-sessions/{id}.json` |
| `(*FileSessionStore).Load` | `(profileID string) (*Session, error)` — missing → `(nil, nil)`; corrupt → error; dead ServePID → Alive false |
| `(*FileSessionStore).Save` | `(sess *Session) error` — writes JSON (snake_case wire keys) |
| `(*FileSessionStore).Clear` | `(profileID string) error` — removes session file (idempotent OK) |
| `DialFunc` | `func() (net.Conn, error)` — remote tunnel end per accept |
| `LocalRelay` | listen + splice; fields include `Dial DialFunc` |
| `(*LocalRelay).Start` | `() error` — listen `127.0.0.1:0`, accept loop in background |
| `(*LocalRelay).LocalPort` | `() int` — bound port after Start |
| `(*LocalRelay).Close` | `() error` — stop listener and active conns |
| `ServeService` | store + profile + Dial (+ optional User, ConfigDir, ServePID) |
| `(*ServeService).Start` | `(ctx context.Context) error` — blocks until cancel; then Clear + close relay |

JSON session file shape:

```json
{
  "local_port": 51234,
  "user": "agent",
  "config_dir": "/abs/path",
  "serve_pid": 12345,
  "profile_id": "default",
  "alive": true
}
```

Serve Start also creates `ConfigDir` and writes at least `ssh_config` mentioning
the bound LocalPort (keys may be placeholders).

## How to Run

From module root (`external/ai-critic-master-2026-07-31`):

```sh
doctest vet ./cmd/agentcli/sshcmd/session_tests
doctest test ./cmd/agentcli/sshcmd/session_tests
# P1 must stay green:
doctest test ./cmd/agentcli/sshcmd/tests
```

```go
import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/doctest/session"
)

// Scenario selects which P2 surface Run exercises (MECE dispatch key).
type Scenario string

const (
	ScenarioSessionSaveLoad    Scenario = "session-save-load"
	ScenarioSessionLoadMissing Scenario = "session-load-missing"
	ScenarioSessionClear       Scenario = "session-clear"
	ScenarioSessionDeadPID     Scenario = "session-dead-pid"
	ScenarioSessionCorrupt     Scenario = "session-corrupt"
	ScenarioRelayEcho          Scenario = "relay-echo"
	ScenarioRelayClose         Scenario = "relay-close"
	ScenarioServeLifecycle     Scenario = "serve-lifecycle"
	ScenarioServeMissingDial   Scenario = "serve-missing-dial"
)

// Request configures absolute paths and scenario inputs (parallel-safe).
// All paths under d.DOCTEST_CASE / t.TempDir — no Setenv/Chdir.
type Request struct {
	Scenario  Scenario
	ProfileID string
	// Root is absolute store root for FileSessionStore ({Root}/ssh-sessions/…).
	Root string
	// ConfigDir is absolute config dir for Serve (ssh_config, keys).
	ConfigDir string

	// SessionToSave is used by save-load / clear / dead-pid scenarios.
	SessionToSave *sshcmd.Session
	// EchoPayload is written by the client through the relay (default "hi").
	EchoPayload string
}

// Response captures outcomes for session store, relay, and serve leaves.
type Response struct {
	// Session store
	Loaded   *sshcmd.Session
	LoadErr  string
	SaveErr  string
	ClearErr string

	// Relay / ports
	LocalPort         int
	EchoGot           string
	EchoErr           string
	DialAfterCloseErr string
	RelayStartErr     string
	RelayCloseErr     string

	// Serve
	ServeErr            string
	SessionAfterStart   *sshcmd.Session
	SessionAfterStop    *sshcmd.Session
	EchoThroughServe    string
	PortClosedAfterStop bool
	SSHConfigExists     bool
	SSHConfigMentionsPort bool
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	if req.EchoPayload == "" {
		req.EchoPayload = "hi"
	}
	if req.Root == "" {
		req.Root = filepath.Join(d.DOCTEST_CASE, "store-root")
	}
	if req.ConfigDir == "" {
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "session-config")
	}

	resp := &Response{}
	store := &sshcmd.FileSessionStore{Root: req.Root}

	switch req.Scenario {
	case ScenarioSessionSaveLoad:
		runSessionSaveLoad(t, req, store, resp)
	case ScenarioSessionLoadMissing:
		runSessionLoadMissing(t, req, store, resp)
	case ScenarioSessionClear:
		runSessionClear(t, req, store, resp)
	case ScenarioSessionDeadPID:
		runSessionDeadPID(t, req, store, resp)
	case ScenarioSessionCorrupt:
		runSessionCorrupt(t, req, store, resp)
	case ScenarioRelayEcho:
		runRelayEcho(t, req, resp)
	case ScenarioRelayClose:
		runRelayClose(t, req, resp)
	case ScenarioServeLifecycle:
		runServeLifecycle(t, req, store, resp)
	case ScenarioServeMissingDial:
		runServeMissingDial(t, req, store, resp)
	default:
		return nil, fmt.Errorf("unknown scenario %q", req.Scenario)
	}
	return resp, nil
}

func runSessionSaveLoad(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	sess := req.SessionToSave
	if sess == nil {
		t.Fatal("SessionToSave required for save-load")
	}
	if err := store.Save(sess); err != nil {
		resp.SaveErr = err.Error()
		return
	}
	loaded, err := store.Load(req.ProfileID)
	if err != nil {
		resp.LoadErr = err.Error()
		return
	}
	resp.Loaded = loaded
}

func runSessionLoadMissing(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	loaded, err := store.Load(req.ProfileID)
	if err != nil {
		resp.LoadErr = err.Error()
	}
	resp.Loaded = loaded
}

func runSessionClear(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	sess := req.SessionToSave
	if sess == nil {
		t.Fatal("SessionToSave required for clear")
	}
	if err := store.Save(sess); err != nil {
		resp.SaveErr = err.Error()
		return
	}
	if err := store.Clear(req.ProfileID); err != nil {
		resp.ClearErr = err.Error()
		return
	}
	loaded, err := store.Load(req.ProfileID)
	if err != nil {
		resp.LoadErr = err.Error()
	}
	resp.Loaded = loaded
}

func runSessionDeadPID(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	sess := req.SessionToSave
	if sess == nil {
		t.Fatal("SessionToSave required for dead-pid")
	}
	if err := store.Save(sess); err != nil {
		resp.SaveErr = err.Error()
		return
	}
	loaded, err := store.Load(req.ProfileID)
	if err != nil {
		resp.LoadErr = err.Error()
		return
	}
	resp.Loaded = loaded
}

func runSessionCorrupt(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	dir := filepath.Join(req.Root, "ssh-sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	path := filepath.Join(dir, req.ProfileID+".json")
	if err := os.WriteFile(path, []byte("not-json{{{"), 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	loaded, err := store.Load(req.ProfileID)
	if err != nil {
		resp.LoadErr = err.Error()
	}
	resp.Loaded = loaded
}

func runRelayEcho(t *testing.T, req *Request, resp *Response) {
	t.Helper()
	relay := &sshcmd.LocalRelay{Dial: echoDialFunc()}
	if err := relay.Start(); err != nil {
		resp.RelayStartErr = err.Error()
		return
	}
	defer relay.Close()
	resp.LocalPort = relay.LocalPort()
	got, err := clientEcho(resp.LocalPort, req.EchoPayload)
	if err != nil {
		resp.EchoErr = err.Error()
		return
	}
	resp.EchoGot = got
}

func runRelayClose(t *testing.T, req *Request, resp *Response) {
	t.Helper()
	_ = req
	relay := &sshcmd.LocalRelay{Dial: echoDialFunc()}
	if err := relay.Start(); err != nil {
		resp.RelayStartErr = err.Error()
		return
	}
	resp.LocalPort = relay.LocalPort()
	if err := relay.Close(); err != nil {
		resp.RelayCloseErr = err.Error()
	}
	// Further dial to the former port must fail.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", resp.LocalPort), 500*time.Millisecond)
	if err != nil {
		resp.DialAfterCloseErr = err.Error()
		return
	}
	conn.Close()
	resp.DialAfterCloseErr = ""
}

func runServeLifecycle(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	if err := os.MkdirAll(req.ConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}

	svc := &sshcmd.ServeService{
		Store:     store,
		ProfileID: req.ProfileID,
		Dial:      echoDialFunc(),
		User:      "agent",
		ConfigDir: req.ConfigDir,
		ServePID:  os.Getpid(), // live process for this test process
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var startErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startErr = svc.Start(ctx)
	}()

	// Wait until session appears Alive with LocalPort > 0.
	var afterStart *sshcmd.Session
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err := store.Load(req.ProfileID)
		if err == nil && loaded != nil && loaded.Alive && loaded.LocalPort > 0 {
			afterStart = loaded
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	resp.SessionAfterStart = afterStart
	if afterStart == nil {
		cancel()
		wg.Wait()
		if startErr != nil {
			resp.ServeErr = startErr.Error()
		} else {
			resp.ServeErr = "timeout waiting for Alive session after Start"
		}
		return
	}
	resp.LocalPort = afterStart.LocalPort

	// Echo through serve port.
	got, err := clientEcho(afterStart.LocalPort, req.EchoPayload)
	if err != nil {
		resp.EchoErr = err.Error()
	} else {
		resp.EchoThroughServe = got
	}

	// Config artifacts.
	sshCfgPath := filepath.Join(req.ConfigDir, "ssh_config")
	if data, err := os.ReadFile(sshCfgPath); err == nil {
		resp.SSHConfigExists = true
		portStr := fmt.Sprintf("%d", afterStart.LocalPort)
		resp.SSHConfigMentionsPort = strings.Contains(string(data), portStr)
	}

	// Teardown via cancel.
	cancel()
	wg.Wait()
	if startErr != nil && startErr != context.Canceled {
		// Allow context.Canceled; record other errors.
		if !strings.Contains(startErr.Error(), "canceled") && !strings.Contains(startErr.Error(), "cancelled") {
			resp.ServeErr = startErr.Error()
		}
	}

	afterStop, err := store.Load(req.ProfileID)
	if err != nil {
		resp.LoadErr = err.Error()
	}
	resp.SessionAfterStop = afterStop

	// Port should no longer accept connections.
	conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", afterStart.LocalPort), 500*time.Millisecond)
	if dialErr != nil {
		resp.PortClosedAfterStop = true
		resp.DialAfterCloseErr = dialErr.Error()
	} else {
		conn.Close()
		resp.PortClosedAfterStop = false
	}
}

func runServeMissingDial(t *testing.T, req *Request, store *sshcmd.FileSessionStore, resp *Response) {
	t.Helper()
	svc := &sshcmd.ServeService{
		Store:     store,
		ProfileID: req.ProfileID,
		Dial:      nil, // missing
		User:      "agent",
		ConfigDir: req.ConfigDir,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := svc.Start(ctx)
	if err != nil {
		resp.ServeErr = err.Error()
	}
}

// echoDialFunc returns a DialFunc whose remote conn echoes bytes.
func echoDialFunc() sshcmd.DialFunc {
	return func() (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			buf := make([]byte, 4096)
			for {
				n, err := b.Read(buf)
				if n > 0 {
					if _, werr := b.Write(buf[:n]); werr != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()
		return a, nil
	}
}

// clientEcho dials 127.0.0.1:port, writes payload, reads same length back.
func clientEcho(port int, payload string) (string, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, payload); err != nil {
		return "", err
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// deadPID starts a short-lived process, kills it, and returns its former PID.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep for dead PID: %v", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	return pid
}

// sampleSession builds a full Session for store leaves (ServePID optional).
func sampleSession(profileID, configDir string, port, servePID int, alive bool) *sshcmd.Session {
	return &sshcmd.Session{
		LocalPort: port,
		User:      "agent",
		ConfigDir: configDir,
		ServePID:  servePID,
		ProfileID: profileID,
		Alive:     alive,
	}
}
```
