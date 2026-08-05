# remote-agent ssh — L2 doctests (P4: agent duplex SSH tunnel)

Plan phase **P4** (classic TDD): authenticated duplex stream from laptop
`ServeService` Dial through agent HTTP/WebSocket to a remote **AdhocServer**,
plus agentcli `--serve` wiring that sets non-nil Dial when a Client is provided.

Sibling of sealed P1 `cmd/agentcli/sshcmd/tests` (12), P2
`cmd/agentcli/sshcmd/session_tests` (9), and P3 `cmd/agentcli/sshcmd/ssh_tests`
(8) — **do not edit** those trees or their ASSERT files.

Packages under test (new symbols **RED** until implementer):

```text
github.com/xhd2015/ai-critic/server/sshtunnel   # session + WS tunnel → AdhocServer
github.com/xhd2015/ai-critic/client              # CreateSSHSession + SSHTunnelDial
github.com/xhd2015/ai-critic/cmd/agentcli        # serve starter wires Client → Dial
github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd  # compose ServeService + CryptoSSHRunner (existing)
```

Out of scope: Cloudflare-specific paths; Unison e2e; multi-profile polish;
full `server.RegisterAPI` bootstrap (tests use httptest + `sshtunnel.RegisterAPI`).

Network model under test:

```text
LocalRelay Accept
  → DialFunc (client.SSHTunnelDial)
       → WebSocket binary /api/remote-agent/ssh/sessions/{id}/tunnel
            → server dials 127.0.0.1:adhocPort
                 → AdhocServer
CryptoSSHRunner → LocalRelay LocalPort → … → remote command
```

# DSN (Domain Specific Notion)

**remote-agent ssh P4** replaces the production gap `Dial = nil` with an
authenticated agent tunnel: create a remote SSH session, open a duplex WS as
`net.Conn`, and feed that Dial into ServeService.

**Participants**

- **sshtunnel.Manager** — process-local session table; CreateSession starts
  AdhocServer (or test BackendDial); Destroy tears down.
- **sshtunnel HTTP/WS routes** — POST/DELETE session; GET tunnel upgrades to
  WebSocket; each WS = one TCP to the session backend.
- **client.Client** — bearer-token HTTP; CreateSSHSession / DestroySSHSession /
  SSHTunnelDial (`net.Conn` over binary WS frames).
- **ServeService + DialFunc** — existing P2; Dial now comes from client tunnel.
- **CryptoSSHRunner** — existing P3; runs remote command via LocalPort.
- **agentcli serve starter** — when Client is set, creates remote session and
  assigns non-nil Dial before ServeService.Start.

**Behaviors**

- Authorized token + public key → CreateSession returns session_id (+ host key).
- Unauthorized token → CreateSession fails (non-2xx / error).
- SSHTunnelDial after Destroy → fail.
- Binary splice: write through Dial → echo backend returns same bytes.
- Full stack: CreateSession → Serve Dial=SSHTunnelDial → CryptoSSHRunner
  `echo p4-ok` → stdout contains needle.
- Serve wiring helper with Client → non-nil DialFunc (no Start required).

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/sshcmd/tunnel_tests/     [Request{Scenario, …}]
│                                      Run dispatches by Scenario (L2 httptest)
├── ssh-session/                       # HTTP CreateSession (named ssh-session: "session" collides with doctest/session)
│   ├── create-authorized/             # valid Bearer → session_id
│   └── create-unauthorized/           # bad/missing token → fail
├── tunnel/                            # client.SSHTunnelDial duplex
│   ├── binary-echo/                   # raw bytes via WS→TCP echo backend
│   └── after-destroy-fails/           # Destroy then Dial → error
├── through-serve/                     # ServeService + CryptoSSHRunner
│   └── remote-command/                # echo p4-ok through agent tunnel
└── serve-wiring/                      # agentcli starter / BuildSSHTunnelDial
    └── dial-from-client/              # Client present → non-nil Dial
