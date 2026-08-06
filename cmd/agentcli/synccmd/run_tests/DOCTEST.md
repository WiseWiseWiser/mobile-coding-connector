# remote-agent sync unison — L2 doctests (P3: run + state)

Plan phase **P3** (classic TDD): injectable Unison argv builder + `RunPair` with
fake `Exec`, plus in-process CLI `sync unison run` — **without** a real Unison
binary, SSH, or network.

Package under test (P1+P2 exist GREEN; P3 APIs may be missing → **RED**):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/synccmd
```

Sealed trees **out of scope** (do not edit):

- `cmd/agentcli/synccmd/sync_tests/` (P1, 28 leaves)
- `cmd/agentcli/synccmd/doctor_status_tests/` (P2)

Out of scope for this tree (later phases): `install`, `reset-archives`, auto-start
serve, real multi-GB Unison transfer e2e.

# DSN (Domain Specific Notion)

**Run** loads a named pair, optionally gates on doctor, builds a Unison child
command (argv + child-only env), executes via injectable `Exec`, and writes
last-run state under `{StoreDir}/state/<name>.json`.

**Participants**

- **synccmd.BuildUnisonCmd** — pure builder: binary + profile basename
  `remote-agent-<name>`, batch/interactive flags, child env
  (`UNISONLOCALHOSTNAME`, optional `UNISON`→UnisonDir); never mutates process env.
- **synccmd.RunPair** — resolve pair → optional Doctor → Exec → write state →
  `RunResult` + error on non-zero exit.
- **synccmd.ExecFunc** — injectable child runner
  `(ctx, name, argv, env, stdout, stderr) (exitCode int, err error)`; production
  default uses `os/exec` with `cmd.Env` merge (no `Setenv` on the parent process).
- **synccmd.RunCLI** — argv after `sync`; dispatches `unison run <name>
  [--skip-doctor] [--interactive]`; wires CLIOpts.Exec + doctor probe hooks.
- **Store / pairs.json** — pair definitions (P1); run requires a resolvable name.
- **State file** — `{StoreDir}/state/<name>.json` written after Exec returns
  (exit code, lastRunAt, message; optional duration/versions).
- **Doctor probes** — reused from P2 when `--skip-doctor` is false; failure aborts
  before Exec.

**Behaviors**

- Argv: binary from `LocalUnisonPath` or `"unison"`; profile arg
  `remote-agent-<name>`; non-interactive may pass `-batch` when pair/batch policy
  wants batch; `--interactive` omits `-batch`.
- Child env always sets `UNISONLOCALHOSTNAME=<pair.LocalHostname>`; may set
  `UNISON=<UnisonDir>` so profile discovery finds `.prf` files.
- Skip doctor false + doctor fail → non-nil error, **no** Exec, **no** state write.
- Skip doctor true → Exec even if serve probe would fail.
- Exit code 0 → success (nil error); non-zero → error containing exit code or
  message; **both** write state.
- Unknown pair / missing name → clear errors; no Exec.
- Unison help lists `run`.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/synccmd/run_tests/     [Request{Mode, PairName, flags, Exec…}]
│                                   Run: seed → BuildUnisonCmd | RunPair | RunCLI
├── help/
│   └── lists-run/                  # unison help includes run
├── build/                          # BuildUnisonCmd pure
│   ├── batch-default/              # profile + UNISONLOCALHOSTNAME (+ batch)
│   └── interactive/                # interactive → no -batch in argv
├── library/                        # RunPair
│   ├── success/
│   │   └── writes-state-exit-0/    # fake Exec 0 → state exitCode 0
│   ├── non-zero/
│   │   └── writes-state-and-error/ # fake Exec non-zero → state + error
│   ├── doctor-gate/
│   │   ├── aborts-when-doctor-fails/       # no skip → no Exec
│   │   └── skip-doctor-allows-serve-down/  # skip → Exec despite serve down
│   └── resolve/
│       ├── unknown-pair/
│       └── missing-name/
└── cli/
    └── success-skip-doctor/        # RunCLI run --skip-doctor + fake Exec
```

