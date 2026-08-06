# remote-agent sync unison — L2 doctests (P2: doctor + status)

Plan phase **P2** (classic TDD): injectable library + in-process CLI for
`sync unison doctor` and `sync unison status` — **without** real Unison binary,
SSH, or network when deps are mocked.

Package under test (P1 exists GREEN; P2 APIs may be missing → **RED**):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/synccmd
```

P1 sealed tree `sync_tests/` (28 leaves) is **out of scope** for this suite —
do not edit sealed P1 asserts. This tree is self-contained.

Out of scope for this tree (later phases): `run`, `install`, `reset-archives`,
real Unison transfer, real SSH serve e2e.

# DSN (Domain Specific Notion)

**Doctor** aggregates readiness checks for one named Unison pair using injectable
probes. **Status** reports pair identity, serve up/down, and last-run state from
optional on-disk state files.

**Participants**

- **synccmd.Doctor** — runs named checks with fakeable version/serve/path deps;
  returns a structured report (and error on resolution failure or critical fail).
- **synccmd.Status** — loads pairs (+ optional `{StoreDir}/state/<name>.json`);
  serve probe injectable; name omitted → all pairs.
- **synccmd.RunCLI** — argv after `sync`; dispatches `unison doctor|status`
  (and existing P1 CRUD); injectable dirs + probe hooks on `CLIOpts`.
- **Store / pairs.json** — pair definitions (P1); doctor requires loadable store
  and resolvable pair name.
- **State file** — `{StoreDir}/state/<name>.json` last-run metadata (written by
  future `run`; missing → status shows `never`).
- **Probe hooks** — `LocalVersion`, `RemoteVersion`, `ServeOK`, `RemotePathOK`
  on opts / CLIOpts (nil → product real defaults; tests always inject).

**Behaviors**

- Doctor resolves pair: explicit name → that pair; name empty → `defaultPair` if
  set, else sole pair if exactly one, else error containing `pair name required`.
- Unknown pair → error containing `unknown pair`.
- Doctor checks (stable names): `local-version`, `remote-version`,
  `versions-match`, `serve`, `local-root`, `remote-root`, `profile`, `pairs-json`.
- Versions match on version number equality (e.g. both `2.54.0`).
- Serve down → `serve` check fails; remote-root may also fail when serve down.
- Any critical check fail → library error and/or CLI non-nil `RunCLI` error;
  report still carries per-check rows.
- Status missing state → `LastRun` / display contains `never`; seeded state
  surfaces `lastRunAt` (or equivalent) from JSON.
- Unison help lists `doctor` and `status` verbs.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/synccmd/doctor_status_tests/   [Request{Mode, PairName, fakes, …}]
│                                           Run: seed → Doctor | Status | RunCLI
├── help/
│   └── lists-doctor-status/                # unison help includes doctor+status
├── doctor/
│   ├── success/
│   │   └── all-ok/                         # all checks pass (fakes)
│   ├── check-fail/
│   │   ├── version-mismatch/               # local ≠ remote version
│   │   ├── serve-down/                     # ServeOK returns error
│   │   └── local-root-missing/             # local path absent
│   ├── resolve/
│   │   ├── unknown-pair/                   # name not in store
│   │   └── name-omitted/
│   │       ├── default-pair/               # uses config.defaultPair
│   │       ├── single-pair/                # exactly one pair → use it
│   │       └── multi-require-name/         # multi pairs → pair name required
│   └── cli/
│       └── fail-exits/                     # RunCLI doctor; non-nil err on fail
└── status/
    ├── named/
    │   ├── never-run/                      # no state file → never
    │   ├── after-seeded-state/             # state JSON lastRunAt visible
    │   └── unknown-pair/
    └── all/
        └── two-pairs-never-run/            # name omitted → both pairs
```

