# remote-agent ssh — L2 doctests (P1)

Plan phase **P1** (classic TDD / greenfield): injectable library API for
`remote-agent ssh` argv modes, optional `user@host` strip, session discovery
gate, help, and `--serve` exclusive rules — **without** real network or OpenSSH.

Package under test (intended, may not exist yet → **RED**):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd
```

Out of scope for this tree: real duplex tunnel, remote ad-hoc SSH server, local
TCP relay bytes, OpenSSH subprocess, Unison e2e, top-level agentcli wiring.

# DSN (Domain Specific Notion)

**remote-agent ssh** is an OpenSSH-shaped CLI surface used later as Unison
`-sshcmd`. Operators invoke `remote-agent ssh` with optional `--serve`, optional
`user@host`, and an optional remote command.

**Participants**

- **sshcmd.Parse** — classifies argv into help | serve | login | command; strips
  optional destination when it matches the strict `user@host` matcher.
- **sshcmd.Run** — executes a mode with injected deps (no process env/cwd).
- **SessionStore** — loads active tunnel session metadata by profile id.
- **ServeStarter** — starts the local serve side (`--serve` only).
- **SSHRunner** — runs interactive login or remote command through a live session.
- **stdout/stderr writers** — injected; help text and errors observed without
  reassigning global `os.Stdout` / `os.Stderr`.

**Behaviors**

- Help (`-h` / `--help`) prints Usage mentioning `--serve` and optional
  `user@host` / command form; nil error.
- `--serve` alone → `ServeStarter.Start`; exclusive with remote command args.
- Client login/command → `SessionStore.Load`; require non-nil `Alive` session or
  error `no active SSH tunnel; run 'remote-agent ssh --serve' first`.
- Alive client → `SSHRunner.Run(sess, remoteArgv, …)` with dest already stripped
  from remote argv.
- Destination matcher: `^[^@/\s]+@[^@/\s]+$` only (path-like `./a@b` is a command).

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/sshcmd/tests/          [Request{Args, Session, …}]
│                                   Run: Parse + Run with mock store/serve/runner
├── help/                           # mode help
│   ├── long-help/                  # --help → Usage on stdout, nil err
│   └── short-help/                 # -h → same
├── serve/                          # mode serve
│   ├── alone/                      # --serve → ServeStarter once; no Runner
│   └── exclusive-with-command/     # --serve + ls → exclusive error
├── login/                          # mode login (no remote command)
│   ├── no-args/
│   │   ├── no-session/             # empty store → tunnel error
│   │   └── alive-session/          # Runner called with empty remote argv
│   └── dest-only/
│       └── alive-session/          # agent@ra stripped; login; empty remote
└── command/                        # mode command (remote argv non-empty)
    ├── bare/
    │   ├── no-session/             # ls, empty store → tunnel error
    │   └── alive-session/          # Runner remote ["ls"]
    ├── dest-stripped/
    │   └── alive-session/          # agent@ra ls -la → ["ls","-la"]
    ├── multi-arg/
    │   └── alive-session/          # uname -a → ["uname","-a"]
    └── pathlike-not-dest/
        └── alive-session/          # ./a@b is command token, not dest
```

**Significance order:** operation mode (help | serve | login | command) →
argv shape (flags / dest / remote tokens) → session gate (missing | alive).

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/long-help` | `--help` → Usage on stdout; nil error; mentions serve + user@host |
| 2 | `help/short-help` | `-h` → same Usage contract |
| 3 | `serve/alone` | `--serve` → mode Serve; ServeStarter once; Runner not called |
| 4 | `serve/exclusive-with-command` | `--serve` + `ls` → exclusive error; no Start/Run |
| 5 | `login/no-args/no-session` | no args, empty store → no active tunnel error |
| 6 | `login/no-args/alive-session` | no args, Alive session → Runner with empty remote |
| 7 | `login/dest-only/alive-session` | `agent@ra` stripped; login; Runner empty remote |
| 8 | `command/bare/no-session` | `ls`, empty store → no active tunnel error |
| 9 | `command/bare/alive-session` | `ls`, Alive → Runner `["ls"]` |
| 10 | `command/dest-stripped/alive-session` | `agent@ra ls -la` → Runner `["ls","-la"]` |
| 11 | `command/multi-arg/alive-session` | `uname -a` → Runner `["uname","-a"]` |
| 12 | `command/pathlike-not-dest/alive-session` | `./a@b` not dest; Runner `["./a@b"]` |

## How to Run

From module root (`external/ai-critic-master-2026-07-31` or workspace with that module):

```sh
doctest vet ./cmd/agentcli/sshcmd/tests
doctest test ./cmd/agentcli/sshcmd/tests
doctest test ./cmd/agentcli/sshcmd/tests/help/long-help
```

```go
import (
	"bytes"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/doctest/session"
)

