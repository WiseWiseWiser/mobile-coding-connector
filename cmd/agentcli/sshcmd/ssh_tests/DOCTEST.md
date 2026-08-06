# remote-agent ssh — L2 doctests (P3: ad-hoc SSH + CryptoSSHRunner + agentcli wire)

Plan phase **P3** (classic TDD): in-process ad-hoc SSH server (`x/crypto/ssh`),
`CryptoSSHRunner` client through session LocalPort, compose with LocalRelay /
ServeService Dial, and top-level `agentcli` `ssh` subcommand wiring.

Sibling of sealed P1 `cmd/agentcli/sshcmd/tests` (12 leaves) and P2
`cmd/agentcli/sshcmd/session_tests` (9 leaves) — **do not edit** those trees.

Package under test (extends P1/P2 surface; new symbols **RED** until implementer):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd
github.com/xhd2015/ai-critic/cmd/agentcli   # ssh subcommand dispatch only
```

Out of scope: live remote ai-critic WebSocket tunnel; Cloudflare; Unison e2e;
system `ssh` binary (L3 optional, prefer avoid).

Network model under test:

```text
CryptoSSHRunner (x/crypto client)
  -> 127.0.0.1:L (LocalRelay / session LocalPort)
  -> DialFunc -> 127.0.0.1:R (AdhocServer)
```

# DSN (Domain Specific Notion)

**remote-agent ssh P3** carries real SSH session bytes on loopback: an ad-hoc
server, a crypto client runner, and CLI dispatch that wires store + serve + runner.

**Participants**

- **AdhocServer** — listen `127.0.0.1:0`; host key generated; public-key auth;
  session channel: empty command → login shell; else remote command exec.
- **ClientKeyPair** — ed25519 key pair for tests (`GenerateClientKeyPair`).
- **CryptoSSHRunner** — `SSHRunner` via `golang.org/x/crypto/ssh` client; dials
  `127.0.0.1:sess.LocalPort`; empty argv → PTY+Shell; else `session.Run`.
- **ServeService + Dial** — Dial opens TCP to AdhocServer (in-process compose).
- **LocalRelay** — P2 splice (unchanged); used under ServeService.
- **agentcli.Run** — top-level `ssh` case → sshcmd with file store + real defaults.

**Behaviors**

- Adhoc accepts authorized pubkey; rejects wrong key.
- Authorized client runs `echo hello` → stdout contains `hello`.
- Login shell channel opens (shell accepts a simple command / exit).
- Through relay: runner command via serve LocalPort succeeds.
- Serve + Adhoc lifecycle: start, command, cancel → session cleared / port closed.
- `agentcli` recognizes `ssh` (not `unknown command`); help returns nil error.
- `agentcli ssh ls` without Alive session → tunnel error string.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/sshcmd/ssh_tests/          [Request{Scenario, …}]
│                                       Run dispatches by Scenario (L2 in-process)
├── adhoc-server/                       # AdhocServer alone (direct dial)
│   ├── authorized/
│   │   ├── remote-command/             # right key + echo hello → stdout
│   │   └── login-shell/                # right key + shell channel works
│   └── unauthorized/
│       └── wrong-key/                  # wrong key → auth failure
├── crypto-runner/                      # CryptoSSHRunner.Run → Adhoc (direct port)
│   └── remote-command/                 # Run(sess, echo) → stdout hello
├── through-relay/                      # ServeService Dial=Adhoc + runner via L
│   ├── remote-command/                 # full stack echo ok / hello
│   └── serve-cancel/                   # start → command → cancel lifecycle
└── agentcli/                           # top-level remote-agent ssh wire
    ├── help-recognized/                # ssh --help not unknown command
    └── command-no-session/             # ssh ls → no active tunnel error
```

**Significance order:** stack surface (adhoc | crypto-runner | through-relay |
agentcli) → auth / lifecycle / CLI outcome within surface.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `adhoc-server/authorized/remote-command` | Authorized key runs remote `echo hello` |
| 2 | `adhoc-server/authorized/login-shell` | Authorized key opens login shell; shell runs |
| 3 | `adhoc-server/unauthorized/wrong-key` | Wrong key rejected (auth error) |
| 4 | `crypto-runner/remote-command` | CryptoSSHRunner.Run command against Adhoc |
| 5 | `through-relay/remote-command` | Serve Dial→Adhoc; runner via LocalPort |
| 6 | `through-relay/serve-cancel` | Full lifecycle cancel after successful command |
| 7 | `agentcli/help-recognized` | `agentcli.Run(…, ["ssh","--help"])` not unknown |
| 8 | `agentcli/command-no-session` | `ssh ls` → no active tunnel error string |

## Exported APIs (implementer contract)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd` (extend P1/P2):