```

**Significance order:** surface (session | tunnel | through-serve | serve-wiring)
→ auth/lifecycle/outcome within surface.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `ssh-session/create-authorized` | Valid token CreateSSHSession → SessionID non-empty |
| 2 | `ssh-session/create-unauthorized` | Wrong/missing token → create error; no session id |
| 3 | `tunnel/binary-echo` | SSHTunnelDial write/read payload through WS splice |
| 4 | `tunnel/after-destroy-fails` | DestroySSHSession then SSHTunnelDial → error |
| 5 | `through-serve/remote-command` | Serve + CryptoSSHRunner `echo p4-ok` via client Dial |
| 6 | `serve-wiring/dial-from-client` | BuildSSHTunnelDial / starter with Client → Dial non-nil |

## Exported APIs (implementer contract)

### Server — `github.com/xhd2015/ai-critic/server/sshtunnel`

| Symbol | Role |
|--------|------|
| `Manager` | Session table + Adhoc (or BackendDial) lifecycle |
| `NewManager` | `() *Manager` |
| `(*Manager).RequiredToken` | Optional `string`; when non-empty, handlers require `Authorization: Bearer <token>` (L2-friendly without full auth middleware) |
| `(*Manager).BackendDial` | Optional `func() (net.Conn, error)`; when set, CreateSession does **not** start AdhocServer — tunnel dials this backend (binary-echo leaf). Production: nil → start AdhocServer per session |
| `RegisterAPI` | `(mux *http.ServeMux)` — default process Manager |
| `RegisterAPIWithManager` | `(mux *http.ServeMux, m *Manager)` — inject Manager (tests) |

**Routes** (match existing `RegisterAPI` + `/api/remote-agent/...` style):

| Method | Path | Behavior |
|--------|------|----------|
| `POST` | `/api/remote-agent/ssh/sessions` | CreateSession; JSON body `{"public_key":"<openssh line or empty>"}`; 200 `{"session_id","user","host_key?"}` |
| `DELETE` | `/api/remote-agent/ssh/sessions/{id}` | Destroy session + Adhoc/backend; 204/200 |
| `GET` | `/api/remote-agent/ssh/sessions/{id}/tunnel` | WebSocket upgrade; **binary** frames duplex to one TCP dial of session backend; close WS → close TCP |

Auth: production mounts under same agent-token middleware as other `/api/*`.
L2 tests set `Manager.RequiredToken` and `client.Token` without `server.bootstrap`.

CreateSession (production path, `BackendDial == nil`):

1. Mint `session_id`.
2. Start `sshcmd.AdhocServer` (User `"agent"`); if `public_key` present, parse and `SetAuthorizedKeys`.
3. Store session; return id + OpenSSH host key string when available.

Tunnel handler: each successful WS upgrade → one `net.Dial` (or BackendDial) to session backend; bidirectional copy of binary payload (text frames optional to reject).

Wire into production `server.RegisterAPI` (implementer):

```go
sshtunnel.RegisterAPI(mux)
```

### Client — `github.com/xhd2015/ai-critic/client`

| Symbol | Role |
|--------|------|
| `CreateSSHSessionRequest` | `struct { PublicKey string \`json:"public_key,omitempty"\` }` |
| `SSHSessionInfo` | `struct { SessionID string; User string; HostKey string }` (JSON snake_case) |
| `(*Client).CreateSSHSession` | `(req CreateSSHSessionRequest) (*SSHSessionInfo, error)` — POST sessions |
| `(*Client).DestroySSHSession` | `(sessionID string) error` — DELETE sessions/{id} |
| `(*Client).SSHTunnelDial` | `(sessionID string) (net.Conn, error)` — WS dial; returned conn implements duplex binary I/O as `net.Conn` |
| `(*Client).SSHTunnelDialFunc` | `(sessionID string) func() (net.Conn, error)` — returns DialFunc for `ServeService.Dial` (each call = new WS) |

WS URL: derive from `Client.Server` (`http`→`ws`, `https`→`wss`) +
`/api/remote-agent/ssh/sessions/{id}/tunnel`; send `Authorization: Bearer` when Token set
(same pattern as HTTP `NewRequest`).

### agentcli serve wiring — `github.com/xhd2015/ai-critic/cmd/agentcli`

| Symbol | Role |
|--------|------|
| `BuildSSHTunnelDial` | `(c *client.Client, publicKey string) (dial sshcmd.DialFunc, info *client.SSHSessionInfo, err error)` — CreateSSHSession then wrap `SSHTunnelDialFunc`; **exported for unit leaf** |
| `sshServeStarter` / `NewSSHServeStarter` | Accept optional `*client.Client` (+ key material path); when Client non-nil, call BuildSSHTunnelDial and set `ServeService.Dial`; when nil, Dial remains nil (P3 behavior / missing-dial) |

Key exchange (v1 sound default, production serve path):

- Client key: ed25519 under ConfigDir (or `sshcmd.GenerateClientKeyPair` in tests).
- CreateSession uploads public key; remote AdhocServer authorizes it.
- CryptoSSHRunner uses matching private signer.

Tests may use `InsecureIgnoreHostKey: true` (as P3).

### Compose (existing, unchanged contract)

```go
info, err := c.CreateSSHSession(client.CreateSSHSessionRequest{PublicKey: pubOpenSSH})
dial := c.SSHTunnelDialFunc(info.SessionID)
svc := &sshcmd.ServeService{Store: store, ProfileID: id, Dial: dial, User: "agent", ConfigDir: dir, ServePID: os.Getpid()}
// Start(ctx); CryptoSSHRunner{Signer: kp.Signer, ...}.Run(sess, []string{"echo","p4-ok"}, …)
```

## How to Run

From module root (`external/ai-critic-master-2026-07-31`):

```sh
doctest vet ./cmd/agentcli/sshcmd/tunnel_tests
doctest test ./cmd/agentcli/sshcmd/tunnel_tests
# P1–P3 must stay green:
doctest test ./cmd/agentcli/sshcmd/tests
doctest test ./cmd/agentcli/sshcmd/session_tests
doctest test ./cmd/agentcli/sshcmd/ssh_tests
```

```go
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/ai-critic/server/sshtunnel"
	"github.com/xhd2015/doctest/session"
	"golang.org/x/crypto/ssh"
)

// Scenario selects which P4 surface Run exercises (MECE dispatch key).
type Scenario string

const (
	ScenarioSessionCreateOK     Scenario = "session-create-authorized"
	ScenarioSessionCreateUnauth Scenario = "session-create-unauthorized"
	ScenarioTunnelBinaryEcho    Scenario = "tunnel-binary-echo"
	ScenarioTunnelAfterDestroy  Scenario = "tunnel-after-destroy"
	ScenarioThroughServeCommand Scenario = "through-serve-remote-command"
	ScenarioServeWiringDial     Scenario = "serve-wiring-dial-from-client"
)

// Request configures absolute paths and scenario inputs (parallel-safe).
// All paths under d.DOCTEST_CASE — no Setenv/Chdir.
type Request struct {
	Scenario  Scenario
	ProfileID string
	// Root is FileSessionStore root ({Root}/ssh-sessions/…).
	Root string
	// ConfigDir is serve / key material dir.
	ConfigDir string

	// Token is the bearer token the client sends (default "test-token").
	Token string
	// ManagerToken is Manager.RequiredToken (default "test-token"; empty skips check).
	// create-unauthorized sets Token != ManagerToken.
	ManagerToken string

	// PublicKeyOpenSSH is uploaded on CreateSession when non-empty.
	PublicKeyOpenSSH string

	// EchoPayload for binary-echo (default "p4-tunnel-hi").
	EchoPayload string
	// RemoteArgv / EchoNeedle for through-serve (default echo p4-ok).
	RemoteArgv []string
	EchoNeedle string
}

// Response captures session, tunnel, serve, and wiring outcomes.
type Response struct {
	// Session create
	SessionID   string
	CreateErr   string
	User        string
	HostKey     string
	HTTPStatus  int

	// Tunnel
	TunnelDialErr string
	EchoWrote     string
	EchoRead      string
	EchoErr       string
	DestroyErr    string

	// Through-serve / CryptoSSHRunner
	ServeErr          string
	SessionAfterStart *sshcmd.Session
	RelayLocalPort    int
	RunnerErr         string
	Stdout            string
	Stderr            string

	// Serve wiring
	DialIsNil   bool
	WiringErr   string
	WiringSessID string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	applyDefaults(d, req)

	resp := &Response{}
	switch req.Scenario {
	case ScenarioSessionCreateOK:
		runSessionCreate(t, d, req, resp, true)
	case ScenarioSessionCreateUnauth:
		runSessionCreate(t, d, req, resp, false)
	case ScenarioTunnelBinaryEcho:
		runTunnelBinaryEcho(t, d, req, resp)
	case ScenarioTunnelAfterDestroy:
		runTunnelAfterDestroy(t, d, req, resp)
	case ScenarioThroughServeCommand:
		runThroughServeCommand(t, d, req, resp)
	case ScenarioServeWiringDial:
		runServeWiringDial(t, d, req, resp)
	default:
		return nil, fmt.Errorf("unknown scenario %q", req.Scenario)
	}
	return resp, nil
}

func applyDefaults(d *session.Doctest, req *Request) {
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	if req.Root == "" {
		req.Root = filepath.Join(d.DOCTEST_CASE, "store-root")
	}
	if req.ConfigDir == "" {
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "session-config")
	}
	if req.Token == "" {
		req.Token = "test-token"
	}
	if req.ManagerToken == "" && req.Scenario != ScenarioSessionCreateUnauth {
		req.ManagerToken = "test-token"
	}
	if req.EchoPayload == "" {
		req.EchoPayload = "p4-tunnel-hi"
	}
	if len(req.RemoteArgv) == 0 {
		req.RemoteArgv = []string{"echo", "p4-ok"}
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "p4-ok"
	}
}

// --- server + client harness ---

func startTunnelServer(t *testing.T, managerToken string, backendDial func() (net.Conn, error)) (*httptest.Server, *sshtunnel.Manager, *client.Client) {
	t.Helper()
	m := sshtunnel.NewManager()
	m.RequiredToken = managerToken
	if backendDial != nil {
		m.BackendDial = backendDial
	}
	mux := http.NewServeMux()
	sshtunnel.RegisterAPIWithManager(mux, m)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	// Client constructed by caller with Token; base URL only here.
	c := client.New(srv.URL, "")
	return srv, m, c
}

func clientWithToken(base *client.Client, token string) *client.Client {
	return client.New(base.Server, token)
}

func mustKeyPair(t *testing.T) *sshcmd.ClientKeyPair {
	t.Helper()
	kp, err := sshcmd.GenerateClientKeyPair()
	if err != nil {
		t.Fatalf("GenerateClientKeyPair: %v", err)
	}
	if kp == nil || kp.Signer == nil || kp.Public == nil {
		t.Fatal("GenerateClientKeyPair returned incomplete pair")
	}
	return kp
}

func publicKeyOpenSSH(pub ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

// startTCPEcho listens on 127.0.0.1:0 and echoes all bytes until EOF.
func startTCPEcho(t *testing.T) (addr string, closeFn func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo listen: %v", err)
	}
	var wg sync.WaitGroup
	done := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()
	closeFn = func() {
		close(done)
		_ = ln.Close()
		wg.Wait()
	}
	t.Cleanup(closeFn)
	return ln.Addr().String(), closeFn
}

// --- scenarios ---

func runSessionCreate(t *testing.T, d *session.Doctest, req *Request, resp *Response, authorized bool) {
	t.Helper()
	_ = d
	mgrToken := req.ManagerToken
	if !authorized {
		// Server expects a token; client sends a different or empty one.
		if mgrToken == "" {
			mgrToken = "expected-token"
			req.ManagerToken = mgrToken
		}
		if req.Token == mgrToken {
			req.Token = "wrong-token"
		}
	}
	_, _, base := startTunnelServer(t, mgrToken, nil)
	c := clientWithToken(base, req.Token)

	pub := req.PublicKeyOpenSSH
	if pub == "" && authorized {
		kp := mustKeyPair(t)
		pub = publicKeyOpenSSH(kp.Public)
	}
	info, err := c.CreateSSHSession(client.CreateSSHSessionRequest{PublicKey: pub})
	if err != nil {
		resp.CreateErr = err.Error()
		return
	}
	if info != nil {
		resp.SessionID = info.SessionID
		resp.User = info.User
		resp.HostKey = info.HostKey
	}
}

func runTunnelBinaryEcho(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	echoAddr, _ := startTCPEcho(t)
	backend := func() (net.Conn, error) {
		return net.DialTimeout("tcp", echoAddr, 5*time.Second)
	}
	_, _, base := startTunnelServer(t, req.ManagerToken, backend)
	c := clientWithToken(base, req.Token)

	info, err := c.CreateSSHSession(client.CreateSSHSessionRequest{})
	if err != nil {
		resp.CreateErr = err.Error()
		return
	}
	resp.SessionID = info.SessionID

	conn, err := c.SSHTunnelDial(info.SessionID)
	if err != nil {
		resp.TunnelDialErr = err.Error()
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	payload := req.EchoPayload
	resp.EchoWrote = payload
	if _, err := io.WriteString(conn, payload); err != nil {
		resp.EchoErr = "write: " + err.Error()
		return
	}
	buf := make([]byte, len(payload)+16)
	n, err := io.ReadAtLeast(conn, buf, len(payload))
	if err != nil {
		resp.EchoErr = "read: " + err.Error()
		return
	}
	resp.EchoRead = string(buf[:n])
}

func runTunnelAfterDestroy(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	// Backend optional; Adhoc or echo both fine — we only need session lifecycle.
	echoAddr, _ := startTCPEcho(t)
	backend := func() (net.Conn, error) {
		return net.DialTimeout("tcp", echoAddr, 5*time.Second)
	}
	_, _, base := startTunnelServer(t, req.ManagerToken, backend)
	c := clientWithToken(base, req.Token)

	info, err := c.CreateSSHSession(client.CreateSSHSessionRequest{})
	if err != nil {
		resp.CreateErr = err.Error()
		return
	}
	resp.SessionID = info.SessionID

	if err := c.DestroySSHSession(info.SessionID); err != nil {
		resp.DestroyErr = err.Error()
		// Still attempt dial — destroy must invalidate even if API reports soft error.
	}
	conn, err := c.SSHTunnelDial(info.SessionID)
	if err != nil {
		resp.TunnelDialErr = err.Error()
		return
	}
	// Unexpected success
	_ = conn.Close()
	resp.TunnelDialErr = ""
}

func runThroughServeCommand(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	if err := os.MkdirAll(req.ConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	kp := mustKeyPair(t)
	pub := publicKeyOpenSSH(kp.Public)

	// Production path: BackendDial nil → Manager starts AdhocServer with pubkey.
	_, _, base := startTunnelServer(t, req.ManagerToken, nil)
	c := clientWithToken(base, req.Token)

	info, err := c.CreateSSHSession(client.CreateSSHSessionRequest{PublicKey: pub})
	if err != nil {
		resp.CreateErr = err.Error()
		return
	}
	resp.SessionID = info.SessionID
	resp.HostKey = info.HostKey

	dial := c.SSHTunnelDialFunc(info.SessionID)
	store := &sshcmd.FileSessionStore{Root: req.Root}
	svc := &sshcmd.ServeService{
		Store:     store,
		ProfileID: req.ProfileID,
		Dial:      dial,
		User:      "agent",
		ConfigDir: req.ConfigDir,
		ServePID:  os.Getpid(),
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
			resp.ServeErr = "timeout waiting for Alive session"
		}
		return
	}
	resp.RelayLocalPort = afterStart.LocalPort

	var stdout, stderr bytes.Buffer
	runner := &sshcmd.CryptoSSHRunner{
		Signer:                kp.Signer,
		Stdout:                &stdout,
		Stderr:                &stderr,
		InsecureIgnoreHostKey: true,
	}
	err = runner.Run(afterStart, req.RemoteArgv, sshcmd.RunnerOpts{ProfileID: req.ProfileID})
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if err != nil {
		resp.RunnerErr = err.Error()
	}

	cancel()
	wg.Wait()
	if startErr != nil && startErr != context.Canceled {
		if !strings.Contains(startErr.Error(), "canceled") && !strings.Contains(startErr.Error(), "cancelled") {
			resp.ServeErr = startErr.Error()
		}
	}
}

func runServeWiringDial(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	kp := mustKeyPair(t)
	pub := publicKeyOpenSSH(kp.Public)

	_, _, base := startTunnelServer(t, req.ManagerToken, nil)
	c := clientWithToken(base, req.Token)

	// BuildSSHTunnelDial is the testable surface for agentcli --serve wiring.
	dial, info, err := agentcli.BuildSSHTunnelDial(c, pub)
	if err != nil {
		resp.WiringErr = err.Error()
		resp.DialIsNil = dial == nil
		return
	}
	if info != nil {
		resp.WiringSessID = info.SessionID
		resp.SessionID = info.SessionID
	}
	resp.DialIsNil = dial == nil
	if dial == nil {
		return
	}
	// Smoke: Dial should open a live conn (Adhoc or backend).
	conn, err := dial()
	if err != nil {
		resp.TunnelDialErr = err.Error()
		return
	}
	_ = conn.Close()
}
```
