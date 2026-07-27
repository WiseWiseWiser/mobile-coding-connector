# Agent Config CLI Flags Doctests

Scenario tests for the shared `config` subcommand on `remote-agent` and
`local-agent`: bare help, `--show` / `--json`, mutual exclusion with `--web`,
and profile-specific config paths under an isolated `HOME`.

Wave C: **short paths** (help, unknown flag, mutual-exclusion / validation
errors that do not load config) run **in-process** via `agentcli.Run` (L2).
**`--show` paths** that must isolate `HOME` / seed config files keep a **binary
subprocess** with `cmd.Env` only (no process `Setenv`/`Chdir`) — short-path
binary debt until a Parallel-safe home inject exists; **not** labeled `e2e`.

# DSN (Domain Specific Notion)

**Participants**

- **In-process agentcli** — `agentcli.Run(RemoteProfile|LocalProfile, args)` for
  help / rejected leaves; stdout/stderr captured via temporary `os.Stdout`/
  `os.Stderr` swap under a suite mutex (`active` profile is package-global).
- **remote-agent / local-agent subprocess** — built only when a leaf needs
  isolated `HOME` for `--show` (seed or empty-file dump).
- **Isolated user HOME** — temp directory on the binary path; agent config under
  `~/.ai-critic/remote-agent-config.json` or `local-agent-config.json`.
- **Config file seed** — optional pretty JSON for `--show` content assertions.
- **session cache** — `DOCTEST_SESSION_ID` keys
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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run: package-level
// `active` profile and temporary stdout/stderr swaps are process-global.
var agentcliInProcessMu sync.Mutex

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

	// UseCLI forces the product binary path (cmd.Env HOME). Default: automatic —
	// binary when HOME isolation is required for --show; else in-process L2.
	UseCLI bool

	// Timeout overrides the default CLI kill timer (0 = default). Binary path only.
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

// needsIsolatedHOME is true when the leaf exercises config load under a temp
// HOME (--show dump / seed). Help and validation-only rejects do not.
func needsIsolatedHOME(req *Request) bool {
	if req.SeedConfig != nil || req.AlsoSeedRemoteConfig != nil {
		return true
	}
	hasShow, hasWeb := false, false
	for _, a := range req.Args {
		switch a {
		case "--show":
			hasShow = true
		case "--web":
			hasWeb = true
		}
	}
	// Successful --show loads UserHomeDir; --show --web fails before load.
	return hasShow && !hasWeb
}

// wantsFlagHelp is true when argv includes -h/--help. less-gen flags.Help
// calls os.Exit(0), which panics under testing — keep those on the binary path.
func wantsFlagHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// useBinaryPath selects L3-ish subprocess when HOME isolation or os.Exit help
// is required. Remaining short paths (bare help, validation rejects) are L2.
func useBinaryPath(req *Request) bool {
	return req.UseCLI || needsIsolatedHOME(req) || wantsFlagHelp(req.Args)
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Profile != ProfileRemote && req.Profile != ProfileLocal {
		return nil, fmt.Errorf("Request.Profile must be remote-agent or local-agent, got %q", req.Profile)
	}
	if len(req.Args) == 0 {
		req.Args = []string{"config"}
	}

	if !useBinaryPath(req) {
		return runConfigInProcess(t, req, resp)
	}
	return runConfigCLI(t, d, req, resp)
}

func runConfigInProcess(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	var profile agentcli.Profile
	switch req.Profile {
	case ProfileLocal:
		profile = agentcli.LocalProfile()
	default:
		profile = agentcli.RemoteProfile()
	}

	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	t.Logf("in-process %s argv: %v", req.Profile, req.Args)
	runErr := agentcli.Run(profile, req.Args)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	outBytes, _ := io.ReadAll(stdoutR)
	errBytes, _ := io.ReadAll(stderrR)
	_ = stdoutR.Close()
	_ = stderrR.Close()

	resp.Stdout = string(outBytes)
	resp.Stderr = string(errBytes)
	if runErr != nil {
		// Match cmd/remote-agent and cmd/local-agent main: print Error to stderr, exit 1.
		msg := fmt.Sprintf("Error: %v\n", runErr)
		resp.Stderr += msg
		resp.ExitCode = 1
	}
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	return resp, nil
}

func runConfigCLI(t *testing.T, d *session.Doctest, req *Request, resp *Response) (*Response, error) {
	t.Helper()
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

	t.Logf("%s argv: %v (HOME=%s) [binary]", req.Profile, req.Args, agentHome)

	cmd := exec.CommandContext(ctx, agentBin, req.Args...)
	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = stripEnvPrefix(agentEnv, "PATH=")
	agentEnv = append(agentEnv, "HOME="+agentHome)
	// tool_resolve init runs `npm bin -g` (and node probes). Under an empty
	// temp HOME that can hang forever; use a minimal PATH so probes fail fast.
	// Config leaves do not need npm/node on PATH.
	agentEnv = append(agentEnv, "PATH="+minimalAgentPATH())
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

// minimalAgentPATH is enough for the Go-built agent binary to run config help /
// show without tool_resolve init hanging on npm under empty HOME.
func minimalAgentPATH() string {
	// Keep go toolchain bin if present (not required for config, harmless).
	parts := []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin"}
	if home, err := os.UserHomeDir(); err == nil {
		// Prefer the parent process home's go/bin only for locating nothing
		// critical; empty agent HOME is separate.
		_ = home
	}
	return strings.Join(parts, string(os.PathListSeparator))
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
