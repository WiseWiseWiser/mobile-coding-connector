# remote-agent sync unison — L2 doctests (P4: install, thin)

Plan phase **P4** (classic TDD): injectable `Install` that ensures local and/or
remote Unison binaries via fakeable hooks, plus CLI
`sync unison install [--local|--remote|--both]` (default both) — **without**
brew, HTTP download, real SSH, or real Unison binary.

Package under test (P1–P3 exist GREEN; P4 APIs may be missing → **RED**):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/synccmd
```

Sealed trees **out of scope** (do not edit):

- `cmd/agentcli/synccmd/sync_tests/` (P1)
- `cmd/agentcli/synccmd/doctor_status_tests/` (P2)
- `cmd/agentcli/synccmd/run_tests/` (P3)

Out of scope for this tree: real brew/HTTP install, `reset-archives`, e2e.

# DSN (Domain Specific Notion)

**Install** ensures Unison is present at the preferred pin version on local
and/or remote sides. Tests never shell out: hooks return version strings or
errors.

**Participants**

- **synccmd.Install** — library entry: scope (local / remote / both) + injectable
  `LocalEnsure` / `RemoteEnsure` / optional `WhichLocal`; returns `InstallReport`
  and error when a requested side fails.
- **synccmd.InstallOpts / InstallReport** — options and structured result
  (versions, OK flags, messages).
- **synccmd.PreferredUnisonVersion** — pinned version constant (`"2.54.0"`)
  referenced by install report/messages.
- **synccmd.RunCLI** — argv after `sync`; dispatches `unison install` with
  `--local` / `--remote` / `--both` (default both); wires hooks on `CLIOpts`.
- **Hook fakes** — L2 always injects `LocalEnsure` / `RemoteEnsure` (nil → product
  may shell out; tests never leave them nil).

**Behaviors**

- Scope empty or `--both` / no flags → ensure local **and** remote (both hooks).
- `--local` / scope `local` → only `LocalEnsure`; remote not called.
- `--remote` / scope `remote` → only `RemoteEnsure(targetPath)`; local not called.
- Local hook error → non-nil error; `LocalOK` false; error text surfaces.
- Success → corresponding `*OK` true and version strings from hooks in report.
- Unison help lists `install`.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/synccmd/install_tests/   [Request{Mode, Scope, hooks, Args…}]
│                                     Run: Install | RunCLI (hooks always injected)
├── help/
│   └── lists-install/                # unison help includes install
├── library/                          # synccmd.Install
│   ├── local/
│   │   ├── success/                  # LocalEnsure ok; remote not called
│   │   └── hook-error/               # LocalEnsure err surfaces
│   ├── remote/
│   │   └── success/                  # RemoteEnsure ok; local not called
│   └── both/
│       └── success/                  # both hooks called
└── cli/
    └── default-both/                 # unison install (no flags) → both
```

**Significance order:** surface (help | library | cli) → scope (local | remote |
both | default) → outcome (success | hook-error).

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/lists-install` | `unison --help` stdout lists `install` |
| 2 | `library/local/success` | Install scope local → LocalEnsure only; LocalOK + version |
| 3 | `library/local/hook-error` | LocalEnsure error → Install err; LocalOK false |
| 4 | `library/remote/success` | Install scope remote → RemoteEnsure only; RemoteOK + version |
| 5 | `library/both/success` | Install scope both → both hooks; both OK + Preferred pin |
| 6 | `cli/default-both` | `unison install` no flags → both hooks; RunErr empty |

## Exported APIs (implementer contract — P4)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (additions on top of P1–P3):

| Symbol | Role |
|--------|------|
| `PreferredUnisonVersion` | `const` string pin, value `"2.54.0"` |
| `InstallOpts` | Scope, LocalEnsure, RemoteEnsure, WhichLocal, RemoteTargetPath, writers |
| `InstallReport` | LocalVersion, RemoteVersion, LocalOK, RemoteOK, Messages |
| `Install` | `(opts InstallOpts) (InstallReport, error)` |
| `CLIOpts` | + LocalEnsure, RemoteEnsure, WhichLocal, RemoteTargetPath |
| `RunCLI` | dispatches `unison install [--local\|--remote\|--both]` |
| `UnisonUsage` | includes `install` |

### InstallOpts / InstallReport

```go
const PreferredUnisonVersion = "2.54.0"