**Significance order:** surface (help | build | library | cli) → outcome class
(success | non-zero | doctor-gate | resolve | flag variant) → concrete leaf.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/lists-run` | `unison --help` stdout lists `run` |
| 2 | `build/batch-default` | BuildUnisonCmd: profile name + `UNISONLOCALHOSTNAME` in env |
| 3 | `build/interactive` | Interactive build omits `-batch` |
| 4 | `library/success/writes-state-exit-0` | RunPair Exec 0 → state exitCode 0 |
| 5 | `library/non-zero/writes-state-and-error` | RunPair Exec ≠0 → state + error |
| 6 | `library/doctor-gate/aborts-when-doctor-fails` | doctor fail, no skip → no Exec |
| 7 | `library/doctor-gate/skip-doctor-allows-serve-down` | --skip-doctor + serve down → Exec |
| 8 | `library/resolve/unknown-pair` | unknown name → `unknown pair` |
| 9 | `library/resolve/missing-name` | empty name → requires name error |
| 10 | `cli/success-skip-doctor` | `unison run mad-max --skip-doctor` → state, nil err |

## Exported APIs (implementer contract — P3)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (additions on top of P1+P2):

| Symbol | Role |
|--------|------|
| `ExecFunc` | `func(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (exitCode int, err error)` |
| `RunOpts` | StoreDir, UnisonDir, SSHConfigDir, Name, SkipDoctor, Interactive, LocalUnisonPath, Exec, Stdout, Stderr, Context, doctor probe hooks |
| `RunResult` | ExitCode int, Message string, Duration (time.Duration or ms), Argv []string (optional echo) |
| `BuildUnisonCmd` | `(opts RunOpts) (argv []string, env []string, workdir string, err error)` |
| `RunPair` | `(opts RunOpts) (RunResult, error)` |
| `CLIOpts` | + `Exec ExecFunc` (doctor hooks already from P2) |
| `RunCLI` | dispatches `unison run <name> [--skip-doctor] [--interactive]` |
| `UnisonUsage` | includes `run` |

### RunOpts fields

```go
type ExecFunc func(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (exitCode int, err error)

type RunOpts struct {
    StoreDir, UnisonDir, SSHConfigDir string
    Name                              string
    SkipDoctor                        bool
    Interactive                       bool
    LocalUnisonPath                   string // empty → "unison"
    Exec                              ExecFunc
    Stdout, Stderr                    io.Writer
    Context                           context.Context // nil → context.Background()
    // Doctor probes when !SkipDoctor (nil → product defaults; tests inject).
    LocalVersion  func() (string, error)
    RemoteVersion func() (string, error)
    ServeOK       func() error
    RemotePathOK  func(remote string) error
}

type RunResult struct {
    ExitCode int
    Message  string
    Duration time.Duration // wall time of Exec; zero OK if unused
    Argv     []string      // optional: argv that was (or would be) executed
}
```

### BuildUnisonCmd / argv + env contract

| Piece | Rule |
|-------|------|
| `argv[0]` | `opts.LocalUnisonPath` if non-empty, else `"unison"` |
| profile | contains `remote-agent-<name>` (Unison profile basename without `.prf`) |
| batch | non-interactive + pair.Batch (default true): include `-batch` somewhere in argv |
| interactive | `opts.Interactive` true → argv must **not** contain `-batch` |
| env | slice suitable for `cmd.Env` (full child env or overlay); **must** include `UNISONLOCALHOSTNAME=<pair.LocalHostname>` |
| UNISON | may set `UNISON=<UnisonDir>` so Unison finds profiles under UnisonDir |
| process env | **never** `os.Setenv` for hostname; child env only |
| workdir | may be empty, UnisonDir, or local root — document in result; tests do not require a specific value unless set |

`BuildUnisonCmd` loads the pair by `Name` from the store; unknown/missing name → error.

### ExecFunc call shape

- `name` = executable (same as `argv[0]` from BuildUnisonCmd, or the binary path).
- `argv` = remaining arguments **after** the binary **or** full argv including `name` as `argv[0]` — implementer picks one shape and stays consistent; tests accept either by scanning for profile token and `-batch`.
- `env` = child environment (must include `UNISONLOCALHOSTNAME=…`).
- Writers: inherit `opts.Stdout` / `opts.Stderr` (no tee).

### RunPair flow

1. Require non-empty `Name` (else error mentioning `run` / `name` / `require`).
2. Load pair; unknown → error containing `unknown pair`.
3. If `!SkipDoctor`: run Doctor with same dirs + probe hooks; on fail return error **without** calling Exec and **without** writing state.
4. `BuildUnisonCmd` → call `Exec` (required for library path; nil Exec → clear error or default real exec — **tests always inject**).
5. Write `{StoreDir}/state/<name>.json` after Exec returns (including non-zero exit).
6. Exit 0 → `(result, nil)`; non-zero → `(result, error)` where error text contains the exit code (e.g. `exit code 1` / `exit 1`) or a non-empty message.

### State file JSON (minimal)

```json
{"lastRunAt":"<RFC3339>","exitCode":0,"message":"ok"}
```

| Field | Rule |
|-------|------|
| `lastRunAt` | non-empty RFC3339 (or RFC3339Nano) timestamp |
| `exitCode` | integer matching Exec exit code |
| `message` | optional summary string |
| duration / versions | optional; not required by asserts |

Compatible with P2 `Status` reader (`lastRunAt`, `exitCode`, `message`).

### CLI argv (after `sync`)

| Args | Behavior |
|------|----------|
| `unison run <name>` | RunPair; doctor on; batch from pair |
| `unison run <name> --skip-doctor` | skip doctor gate |
| `unison run <name> --interactive` | interactive (no `-batch`) |
| `unison run` (no name) | error (missing name) |
| `unison` / `unison --help` | Usage listing `run` (+ P1/P2 verbs) |

### Error substring contracts

| Situation | Error must contain |
|-----------|-------------------|
| unknown pair | `unknown pair` |
| missing name | non-empty; prefer `run` and/or `name` / `require` |
| doctor fail (no skip) | non-nil (prefer `doctor` / `fail` / `check`) |
| Exec non-zero | exit code digit(s) or non-empty failure text |
| nil Exec in library when required | non-nil (tests inject Exec) |

## How to Run

From module root:

```sh
doctest vet ./cmd/agentcli/synccmd/run_tests
doctest test ./cmd/agentcli/synccmd/run_tests
doctest test ./cmd/agentcli/synccmd/run_tests/library/success/writes-state-exit-0
```

All leaves are L2 in-process (unlabeled). Expect **RED** until P3 APIs land.
P1/P2: `sync_tests` and `doctor_status_tests` must stay GREEN.

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
	"github.com/xhd2015/doctest/session"
)

// Request configures BuildUnisonCmd / RunPair / in-process RunCLI.
// Parallel-safe: paths from d.DOCTEST_CASE; no Setenv/Chdir.
type Request struct {
	// Mode selects Run path: "build" | "run" | "cli".
	Mode string

	// PairName is the pair for build/run library calls.
	PairName string

	// Args is RunCLI argv after `sync` when Mode=="cli".
	Args []string

	// Injectable absolute dirs (root Setup fills defaults under DOCTEST_CASE).
	StoreDir     string
	UnisonDir    string
	SSHConfigDir string

	// Flags for library RunPair / BuildUnisonCmd.
	SkipDoctor      bool
	Interactive     bool
	LocalUnisonPath string

	// FakeExitCode is returned by the harness default Exec when Exec is nil.
	FakeExitCode int
	// FakeExecErr, when non-empty, makes default Exec return (FakeExitCode, error).
	FakeExecErr string
	// DisableDefaultExec: when true and Exec is nil, pass nil Exec to product
	// (only for leaves that assert product handling; default false).
	DisableDefaultExec bool

	// Custom Exec overrides the harness default fake (still wrapped for capture).
	Exec synccmd.ExecFunc

	// Probe hooks — set by leaves; nil means product defaults (tests inject).
	LocalVersion  func() (string, error)
	RemoteVersion func() (string, error)
	ServeOK       func() error
	RemotePathOK  func(remote string) error

	// SeedPairsJSON, when non-empty, written to {StoreDir}/pairs.json before Run.
	SeedPairsJSON string

	// SeedProfile when true writes a profile for FocusPair/PairName.
	SeedProfile bool

	// FocusPair is the pair used for profile seeding when PairName empty.
	FocusPair string

	// LocalPath / RemotePath workspace dirs under case.
	LocalPath  string
	RemotePath string
}

// Response captures builder output, RunPair result, CLI I/O, Exec capture, state.
type Response struct {
	Stdout string
	Stderr string
	RunErr string // CLI RunCLI error string; empty if nil

	// BuildUnisonCmd
	Argv     []string
	Env      []string
	WorkDir  string
	BuildErr string

	// RunPair
	Result     synccmd.RunResult
	RunPairErr string

	// Captured Exec invocation (library run or CLI when Exec injected).
	ExecCalled bool
	ExecName   string
	ExecArgv   []string
	ExecEnv    []string

	// State file after run
	StatePath    string
	StateExists  bool
	StateJSON    string
	StateExit    *int
	StateLastRun string
	StateMessage string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	_ = d

	resp := &Response{}

	if err := applySeeds(req); err != nil {
		return nil, err
	}

	// Recording Exec wrapper shared by run + cli modes.
	makeExec := func() synccmd.ExecFunc {
		base := req.Exec
		if base == nil && !req.DisableDefaultExec {
			exitCode := req.FakeExitCode
			errStr := req.FakeExecErr
			base = func(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (int, error) {
				_ = ctx
				_ = name
				_ = argv
				_ = env
				if stdout != nil {
					_, _ = io.WriteString(stdout, "fake-unison-ok\n")
				}
				if errStr != "" {
					return exitCode, fmt.Errorf("%s", errStr)
				}
				return exitCode, nil
			}
		}
		if base == nil {
			return nil
		}
		return func(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (int, error) {
			resp.ExecCalled = true
			resp.ExecName = name
			resp.ExecArgv = append([]string(nil), argv...)
			resp.ExecEnv = append([]string(nil), env...)
			return base(ctx, name, argv, env, stdout, stderr)
		}
	}

	switch req.Mode {
	case "build":
		argv, env, workdir, err := synccmd.BuildUnisonCmd(synccmd.RunOpts{
			StoreDir:        req.StoreDir,
			UnisonDir:       req.UnisonDir,
			SSHConfigDir:    req.SSHConfigDir,
			Name:            req.PairName,
			SkipDoctor:      req.SkipDoctor,
			Interactive:     req.Interactive,
			LocalUnisonPath: req.LocalUnisonPath,
		})
		resp.Argv = argv
		resp.Env = env
		resp.WorkDir = workdir
		if err != nil {
			resp.BuildErr = err.Error()
		}
		return resp, nil

	case "run":
		var outBuf, errBuf bytes.Buffer
		result, err := synccmd.RunPair(synccmd.RunOpts{
			StoreDir:        req.StoreDir,
			UnisonDir:       req.UnisonDir,
			SSHConfigDir:    req.SSHConfigDir,
			Name:            req.PairName,
			SkipDoctor:      req.SkipDoctor,
			Interactive:     req.Interactive,
			LocalUnisonPath: req.LocalUnisonPath,
			Exec:            makeExec(),
			Stdout:          &outBuf,
			Stderr:          &errBuf,
			LocalVersion:    req.LocalVersion,
			RemoteVersion:   req.RemoteVersion,
			ServeOK:         req.ServeOK,
			RemotePathOK:    req.RemotePathOK,
		})
		resp.Result = result
		resp.Stdout = outBuf.String()
		resp.Stderr = errBuf.String()
		if err != nil {
			resp.RunPairErr = err.Error()
		}
		loadState(req, resp)
		return resp, nil

	case "cli":
		var outBuf, errBuf bytes.Buffer
		args := req.Args
		if args == nil {
			args = []string{}
		}
		err := synccmd.RunCLI(args, synccmd.CLIOpts{
			StoreDir:      req.StoreDir,
			UnisonDir:     req.UnisonDir,
			SSHConfigDir:  req.SSHConfigDir,
			Stdout:        &outBuf,
			Stderr:        &errBuf,
			LocalVersion:  req.LocalVersion,
			RemoteVersion:  req.RemoteVersion,
			ServeOK:       req.ServeOK,
			RemotePathOK:  req.RemotePathOK,
			Exec:          makeExec(),
		})
		resp.Stdout = outBuf.String()
		resp.Stderr = errBuf.String()
		if err != nil {
			resp.RunErr = err.Error()
		}
		loadState(req, resp)
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown Request.Mode %q (want build|run|cli)", req.Mode)
	}
}

func applySeeds(req *Request) error {
	if req.SeedPairsJSON != "" {
		if err := os.MkdirAll(req.StoreDir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(req.StoreDir, "pairs.json")
		if err := os.WriteFile(path, []byte(req.SeedPairsJSON), 0o644); err != nil {
			return err
		}
	}
	if req.SeedProfile {
		name := req.FocusPair
		if name == "" {
			name = req.PairName
		}
		if name == "" {
			return fmt.Errorf("SeedProfile requires FocusPair or PairName")
		}
		st := &synccmd.Store{Dir: req.StoreDir}
		if p, err := synccmd.GetPair(st, name); err == nil && p != nil {
			if _, werr := synccmd.WriteUnisonProfile(req.UnisonDir, req.SSHConfigDir, p); werr != nil {
				return werr
			}
		} else {
			if err := os.MkdirAll(req.UnisonDir, 0o755); err != nil {
				return err
			}
			path := filepath.Join(req.UnisonDir, synccmd.ProfileFileName(name))
			if err := os.WriteFile(path, []byte("# test profile\n"), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadState(req *Request, resp *Response) {
	name := req.PairName
	if name == "" {
		name = req.FocusPair
	}
	// CLI leaves may only set Args; try to recover name from Args.
	if name == "" && len(req.Args) >= 3 && req.Args[0] == "unison" && req.Args[1] == "run" {
		if !strings.HasPrefix(req.Args[2], "-") {
			name = req.Args[2]
		}
	}
	if name == "" {
		return
	}
	path := filepath.Join(req.StoreDir, "state", name+".json")
	resp.StatePath = path
	data, err := os.ReadFile(path)
	if err != nil {
		resp.StateExists = false
		return
	}
	resp.StateExists = true
	resp.StateJSON = string(data)
	var raw struct {
		LastRunAt string `json:"lastRunAt"`
		ExitCode  *int   `json:"exitCode"`
		Message   string `json:"message"`
	}
	if jerr := json.Unmarshal(data, &raw); jerr == nil {
		resp.StateLastRun = raw.LastRunAt
		resp.StateExit = raw.ExitCode
		resp.StateMessage = raw.Message
	}
}

type simpleError struct{ s string }

func (e *simpleError) Error() string { return e.s }

// --- helpers used by leaves ---

// pairJSON builds a minimal version-1 pairs.json with one or more pairs.
func pairJSON(defaultPair string, pairs ...synccmd.Pair) string {
	cfg := synccmd.Config{
		Version:     1,
		DefaultPair: defaultPair,
		Pairs:       pairs,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

// onePair is a convenience Pair for seeds (batch true default).
func onePair(name, local, remote string) synccmd.Pair {
	return synccmd.Pair{
		Name:          name,
		Backend:       "unison",
		Local:         local,
		Remote:        remote,
		Prefer:        "newer",
		Ignore:        []string{"Name .DS_Store"},
		LocalHostname: "remote-agent-" + name,
		RemoteUnison:  "/usr/local/bin/unison",
		Times:         true,
		Auto:          true,
		Batch:         true,
	}
}

// fakeVersion returns a version hook that always yields v.
func fakeVersion(v string) func() (string, error) {
	return func() (string, error) { return v, nil }
}

// serveUp is ServeOK that succeeds.
func serveUp() func() error {
	return func() error { return nil }
}

// serveDown is ServeOK that fails.
func serveDown(msg string) func() error {
	return func() error { return &simpleError{s: msg} }
}

// remotePathOK always succeeds.
func remotePathOK() func(string) error {
	return func(string) error { return nil }
}

// doctorHappyHooks sets matching versions, serve up, remote path ok.
func doctorHappyHooks(req *Request) {
	req.LocalVersion = fakeVersion("2.54.0")
	req.RemoteVersion = fakeVersion("2.54.0")
	req.ServeOK = serveUp()
	req.RemotePathOK = remotePathOK()
}

// envHas reports whether env slice has KEY=value or KEY with value matching wantVal
// (wantVal empty → any value for KEY).
func envHas(env []string, key, wantVal string) bool {
	prefix := key + "="
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			continue
		}
		val := strings.TrimPrefix(e, prefix)
		if wantVal == "" || val == wantVal {
			return true
		}
	}
	return false
}

// argvJoined returns strings.Join(argv, " ") for substring scans.
func argvJoined(argv []string) string {
	return strings.Join(argv, " ")
}

// argvHasToken reports exact token presence in argv.
func argvHasToken(argv []string, tok string) bool {
	for _, a := range argv {
		if a == tok {
			return true
		}
	}
	return false
}

// seedMadMax seeds a standard mad-max pair + profile + happy doctor hooks.
func seedMadMax(req *Request) {
	req.PairName = "mad-max"
	req.FocusPair = "mad-max"
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	req.SeedProfile = true
	doctorHappyHooks(req)
}
```