// Request configures argv and injectable session/mock outcomes.
// Parallel-safe: all deps via Request; no Setenv/Chdir.
type Request struct {
	// Args are the argv after the `ssh` subcommand (e.g. ["--help"], ["ls"]).
	Args []string

	// ProfileID passed to SessionStore.Load / ServeStarter / Runner.
	// Empty means harness default "default".
	ProfileID string

	// Session is returned by the mock SessionStore.Load.
	// nil means no session on disk (missing).
	Session *sshcmd.Session

	// StoreErr, when non-nil, is returned from SessionStore.Load.
	StoreErr error

	// ServeErr / RunnerErr are returned from the respective mocks when called.
	ServeErr  error
	RunnerErr error
}

// Response captures parse classification, Run I/O, and mock call records.
type Response struct {
	Stdout string
	Stderr string

	// Parse fields (from sshcmd.Parse).
	Mode       sshcmd.Mode
	Dest       string
	RemoteArgv []string
	ParseErr   string

	// RunErr is the error string from sshcmd.Run (empty if nil).
	RunErr string

	// Mock observations.
	ServeStartCalls  int
	ServeProfileID   string
	RunnerCalls      int
	RunnerRemoteArgv []string
	RunnerProfileID  string
	RunnerSession    *sshcmd.Session
	StoreLoadCalls   int
	StoreProfileID   string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	_ = d // paths unused in pure L2 mocks; reserved for parallel-safe harness

	if req.Args == nil {
		req.Args = []string{}
	}
	profileID := req.ProfileID
	if profileID == "" {
		profileID = "default"
	}

	resp := &Response{}
	store := &mockSessionStore{
		sess: req.Session,
		err:  req.StoreErr,
	}
	serve := &mockServeStarter{err: req.ServeErr}
	runner := &mockSSHRunner{err: req.RunnerErr}

	var stdout, stderr bytes.Buffer

	parsed, parseErr := sshcmd.Parse(req.Args)
	if parseErr != nil {
		resp.ParseErr = parseErr.Error()
	}
	if parsed != nil {
		resp.Mode = parsed.Mode
		resp.Dest = parsed.Dest
		resp.RemoteArgv = append([]string(nil), parsed.RemoteArgv...)
	}

	runErr := sshcmd.Run(sshcmd.Options{
		Args:      req.Args,
		ProfileID: profileID,
		Store:     store,
		Serve:     serve,
		Runner:    runner,
		Stdout:    &stdout,
		Stderr:    &stderr,
	})

	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if runErr != nil {
		resp.RunErr = runErr.Error()
	}

	resp.ServeStartCalls = serve.calls
	resp.ServeProfileID = serve.profileID
	resp.RunnerCalls = runner.calls
	resp.RunnerRemoteArgv = append([]string(nil), runner.remoteArgv...)
	resp.RunnerProfileID = runner.profileID
	resp.RunnerSession = runner.sess
	resp.StoreLoadCalls = store.calls
	resp.StoreProfileID = store.profileID

	return resp, nil
}

// mockSessionStore implements sshcmd.SessionStore for tests.
type mockSessionStore struct {
	sess      *sshcmd.Session
	err       error
	calls     int
	profileID string
}

func (m *mockSessionStore) Load(profileID string) (*sshcmd.Session, error) {
	m.calls++
	m.profileID = profileID
	if m.err != nil {
		return nil, m.err
	}
	return m.sess, nil
}

// mockServeStarter implements sshcmd.ServeStarter for tests.
type mockServeStarter struct {
	err       error
	calls     int
	profileID string
}

func (m *mockServeStarter) Start(opts sshcmd.ServeOpts) error {
	m.calls++
	m.profileID = opts.ProfileID
	return m.err
}

// mockSSHRunner implements sshcmd.SSHRunner for tests.
type mockSSHRunner struct {
	err        error
	calls      int
	sess       *sshcmd.Session
	remoteArgv []string
	profileID  string
}

func (m *mockSSHRunner) Run(sess *sshcmd.Session, remoteArgv []string, opts sshcmd.RunnerOpts) error {
	m.calls++
	m.sess = sess
	m.remoteArgv = append([]string(nil), remoteArgv...)
	m.profileID = opts.ProfileID
	return m.err
}

// aliveSession returns a minimal Alive session for client happy paths.
func aliveSession() *sshcmd.Session {
	return &sshcmd.Session{
		LocalPort: 18022,
		User:      "agent",
		ConfigDir: "/tmp/mock-ssh-config",
		ServePID:  0,
		ProfileID: "default",
		Alive:     true,
	}
}

// errText prefers RunErr, then ParseErr (exclusive parse failures).
func errText(resp *Response) string {
	if resp == nil {
		return ""
	}
	if resp.RunErr != "" {
		return resp.RunErr
	}
	return resp.ParseErr
}
```