**Significance order:** operation family (help | doctor | status) → outcome class
(success | check-fail | resolve | cli-exit | named | all) → variant.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/lists-doctor-status` | `unison --help` stdout lists `doctor` and `status` |
| 2 | `doctor/success/all-ok` | Doctor with matching versions, serve up, roots+profile → AllOK |
| 3 | `doctor/check-fail/version-mismatch` | local≠remote → `versions-match` fails; AllOK false |
| 4 | `doctor/check-fail/serve-down` | ServeOK error → `serve` fails |
| 5 | `doctor/check-fail/local-root-missing` | missing local dir → `local-root` fails |
| 6 | `doctor/resolve/unknown-pair` | Doctor name ghost → `unknown pair` |
| 7 | `doctor/resolve/name-omitted/default-pair` | empty name + defaultPair → that pair |
| 8 | `doctor/resolve/name-omitted/single-pair` | empty name + one pair → that pair |
| 9 | `doctor/resolve/name-omitted/multi-require-name` | empty name + multi → `pair name required` |
| 10 | `doctor/cli/fail-exits` | `unison doctor mad-max` with serve down → RunCLI error |
| 11 | `status/named/never-run` | status mad-max, no state → never |
| 12 | `status/named/after-seeded-state` | seeded state → lastRunAt visible |
| 13 | `status/named/unknown-pair` | status ghost → `unknown pair` |
| 14 | `status/all/two-pairs-never-run` | status no name → both pairs, never |

## Exported APIs (implementer contract — P2)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (additions on top of P1):

| Symbol | Role |
|--------|------|
| `DoctorOpts` | StoreDir, UnisonDir, SSHConfigDir, Name, LocalVersion, RemoteVersion, ServeOK, RemotePathOK |
| `DoctorCheck` | Name, OK, Detail |
| `DoctorReport` | PairName, Checks []DoctorCheck, AllOK bool |
| `Doctor` | `(opts DoctorOpts) (DoctorReport, error)` |
| `StatusOpts` | StoreDir, UnisonDir, SSHConfigDir, Name, ServeOK |
| `StatusItem` | Name, Local, Remote, ServeOK bool, LastRun string, LastExit *int (optional) |
| `StatusReport` | Items []StatusItem |
| `Status` | `(opts StatusOpts) (StatusReport, error)` |
| `CLIOpts` | P1 fields + LocalVersion, RemoteVersion, ServeOK, RemotePathOK hooks |
| `RunCLI` | dispatches `unison doctor [<name>]`, `unison status [<name>]` |
| `UnisonUsage` | includes `doctor` and `status` |

### DoctorOpts / StatusOpts fields

```go
type DoctorOpts struct {
    StoreDir, UnisonDir, SSHConfigDir string
    Name string // empty → resolve defaultPair / sole pair
    LocalVersion  func() (string, error)
    RemoteVersion func() (string, error)
    ServeOK       func() error            // nil error = up
    RemotePathOK  func(remote string) error
}

