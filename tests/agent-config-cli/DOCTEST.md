# Agent Config CLI Flags Doctests

End-to-end tests for the shared `config` subcommand on `remote-agent` and
`local-agent`: bare help, `--show` / `--json`, mutual exclusion with `--web`,
and profile-specific config paths under an isolated `HOME`.

Classic TDD: production still opens the web UI on bare `config` and lacks
`--web` / `--show` / `--json`. Leaves express the **target** contract and are
expected **RED** until implementation lands.

# DSN (Domain Specific Notion)

The harness runs the real agent CLI binaries against a temp user home so config
files never touch the developer machine. No server is required for these leaves.

**Participants**

- **remote-agent / local-agent subprocess** — built from `./cmd/remote-agent` and
  `./cmd/local-agent`; both share `cmd/agentcli` `runConfig` with a profile
  (`active`) that selects CLI name and config filename.
- **Isolated user HOME** — temp directory; agent config lives under
  `~/.ai-critic/remote-agent-config.json` or `local-agent-config.json`.
- **Config file seed** — optional pretty JSON written before the CLI runs for
  `--show` content assertions; missing file exercises empty-ish dump.
- **Harness timer** — CLI runs under a short wall-clock timeout so bare `config`
  (current UI path blocks forever) fails fast instead of hanging the suite.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/agent-config-cli-doctest-<id>/` for shared binaries (file lock).

**Behaviors (target)**

- Bare `config` prints help to stdout, exits 0, does **not** start the config UI
  (no `Config UI running`).
- `config --help` / `-h` print the same help family (mentions `--web`, `--show`).
- `--show` loads saved config via `loadConfig()` and prints **pretty JSON** on
  stdout; missing file → empty-ish config with empty domains; tokens unredacted.
- `--show --json` is a no-op success path identical to `--show`.
- `--json` alone errors (requires `--show`).
- `--show` and `--web` are mutually exclusive (non-zero error).
- Unknown flags → non-zero error pointing at help.
- local-agent help uses `local-agent` branding; `--show` reads only
  `local-agent-config.json`.

## Version

0.0.2

## Decision Tree

```
[agent config CLI]
 |
 +-- remote-agent/                         (GROUP) Profile = remote-agent
 |    |
 |    +-- help/                            (GROUP) help / bare mode
 |    |    +-- bare/                       (LEAF)  config → help, exit 0, no UI banner
 |    |    +-- long-help/                  (LEAF)  config --help → help family
 |    |    +-- short-help/                 (LEAF)  config -h → help family
 |    |
 |    +-- show/                            (GROUP) --show dump
 |    |    +-- missing-file/               (LEAF)  no config file → empty-ish pretty JSON
 |    |    +-- with-domains/               (LEAF)  seeded domains+default match stdout
 |    |    +-- show-json-noop/             (LEAF)  --show --json ≡ --show
 |    |
 |    +-- rejected/                        (GROUP) invalid flags / combos
 |         +-- json-alone/                 (LEAF)  --json without --show → error
 |         +-- show-and-web/               (LEAF)  --show --web mutual exclusion
 |         +-- unknown-flag/               (LEAF)  unknown flag → non-zero
 |
 +-- local-agent/                          (GROUP) Profile = local-agent
      |
      +-- help/
      |    +-- bare/                       (LEAF)  local-agent config → help branding
      |
      +-- show/
      |    +-- missing-file/               (LEAF)  empty-ish from local path
      |    +-- with-domains/               (LEAF)  local-agent-config.json content
      |
      +-- rejected/
           +-- json-alone/                 (LEAF)  --json alone (local binary)
           +-- show-and-web/               (LEAF)  mutual exclusion (local binary)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `remote-agent/help/bare` | Bare `config` → help stdout, exit 0, no UI banner |