| Symbol | Role |
|--------|------|
| `AdhocServer` | In-process SSH server; fields: `User` (default `"agent"`) |
| `(*AdhocServer).Start` | `() error` — listen `127.0.0.1:0`, accept SSH |
| `(*AdhocServer).Addr` | `() string` — `"127.0.0.1:<port>"` after Start |
| `(*AdhocServer).LocalPort` | `() int` — bound port after Start |
| `(*AdhocServer).Close` | `() error` — stop listener |
| `(*AdhocServer).HostKey` | `() ssh.PublicKey` — server host public key |
| `(*AdhocServer).SetAuthorizedKeys` | `(keys []ssh.PublicKey)` — replace authorized set |
| `GenerateClientKeyPair` | `() (*ClientKeyPair, error)` — ed25519 client keys |
| `ClientKeyPair` | `struct { Signer ssh.Signer; Public ssh.PublicKey }` |
| `CryptoSSHRunner` | Implements `SSHRunner`; fields: `Signer`, `Stdout`, `Stderr`, `Stdin`, `InsecureIgnoreHostKey` (bool), optional `HostKeyCallback` |
| `(*CryptoSSHRunner).Run` | `(sess *Session, remoteArgv []string, opts RunnerOpts) error` — dial `127.0.0.1:sess.LocalPort`; empty argv → RequestPty+Shell; else join argv and `session.Run` |
| `DialTCP` | `(addr string) DialFunc` — **optional** helper; tests inline `net.DialTimeout` equivalently |

Compose pattern (tests / serve wiring):

```go
adhoc := &sshcmd.AdhocServer{User: "agent", ForcePipeShell: true}
adhoc.SetAuthorizedKeys([]ssh.PublicKey{kp.Public})
_ = adhoc.Start()
defer adhoc.Close()

svc := &sshcmd.ServeService{
  Store: store, ProfileID: id, User: "agent",
  ConfigDir: configDir, ServePID: os.Getpid(),
  Dial: sshcmd.DialTCP(adhoc.Addr()), // or equivalent net.Dial wrapper
}
// Start(ctx) in goroutine; runner dials sess.LocalPort (relay)
```

Agentcli wire (package `agentcli`):

| Symbol / change | Role |
|-----------------|------|
| `Run` switch `case "ssh"` | Dispatch to ssh wiring (not `unknown command: ssh`) |
| Default store | `FileSessionStore` under profile home / config root (parallel-safe path; may use `testhooks.UserHomeDir`) |
| Default runner | `CryptoSSHRunner` (keys from session ConfigDir when present; missing session still fails gate first) |
| Default serve | `ServeService` as `ServeStarter` for `--serve` (Dial may be stub/error until remote tunnel exists; P3 client path is the gate + help) |

Help content remains `sshcmd.Usage` (P1). Error string remains
`sshcmd.ErrNoActiveTunnel` / `no active SSH tunnel; run 'remote-agent ssh --serve' first`.

## How to Run

From module root (`external/ai-critic-master-2026-07-31`):

```sh
doctest vet ./cmd/agentcli/sshcmd/ssh_tests
doctest test ./cmd/agentcli/sshcmd/ssh_tests
# P1 + P2 must stay green:
doctest test ./cmd/agentcli/sshcmd/tests
doctest test ./cmd/agentcli/sshcmd/session_tests
```