type InstallOpts struct {
    // Scope is "local", "remote", or "both". Empty → "both".
    Scope string

    // LocalEnsure ensures local unison is installed/available; returns version.
    // Nil → product may shell out (brew/path); tests always inject.
    LocalEnsure func() (version string, err error)

    // RemoteEnsure places/ensures remote binary at targetPath; returns version.
    // Nil → product may download/scp; tests always inject.
    RemoteEnsure func(targetPath string) (version string, err error)

    // WhichLocal optionally probes existing local binary (path + version).
    // Nil OK; not required by thin P4 leaves.
    WhichLocal func() (path, version string, err error)

    // RemoteTargetPath is passed to RemoteEnsure. Empty → product default path.
    RemoteTargetPath string

    Stdout io.Writer
    Stderr io.Writer
}

type InstallReport struct {
    LocalVersion  string
    RemoteVersion string
    LocalOK       bool
    RemoteOK      bool
    Messages      []string // optional human lines; may mention PreferredUnisonVersion
}
```

### Scope / CLI mapping

| Scope / flags | LocalEnsure | RemoteEnsure |
|---------------|-------------|--------------|
| `local` / `--local` | called | **not** called |
| `remote` / `--remote` | **not** called | called with RemoteTargetPath (or product default) |
| `both` / `--both` / empty / no flags | called | called |

### Success / error contracts

| Situation | Report | Error |
|-----------|--------|-------|
| local success | LocalOK true; LocalVersion = hook version | nil |
| remote success | RemoteOK true; RemoteVersion = hook version | nil |
| both success | both OK + both versions | nil |
| LocalEnsure error (local or both) | LocalOK false | non-nil; text contains hook error or `install` / `local` |
| RemoteEnsure error | RemoteOK false | non-nil (not asserted in thin tree) |

`PreferredUnisonVersion` must equal `"2.54.0"`. Report may surface it in
`Messages` or product docs; leaves assert the constant and hook-returned versions.

### CLI argv (after `sync`)

| Args | Behavior |
|------|----------|
| `unison install` | scope both (default) |
| `unison install --local` | scope local |
| `unison install --remote` | scope remote |
| `unison install --both` | scope both |
| `unison` / `unison --help` | Usage listing `install` (+ P1–P3 verbs) |

Unknown install flags → non-nil error (not covered by thin leaves).

### CLIOpts install fields

```go
// Additional CLIOpts fields for install (beside P1–P3):
LocalEnsure      func() (string, error)
RemoteEnsure     func(targetPath string) (string, error)
WhichLocal       func() (path, version string, err error)
RemoteTargetPath string
```

## How to Run

From module root:

```sh
doctest vet ./cmd/agentcli/synccmd/install_tests
doctest test ./cmd/agentcli/synccmd/install_tests
doctest test ./cmd/agentcli/synccmd/install_tests/library/both/success
```

All leaves are L2 in-process (unlabeled). Expect **RED** until P4 APIs land.
P1–P3 sealed trees must stay GREEN and unmodified.

```go
import (
	"bytes"
	"fmt"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
	"github.com/xhd2015/doctest/session"
)