| 2 | `remote-agent/help/long-help` | `config --help` documents `--web` / `--show` |
| 3 | `remote-agent/help/short-help` | `config -h` same help family |
| 4 | `remote-agent/show/missing-file` | No file → pretty empty-ish JSON, exit 0 |
| 5 | `remote-agent/show/with-domains` | Seeded remote config pretty-printed on stdout |
| 6 | `remote-agent/show/show-json-noop` | `--show --json` matches `--show` success |
| 7 | `remote-agent/rejected/json-alone` | `--json` alone → non-zero, mentions `--show` |
| 8 | `remote-agent/rejected/show-and-web` | `--show --web` mutual exclusion |
| 9 | `remote-agent/rejected/unknown-flag` | Unknown flag → non-zero error |
| 10 | `local-agent/help/bare` | Bare local `config` → `local-agent` help, no UI |
| 11 | `local-agent/show/missing-file` | Missing local config → empty-ish JSON |
| 12 | `local-agent/show/with-domains` | Reads `local-agent-config.json` only |
| 13 | `local-agent/rejected/json-alone` | Local `--json` alone errors |
| 14 | `local-agent/rejected/show-and-web` | Local `--show --web` errors |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| CLI profile (remote vs local) | remote-agent/*, local-agent/* |
| Help form (bare / --help / -h) | help/bare, help/long-help, help/short-help |
| Config file presence | show/missing-file, show/with-domains |
| `--json` with `--show` | show/show-json-noop |
| `--json` alone | rejected/json-alone |
| `--show` + `--web` | rejected/show-and-web |
| Unknown flag | rejected/unknown-flag |
| No UI on bare | help/bare (remote + local) |
| Config path isolation | local-agent/show/* |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/agent-config-cli
doctest test ./tests/agent-config-cli/...
```

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

// AgentConfigFile is the persisted agent config JSON shape.
type AgentConfigFile struct {
	Default         string           `json:"default,omitempty"`
	Domains         []DomainEntry    `json:"domains"`
	ProjectBindings []ProjectBinding `json:"project_bindings,omitempty"`
}

// DomainEntry is one saved server+token pair.
type DomainEntry struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

// ProjectBinding is an optional project_bindings row.
type ProjectBinding struct {
	Server    string `json:"server"`
	RemoteDir string `json:"remote_dir"`
	LocalPath string `json:"local_path"`
}

// Profile selects which CLI binary and config filename to use.
type Profile string

const (
	ProfileRemote Profile = "remote-agent"
	ProfileLocal  Profile = "local-agent"
)

type Request struct {
	// Profile is remote-agent or local-agent (required before Run).
	Profile Profile

	// Args are argv after the binary name (must start with "config" for these leaves).
	Args []string

	// SeedConfig writes the profile's config file under isolated HOME when non-nil.
	SeedConfig *AgentConfigFile

	// AlsoSeedRemoteConfig (local only) writes a remote-agent-config.json sentinel
	// so isolation leaves can prove the wrong file is not read.
	AlsoSeedRemoteConfig *AgentConfigFile

	// Timeout overrides the default CLI kill timer (0 = default).
	Timeout time.Duration
}

type Response struct {
	ExitCode  int
	Stdout    string
	Stderr    string
	Combined  string
	TimedOut  bool
	AgentHome string
	// ConfigPath is the profile config path under AgentHome.
	ConfigPath string
	// RemoteConfigPath is always HOME/.ai-critic/remote-agent-config.json when set up.
	RemoteConfigPath string
	// LocalConfigPath is always HOME/.ai-critic/local-agent-config.json when set up.
	LocalConfigPath string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Profile != ProfileRemote && req.Profile != ProfileLocal {
		return nil, fmt.Errorf("Request.Profile must be remote-agent or local-agent, got %q", req.Profile)
	}
	if len(req.Args) == 0 {
		req.Args = []string{"config"}
	}

	// DOCTEST_ROOT is tests/agent-config-cli; module root is two levels up.
	// Do not walk from cwd: doctest runs under mapping-gen which has its own go.mod.
	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)
	remoteBin, localBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	agentBin := remoteBin
	if req.Profile == ProfileLocal {
		agentBin = localBin
	}

	agentHome, err := os.MkdirTemp("", "agent-config-cli-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	aiCriticDir := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticDir, 0755); err != nil {
		return nil, err
	}
	resp.RemoteConfigPath = filepath.Join(aiCriticDir, "remote-agent-config.json")
	resp.LocalConfigPath = filepath.Join(aiCriticDir, "local-agent-config.json")
	if req.Profile == ProfileLocal {
		resp.ConfigPath = resp.LocalConfigPath
	} else {
		resp.ConfigPath = resp.RemoteConfigPath
	}

	if req.SeedConfig != nil {
		if err := writeAgentConfigFile(resp.ConfigPath, req.SeedConfig); err != nil {
			return nil, err
		}
	}
	if req.AlsoSeedRemoteConfig != nil {
		if err := writeAgentConfigFile(resp.RemoteConfigPath, req.AlsoSeedRemoteConfig); err != nil {
			return nil, err
		}
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 4 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t.Logf("%s argv: %v (HOME=%s)", req.Profile, req.Args, agentHome)

	cmd := exec.CommandContext(ctx, agentBin, req.Args...)
	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)
	// Avoid GUI/browser side effects if an old bare path is hit.
	agentEnv = append(agentEnv, "BROWSER=true", "DISPLAY=")
	cmd.Env = agentEnv

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	if ctx.Err() == context.DeadlineExceeded {
		resp.TimedOut = true
		resp.ExitCode = -1
		return resp, nil
	}
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			return nil, runErr
		}
	}
	return resp, nil
}

func writeAgentConfigFile(path string, cfg *AgentConfigFile) error {
	if cfg.Domains == nil {
		cfg.Domains = []DomainEntry{}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func stripEnvPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func sessionCacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "agent-config-cli-doctest-"+sessionID)
}

func withFileLock(t *testing.T, lockPath string, fn func() error) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock %s: %w", lockPath, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func buildSessionBinariesOnce(t *testing.T, moduleRoot, cacheDir string) (remoteBin, localBin string) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	remoteBin = filepath.Join(cacheDir, "remote-agent")
	localBin = filepath.Join(cacheDir, "local-agent")
	ready := filepath.Join(cacheDir, "binaries.ready")
	lock := filepath.Join(cacheDir, "build.lock")
	err := withFileLock(t, lock, func() error {
		if fileExists(ready) && fileExists(remoteBin) && fileExists(localBin) {
			return nil
		}
		for _, spec := range []struct {
			out string
			pkg string
		}{
			{remoteBin, "./cmd/remote-agent"},
			{localBin, "./cmd/local-agent"},
		} {
			cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
			cmd.Dir = moduleRoot
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
			}
		}
		return os.WriteFile(ready, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("session binaries cache: %s", cacheDir)
	return remoteBin, localBin
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```