```go
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/doctest/session"
	"golang.org/x/crypto/ssh"
)

// Scenario selects which P3 surface Run exercises (MECE dispatch key).
type Scenario string

const (
	ScenarioAdhocAuthAcceptCommand Scenario = "adhoc-auth-accept-command"
	ScenarioAdhocLoginShell        Scenario = "adhoc-login-shell"
	ScenarioAdhocAuthReject        Scenario = "adhoc-auth-reject"
	ScenarioCryptoRunnerCommand    Scenario = "crypto-runner-command"
	ScenarioRelayCommand           Scenario = "through-relay-command"
	ScenarioRelayServeCancel       Scenario = "through-relay-serve-cancel"
	ScenarioAgentcliHelp           Scenario = "agentcli-help"
	ScenarioAgentcliNoSession      Scenario = "agentcli-no-session"
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

	// RemoteCommand is joined for session.Run / CryptoSSHRunner remote argv.
	// Default "echo hello" for command leaves.
	RemoteCommand string
	// RemoteArgv when non-empty overrides splitting RemoteCommand (runner leaves).
	RemoteArgv []string
	// EchoNeedle is substring expected in remote stdout (default "hello").
	EchoNeedle string
}

// Response captures SSH, runner, compose, and agentcli outcomes.
type Response struct {
	// Adhoc / ports
	AdhocPort     int
	AdhocStartErr string
	AdhocCloseErr string

	// Auth / command via direct x/crypto client (adhoc leaves)
	AuthErr    string
	Stdout     string
	Stderr     string
	CommandErr string
	ShellErr   string
	ShellOK    bool

	// CryptoSSHRunner
	RunnerErr string

	// Through-relay / serve
	ServeErr          string
	SessionAfterStart *sshcmd.Session
	SessionAfterStop  *sshcmd.Session
	RelayLocalPort    int
	PortClosedAfterStop bool

	// agentcli
	AgentcliErr string
	// UnknownCommand is true when error text looks like top-level unknown cmd.
	UnknownCommand bool
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	if req.Root == "" {
		req.Root = filepath.Join(d.DOCTEST_CASE, "store-root")
	}
	if req.ConfigDir == "" {
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "session-config")
	}
	if req.RemoteCommand == "" {
		req.RemoteCommand = "echo hello"
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "hello"
	}
	if len(req.RemoteArgv) == 0 && req.RemoteCommand != "" {
		req.RemoteArgv = strings.Fields(req.RemoteCommand)
	}

	resp := &Response{}
	switch req.Scenario {
	case ScenarioAdhocAuthAcceptCommand:
		runAdhocAuthAcceptCommand(t, d, req, resp)
	case ScenarioAdhocLoginShell:
		runAdhocLoginShell(t, d, req, resp)
	case ScenarioAdhocAuthReject:
		runAdhocAuthReject(t, d, req, resp)
	case ScenarioCryptoRunnerCommand:
		runCryptoRunnerCommand(t, d, req, resp)
	case ScenarioRelayCommand:
		runThroughRelayCommand(t, d, req, resp)
	case ScenarioRelayServeCancel:
		runThroughRelayServeCancel(t, d, req, resp)
	case ScenarioAgentcliHelp:
		runAgentcliHelp(t, d, req, resp)
	case ScenarioAgentcliNoSession:
		runAgentcliNoSession(t, d, req, resp)
	default:
		return nil, fmt.Errorf("unknown scenario %q", req.Scenario)
	}
	return resp, nil
}

// --- key helpers ---

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

func startAdhoc(t *testing.T, user string, authorized ...ssh.PublicKey) *sshcmd.AdhocServer {
	t.Helper()
	if user == "" {
		user = "agent"
	}
	// ForcePipeShell: doctests need deterministic pipe shell (empty PS1).
	s := &sshcmd.AdhocServer{User: user, ForcePipeShell: true}
	if len(authorized) > 0 {
		s.SetAuthorizedKeys(authorized)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("AdhocServer.Start: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if s.LocalPort() <= 0 {
		t.Fatalf("AdhocServer.LocalPort: got %d", s.LocalPort())
	}
	return s
}

func dialSSH(addr, user string, signer ssh.Signer, hostKey ssh.PublicKey, insecure bool) (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	if !insecure && hostKey != nil {
		cfg.HostKeyCallback = ssh.FixedHostKey(hostKey)
	}
	return ssh.Dial("tcp", addr, cfg)
}

// dialFuncTo builds a DialFunc that TCP-dials addr (ServeService remote side).
// Implementer may also export sshcmd.DialTCP(addr); tests use this inline form
// so DialTCP remains optional.
func dialFuncTo(addr string) sshcmd.DialFunc {
	return func() (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 5*time.Second)
	}
}

// --- adhoc leaves ---

func runAdhocAuthAcceptCommand(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	kp := mustKeyPair(t)
	srv := startAdhoc(t, "agent", kp.Public)
	resp.AdhocPort = srv.LocalPort()

	client, err := dialSSH(srv.Addr(), "agent", kp.Signer, srv.HostKey(), true)
	if err != nil {
		resp.AuthErr = err.Error()
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		resp.CommandErr = err.Error()
		return
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(req.RemoteCommand)
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if err != nil {
		resp.CommandErr = err.Error()
	}
}

func runAdhocLoginShell(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	_ = req
	kp := mustKeyPair(t)
	srv := startAdhoc(t, "agent", kp.Public)
	resp.AdhocPort = srv.LocalPort()

	client, err := dialSSH(srv.Addr(), "agent", kp.Signer, srv.HostKey(), true)
	if err != nil {
		resp.AuthErr = err.Error()
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		resp.ShellErr = err.Error()
		return
	}
	defer session.Close()

	// Request a PTY and start login shell (empty command path on server).
	modes := ssh.TerminalModes{ssh.ECHO: 0}
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		resp.ShellErr = "RequestPty: " + err.Error()
		return
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		resp.ShellErr = err.Error()
		return
	}
	// crypto/ssh copies stdout and stderr in concurrent goroutines. Sharing a
	// bare bytes.Buffer races (ReadFrom grow) and intermittently loses shell-ok
	// under parallel suite load. syncWriter serializes writes.
	var stdout safeBuffer
	session.Stdout = &stdout
	session.Stderr = &stdout
	if err := session.Shell(); err != nil {
		resp.ShellErr = "Shell: " + err.Error()
		return
	}
	// Drive a trivial command then exit so the leaf is deterministic.
	_, _ = io.WriteString(stdin, "echo shell-ok\nexit\n")
	_ = stdin.Close()
	waitErr := session.Wait()
	resp.Stdout = stdout.String()
	if waitErr != nil && !strings.Contains(resp.Stdout, "shell-ok") {
		resp.ShellErr = waitErr.Error()
		return
	}
	resp.ShellOK = strings.Contains(resp.Stdout, "shell-ok")
	if !resp.ShellOK {
		// Some shells print without the exact token if non-interactive; accept
		// successful Shell() + Wait with empty error as soft OK via ShellOK
		// only when needle found. ASSERT requires ShellOK.
		resp.ShellErr = "shell stdout missing shell-ok"
	}
}

// safeBuffer serializes Write for concurrent ssh.Session stdout+stderr copy
// goroutines. Intentionally does not implement ReaderFrom: io.Copy then uses
// Write per chunk (safe). A locked ReadFrom would deadlock when both streams
// call it (each holds the mutex for the full read-until-EOF).
type safeBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (w *safeBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *safeBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func runAdhocAuthReject(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	_ = req
	good := mustKeyPair(t)
	bad := mustKeyPair(t)
	srv := startAdhoc(t, "agent", good.Public)
	resp.AdhocPort = srv.LocalPort()

	client, err := dialSSH(srv.Addr(), "agent", bad.Signer, srv.HostKey(), true)
	if err != nil {
		resp.AuthErr = err.Error()
		return
	}
	// Unexpected success: close and leave AuthErr empty (ASSERT expects error).
	_ = client.Close()
	resp.AuthErr = ""
}

// --- crypto runner ---

func runCryptoRunnerCommand(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	kp := mustKeyPair(t)
	srv := startAdhoc(t, "agent", kp.Public)
	resp.AdhocPort = srv.LocalPort()

	var stdout, stderr bytes.Buffer
	runner := &sshcmd.CryptoSSHRunner{
		Signer:                kp.Signer,
		Stdout:                &stdout,
		Stderr:                &stderr,
		InsecureIgnoreHostKey: true,
	}
	sess := &sshcmd.Session{
		LocalPort: srv.LocalPort(),
		User:      "agent",
		ConfigDir: req.ConfigDir,
		ProfileID: req.ProfileID,
		Alive:     true,
	}
	err := runner.Run(sess, req.RemoteArgv, sshcmd.RunnerOpts{ProfileID: req.ProfileID})
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if err != nil {
		resp.RunnerErr = err.Error()
	}
}

// --- through relay ---

func runThroughRelayCommand(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	if err := os.MkdirAll(req.ConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	kp := mustKeyPair(t)
	srv := startAdhoc(t, "agent", kp.Public)
	resp.AdhocPort = srv.LocalPort()

	store := &sshcmd.FileSessionStore{Root: req.Root}
	svc := &sshcmd.ServeService{
		Store:     store,
		ProfileID: req.ProfileID,
		Dial:      dialFuncTo(srv.Addr()),
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
	err := runner.Run(afterStart, req.RemoteArgv, sshcmd.RunnerOpts{ProfileID: req.ProfileID})
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

func runThroughRelayServeCancel(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	// Same stack as remote-command, then assert teardown fields.
	runThroughRelayCommand(t, d, req, resp)
	if resp.SessionAfterStart == nil {
		return
	}
	store := &sshcmd.FileSessionStore{Root: req.Root}
	afterStop, err := store.Load(req.ProfileID)
	if err != nil {
		resp.ServeErr = "load after stop: " + err.Error()
	}
	resp.SessionAfterStop = afterStop

	port := resp.RelayLocalPort
	conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if dialErr != nil {
		resp.PortClosedAfterStop = true
	} else {
		_ = conn.Close()
		resp.PortClosedAfterStop = false
	}
}

// --- agentcli ---

func runAgentcliHelp(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	_ = req
	// In-process CLI dispatch. Help may write to process stdout (not captured);
	// assert recognition via error contract only (parallel-safe).
	err := agentcli.Run(agentcli.RemoteProfile(), []string{"ssh", "--help"})
	if err != nil {
		resp.AgentcliErr = err.Error()
		resp.UnknownCommand = isUnknownSSHCommand(err.Error())
	}
}

func runAgentcliNoSession(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	_ = req
	err := agentcli.Run(agentcli.RemoteProfile(), []string{"ssh", "ls"})
	if err != nil {
		resp.AgentcliErr = err.Error()
		resp.UnknownCommand = isUnknownSSHCommand(err.Error())
	}
}

func isUnknownSSHCommand(msg string) bool {
	return strings.Contains(msg, "unknown command") && strings.Contains(msg, "ssh")
}
```