// Request configures library Install or in-process RunCLI.
// Parallel-safe: paths from d.DOCTEST_CASE; no Setenv/Chdir.
type Request struct {
	// Mode selects Run path: "install" | "cli".
	Mode string

	// Scope for library Install: "local" | "remote" | "both" | "" (→ both).
	Scope string

	// Args is RunCLI argv after `sync` when Mode=="cli".
	Args []string

	// RemoteTargetPath passed to Install / CLIOpts (and recorded for RemoteEnsure).
	RemoteTargetPath string

	// Custom hooks (optional). When nil, harness builds fakes from Fake* fields.
	LocalEnsure  func() (string, error)
	RemoteEnsure func(targetPath string) (string, error)
	WhichLocal   func() (path, version string, err error)

	// FakeLocalVersion / FakeRemoteVersion returned by default harness hooks.
	// Empty → "2.54.0".
	FakeLocalVersion  string
	FakeRemoteVersion string

	// LocalEnsureErr / RemoteEnsureErr: when non-empty, default hook returns that error.
	LocalEnsureErr  string
	RemoteEnsureErr string

	// DisableDefaultLocal / DisableDefaultRemote: pass nil for that hook to product
	// (not used by thin leaves; tests inject hooks).
	DisableDefaultLocal  bool
	DisableDefaultRemote bool
}

// Response captures Install report, CLI I/O, and hook invocation capture.
type Response struct {
	Stdout string
	Stderr string
	RunErr string // CLI RunCLI error string; empty if nil

	Report     synccmd.InstallReport
	InstallErr string

	LocalEnsureCalled  bool
	RemoteEnsureCalled bool
	LocalEnsureCalls   int
	RemoteEnsureCalls  int
	RemoteEnsurePath   string // last targetPath passed to RemoteEnsure
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	_ = d

	resp := &Response{}

	localVer := req.FakeLocalVersion
	if localVer == "" {
		localVer = "2.54.0"
	}
	remoteVer := req.FakeRemoteVersion
	if remoteVer == "" {
		remoteVer = "2.54.0"
	}

	makeLocal := func() func() (string, error) {
		base := req.LocalEnsure
		if base == nil && !req.DisableDefaultLocal {
			errStr := req.LocalEnsureErr
			v := localVer
			base = func() (string, error) {
				if errStr != "" {
					return "", fmt.Errorf("%s", errStr)
				}
				return v, nil
			}
		}
		if base == nil {
			return nil
		}
		return func() (string, error) {
			resp.LocalEnsureCalled = true
			resp.LocalEnsureCalls++
			return base()
		}
	}

	makeRemote := func() func(string) (string, error) {
		base := req.RemoteEnsure
		if base == nil && !req.DisableDefaultRemote {
			errStr := req.RemoteEnsureErr
			v := remoteVer
			base = func(targetPath string) (string, error) {
				if errStr != "" {
					return "", fmt.Errorf("%s", errStr)
				}
				return v, nil
			}
		}
		if base == nil {
			return nil
		}
		return func(targetPath string) (string, error) {
			resp.RemoteEnsureCalled = true
			resp.RemoteEnsureCalls++
			resp.RemoteEnsurePath = targetPath
			return base(targetPath)
		}
	}

	switch req.Mode {
	case "install":
		report, err := synccmd.Install(synccmd.InstallOpts{
			Scope:            req.Scope,
			LocalEnsure:      makeLocal(),
			RemoteEnsure:     makeRemote(),
			WhichLocal:       req.WhichLocal,
			RemoteTargetPath: req.RemoteTargetPath,
		})
		resp.Report = report
		if err != nil {
			resp.InstallErr = err.Error()
		}
		return resp, nil

	case "cli":
		var outBuf, errBuf bytes.Buffer
		args := req.Args
		if args == nil {
			args = []string{}
		}
		err := synccmd.RunCLI(args, synccmd.CLIOpts{
			Stdout:           &outBuf,
			Stderr:           &errBuf,
			LocalEnsure:      makeLocal(),
			RemoteEnsure:     makeRemote(),
			WhichLocal:       req.WhichLocal,
			RemoteTargetPath: req.RemoteTargetPath,
		})
		resp.Stdout = outBuf.String()
		resp.Stderr = errBuf.String()
		if err != nil {
			resp.RunErr = err.Error()
		}
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown Request.Mode %q (want install|cli)", req.Mode)
	}
}

// --- helpers used by leaves ---

// wantPreferred is the pin asserted by leaves (must match PreferredUnisonVersion).
const wantPreferred = "2.54.0"
```