type StatusOpts struct {
    StoreDir, UnisonDir, SSHConfigDir string
    Name string // empty → all pairs
    ServeOK func() error
}
```

### Doctor check names (stable assert contract)

| Name | Pass when |
|------|-----------|
| `pairs-json` | store loads |
| `local-version` | LocalVersion returns non-empty version, nil err |
| `remote-version` | RemoteVersion returns non-empty version, nil err |
| `versions-match` | local and remote version strings equal (after trivial trim) |
| `serve` | ServeOK() == nil |
| `local-root` | pair.Local exists as directory |
| `remote-root` | RemotePathOK(remote) == nil (if hook nil and serve down → fail) |
| `profile` | `{UnisonDir}/remote-agent-<name>.prf` exists |

### Doctor error contracts

| Situation | Error must contain |
|-----------|-------------------|
| unknown pair | `unknown pair` |
| name omitted, multi pairs, no default | `pair name required` |
| any check failed (library and CLI) | non-nil error (prefer `doctor` or `fail` / `check`) |

When pair resolves but checks fail: return populated `DoctorReport` (AllOK false)
**and** non-nil error so CLI can exit non-zero without re-deriving.

### Status contracts

| Situation | Behavior |
|-----------|----------|
| named pair, no state file | Item.LastRun contains `never` (case-insensitive ok); ServeOK from probe |
| named pair, state present | LastRun reflects `lastRunAt` from `{StoreDir}/state/<name>.json` |
| name omitted | one StatusItem per pair |
| unknown name | error containing `unknown pair` |

State file JSON shape (minimal):

```json
{"lastRunAt":"2026-01-15T10:00:00Z","exitCode":0,"message":"ok"}
```

### CLI argv (after `sync`)

| Args | Behavior |
|------|----------|
| `unison doctor [<name>]` | Doctor; table-ish check lines on stdout; err if fail |
| `unison status [<name>]` | Status; pair lines on stdout |
| `unison` / `unison --help` | Usage listing doctor + status (+ P1 verbs) |

## How to Run

From module root:

```sh
doctest vet ./cmd/agentcli/synccmd/doctor_status_tests
doctest test ./cmd/agentcli/synccmd/doctor_status_tests
doctest test ./cmd/agentcli/synccmd/doctor_status_tests/doctor/success/all-ok
```

All leaves are L2 in-process (unlabeled). Expect **RED** until P2 APIs land.
P1: `doctest test ./cmd/agentcli/synccmd/sync_tests` must stay GREEN.

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
	"github.com/xhd2015/doctest/session"
)

// Request configures library Doctor/Status or in-process RunCLI.
// Parallel-safe: paths from d.DOCTEST_CASE; no Setenv/Chdir.
type Request struct {
	// Mode selects Run path: "doctor" | "status" | "cli".
	Mode string

	// PairName is the pair for doctor/status library calls (may be empty for resolve / all).
	PairName string

	// Args is RunCLI argv after `sync` when Mode=="cli".
	Args []string

	// Injectable absolute dirs (root Setup fills defaults under DOCTEST_CASE).
	StoreDir     string
	UnisonDir    string
	SSHConfigDir string

	// Probe hooks — set by leaves; nil means product defaults (tests inject).
	LocalVersion  func() (string, error)
	RemoteVersion func() (string, error)
	ServeOK       func() error
	RemotePathOK  func(remote string) error

	// SeedPairsJSON, when non-empty, written to {StoreDir}/pairs.json before Run.
	SeedPairsJSON string

	// SeedStateJSON maps pair name → raw JSON body for {StoreDir}/state/<name>.json.
	SeedStateJSON map[string]string

	// SeedProfile when true writes a profile for FocusPair/PairName via WriteUnisonProfile
	// (or a minimal file) after seeding store.
	SeedProfile bool

	// FocusPair is the pair used for profile seeding when PairName empty.
	FocusPair string

	// LocalPath / RemotePath workspace dirs under case.
	LocalPath  string
	RemotePath string

	// EnsureLocalRoot: when false, remove LocalPath after root mkdir so local-root fails.
	// Default true (root creates dir).
	EnsureLocalRoot bool
	// LocalRootEnsured tracks whether leaf overrode EnsureLocalRoot (internal).
	// Leaves set EnsureLocalRoot=false explicitly; root sets default true if zero-value
	// via EnsureLocalRootSet.
	EnsureLocalRootSet bool
}

// Response captures library reports and/or CLI I/O.
type Response struct {
	Stdout string
	Stderr string
	RunErr string // CLI RunCLI error string; empty if nil

	// Library doctor
	Doctor    synccmd.DoctorReport
	DoctorErr string

	// Library status
	Status    synccmd.StatusReport
	StatusErr string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	_ = d

	resp := &Response{}

	if err := applySeeds(req); err != nil {
		return nil, err
	}

	if req.EnsureLocalRootSet && !req.EnsureLocalRoot {
		_ = os.RemoveAll(req.LocalPath)
	}

	switch req.Mode {
	case "doctor":
		report, err := synccmd.Doctor(synccmd.DoctorOpts{
			StoreDir:      req.StoreDir,
			UnisonDir:     req.UnisonDir,
			SSHConfigDir:  req.SSHConfigDir,
			Name:          req.PairName,
			LocalVersion:  req.LocalVersion,
			RemoteVersion: req.RemoteVersion,
			ServeOK:       req.ServeOK,
			RemotePathOK:  req.RemotePathOK,
		})
		resp.Doctor = report
		if err != nil {
			resp.DoctorErr = err.Error()
		}
		return resp, nil

	case "status":
		report, err := synccmd.Status(synccmd.StatusOpts{
			StoreDir:     req.StoreDir,
			UnisonDir:    req.UnisonDir,
			SSHConfigDir: req.SSHConfigDir,
			Name:         req.PairName,
			ServeOK:      req.ServeOK,
		})
		resp.Status = report
		if err != nil {
			resp.StatusErr = err.Error()
		}
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
			RemoteVersion: req.RemoteVersion,
			ServeOK:       req.ServeOK,
			RemotePathOK:  req.RemotePathOK,
		})
		resp.Stdout = outBuf.String()
		resp.Stderr = errBuf.String()
		if err != nil {
			resp.RunErr = err.Error()
		}
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown Request.Mode %q (want doctor|status|cli)", req.Mode)
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
	if len(req.SeedStateJSON) > 0 {
		stateDir := filepath.Join(req.StoreDir, "state")
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return err
		}
		for name, body := range req.SeedStateJSON {
			p := filepath.Join(stateDir, name+".json")
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				return err
			}
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
		// Prefer public WriteUnisonProfile when pair is loadable; else write minimal file.
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

type simpleError struct{ s string }

func (e *simpleError) Error() string { return e.s }

// --- helpers used by leaves ---

// checkNamed returns the DoctorCheck with the given name, or nil.
func checkNamed(report synccmd.DoctorReport, name string) *synccmd.DoctorCheck {
	for i := range report.Checks {
		if report.Checks[i].Name == name {
			c := report.Checks[i]
			return &c
		}
	}
	return nil
}

// statusNamed returns the StatusItem with the given name, or nil.
func statusNamed(report synccmd.StatusReport, name string) *synccmd.StatusItem {
	for i := range report.Items {
		if report.Items[i].Name == name {
			it := report.Items[i]
			return &it
		}
	}
	return nil
}

// pairJSON builds a minimal version-1 pairs.json with one or more pairs.
// defaultPair may be empty. Paths should be absolute.
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

// onePair is a convenience Pair for seeds.
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

// remotePathFail always fails.
func remotePathFail(msg string) func(string) error {
	return func(string) error { return &simpleError{s: msg} }
}

// doctorHappyHooks sets matching versions, serve up, remote path ok.
func doctorHappyHooks(req *Request) {
	req.LocalVersion = fakeVersion("2.54.0")
	req.RemoteVersion = fakeVersion("2.54.0")
	req.ServeOK = serveUp()
	req.RemotePathOK = remotePathOK()
}
```
