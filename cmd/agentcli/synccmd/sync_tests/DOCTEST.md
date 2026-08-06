# remote-agent sync unison — L2 doctests (P1: pair store + CRUD + profile)

Plan phase **P1** (classic TDD / greenfield): injectable library + in-process CLI
for `remote-agent sync` / `remote-agent sync unison` pair store CRUD and Unison
profile emit — **without** real Unison binary, SSH, or network.

Package under test (intended, may not exist yet → **RED**):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/synccmd
```

Out of scope for this tree (later phases): `doctor`, `status`, `run`, `install`,
`reset-archives`, auto-serve, real Unison/SSH e2e, top-level `agentcli` switch
wiring beyond `synccmd.RunCLI`.

# DSN (Domain Specific Notion)

**remote-agent sync unison** manages named local↔remote Unison pair definitions
and regenerable profile files used later as `unison remote-agent-<name>`.

**Participants**

- **synccmd.RunCLI** — in-process CLI entry for argv after `sync`; injectable
  `StoreDir` / `UnisonDir` / `SSHConfigDir` and stdout/stderr writers.
- **Store** — loads/saves `{StoreDir}/pairs.json` (versioned pair list).
- **Pair** — one named sync definition (local/remote paths, prefer, ignore,
  localHostname, remoteUnison, times/auto/batch).
- **WriteUnisonProfile / RenderUnisonProfile** — emit
  `{UnisonDir}/remote-agent-<name>.prf` from a Pair + SSHConfigDir.
- **InitPair / SetPair / RmPair / ListPairs / GetPair** — library CRUD used by
  CLI and tests.

**Behaviors**

- Help at sync and unison levels prints Usage (trailing `\n`); nil error.
- `unison init|add <name> <local> <remote>` creates pair with defaults, writes
  store + profile; duplicate name errors.
- `list` / `show` read store; empty list OK; unknown pair errors.
- `set` partial-updates fields and regenerates profile when paths/options change.
- `rm` removes pair; purge profile by default (`--no-purge-profile` keeps `.prf`).
- Profile contains dual roots (`root =`, `ssh://remote-agent//…`), `sshargs` with
  `-F {SSHConfigDir}/ssh_config`, `servercmd`, prefer/auto/batch/times, ignore lines.
- Unknown subcommands and missing required args surface clear `error` strings.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/synccmd/sync_tests/     [Request{Args, StoreDir, …}]
│                                    Run: PreArgs* + RunCLI + read store/profile
├── help/                            # help / usage surfaces
│   ├── sync/
│   │   ├── bare/                    # [] → sync help
│   │   └── long-flag/               # --help → sync help
│   └── unison/
│       ├── bare/                    # [unison] → unison help
│       └── long-flag/               # [unison --help] → unison help
├── init/                            # create pair (init|add)
│   ├── success/
│   │   ├── defaults/                # name+local+remote → store+profile defaults
│   │   ├── add-alias/               # add == init
│   │   └── custom-options/          # prefer, local-hostname, remote-unison
│   └── errors/
│       ├── missing-args/            # incomplete argv
│       └── duplicate-name/          # pair already exists
├── list/
│   ├── empty/                       # no pairs yet
│   └── after-two-inits/             # two names visible
├── show/
│   ├── found/                       # fields for existing pair
│   └── errors/
│       ├── unknown-pair/
│       └── missing-name/
├── set/
│   ├── success/
│   │   ├── local-remote-regen-profile/
│   │   ├── prefer-bool-flags/
│   │   ├── ignore-replace/
│   │   └── local-hostname/
│   └── errors/
│       ├── unknown-pair/
│       └── missing-name/
├── rm/
│   ├── success/
│   │   ├── purge-profile/           # default / --purge-profile removes .prf
│   │   └── no-purge-profile/        # --no-purge-profile keeps .prf
│   └── errors/
│       ├── unknown-pair/
│       └── missing-name/
├── profile/
│   └── critical-lines/              # golden-ish profile content contracts
├── unknown/
│   ├── sync-subcommand/             # unknown under sync
│   └── unison-subcommand/           # unknown under unison
└── store/
    └── corrupt-json/                # bad pairs.json → CLI error
```

**Significance order:** CLI operation family (help | init | list | show | set |
rm | unknown | store-edge) → outcome (success | errors) → flag/arg variant.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/sync/bare` | no args → sync Usage on stdout; nil err; trailing newline |
| 2 | `help/sync/long-flag` | `--help` → same sync help contract |
| 3 | `help/unison/bare` | `unison` alone → unison Usage listing CRUD verbs |
| 4 | `help/unison/long-flag` | `unison --help` → unison help |
| 5 | `init/success/defaults` | init name+abs paths → pairs.json + profile + defaults |
| 6 | `init/success/add-alias` | `add` creates pair like init |
| 7 | `init/success/custom-options` | `--prefer`, `--local-hostname`, `--remote-unison` stored |
| 8 | `init/errors/missing-args` | incomplete init argv → error |
| 9 | `init/errors/duplicate-name` | second init same name → already exists |
| 10 | `list/empty` | empty store → empty list output, nil err |
| 11 | `list/after-two-inits` | two pairs → both names on stdout |
| 12 | `show/found` | show existing → name/local/remote (and key fields) |
| 13 | `show/errors/unknown-pair` | show missing name → unknown pair error |
| 14 | `show/errors/missing-name` | show without name → error |
| 15 | `set/success/local-remote-regen-profile` | set paths → store + regenerated profile roots |
| 16 | `set/success/prefer-bool-flags` | prefer + --no-times/--no-auto/--no-batch |
| 17 | `set/success/ignore-replace` | `--ignore` lines replace ignore list + profile |
| 18 | `set/success/local-hostname` | set `--local-hostname` updates pair field |
| 19 | `set/errors/unknown-pair` | set unknown → error |
| 20 | `set/errors/missing-name` | set without name → error |
| 21 | `rm/success/purge-profile` | rm removes pair and `.prf` |
| 22 | `rm/success/no-purge-profile` | `--no-purge-profile` keeps `.prf` |
| 23 | `rm/errors/unknown-pair` | rm unknown → error |
| 24 | `rm/errors/missing-name` | rm without name → error |
| 25 | `profile/critical-lines` | profile contains root/sshargs/servercmd/prefer/ignore |
| 26 | `unknown/sync-subcommand` | `sync foo` → unknown subcommand error |
| 27 | `unknown/unison-subcommand` | `unison foo` → unknown unison subcommand |
| 28 | `store/corrupt-json` | corrupt pairs.json → list/show surface error |

## Exported APIs (implementer contract)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd`:

| Symbol | Role |
|--------|------|
| `Pair` | name, backend, local, remote, prefer, ignore, localHostname, remoteUnison, times, auto, batch |
| `Config` | version, defaultPair, pairs |
| `Store` | `{Dir string}` — file `{Dir}/pairs.json` |
| `(*Store).Load` | `() (*Config, error)` — missing file → empty Config (version 1, no pairs), nil error |
| `(*Store).Save` | `(*Config) error` |
| `InitOpts` / `InitPair` | create pair; error if name exists; applies defaults |
| `SetOpts` / `SetPair` | partial update by name; regenerates profile when called from CLI |
| `RmOpts` / `RmPair` | remove pair; `PurgeProfile` controls `.prf` deletion |
| `GetPair` / `ListPairs` | read helpers |
| `RenderUnisonProfile` | `(sshConfigDir string, p *Pair) string` |
| `WriteUnisonProfile` | `(unisonDir, sshConfigDir string, p *Pair) (path string, err error)` → `{unisonDir}/remote-agent-<name>.prf` |
| `ProfileFileName` | `(name string) string` → `remote-agent-<name>.prf` |
| `CLIOpts` | StoreDir, UnisonDir, SSHConfigDir, Stdout, Stderr |
| `RunCLI` | `(args []string, opts CLIOpts) error` — argv **after** `sync` |

### Pair defaults (init, flags omitted)

| Field | Default |
|-------|---------|
| backend | `unison` |
| prefer | `newer` |
| localHostname | `remote-agent-<name>` |
| remoteUnison | `/usr/local/bin/unison` |
| times, auto, batch | `true` |
| ignore | `Name .DS_Store`, `Name node_modules`, `Name *.log`, `Path tmp`, `Path log` |

Paths: callers should pass absolute local/remote; library may `filepath.Abs` when relative.

### Profile wire shape (critical lines)

```
root = <local abs>
root = ssh://remote-agent//<remote abs without leading issues; form ssh://remote-agent//abs/path>
sshargs = -F <SSHConfigDir>/ssh_config …
servercmd = <remoteUnison>
auto = true|false
batch = true|false
times = true|false
prefer = <prefer>
ignore = <each ignore entry>
```

### CLI argv (after `sync`)

| Args | Behavior |
|------|----------|
| `[]`, `help`, `-h`, `--help` | sync help |
| `unison`, `unison help`, `unison -h/--help` | unison help |
| `unison init\|add <name> <local> <remote> [flags]` | create |
| `unison list` | list names |
| `unison show <name>` | show one |
| `unison set <name> [flags]` | partial update + regen profile |
| `unison rm <name> [--yes] [--purge-profile\|--no-purge-profile]` | delete; **default purge profile** |

Init/set flags (subset): `--prefer`, `--local-hostname`, `--remote-unison`,
`--times`/`--no-times`, `--auto`/`--no-auto`, `--batch`/`--no-batch`,
`--ignore LINE` (repeatable; if any present on set, **replaces** ignore list).

### Error substring contracts (case-sensitive enough to match)

| Situation | Error must contain |
|-----------|-------------------|
| duplicate init | `already exists` |
| unknown pair | `unknown pair` |
| unknown sync subcommand | `unknown` |
| unknown unison subcommand | `unknown` |
| missing required args | non-empty error (prefer mentioning the verb or `require`) |

## How to Run

From module root:

```sh
doctest vet ./cmd/agentcli/synccmd/sync_tests
doctest test ./cmd/agentcli/synccmd/sync_tests
doctest test ./cmd/agentcli/synccmd/sync_tests/help/sync/bare
```

All leaves are L2 in-process (unlabeled). Expect **RED** until `synccmd` is implemented.

```go
import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
	"github.com/xhd2015/doctest/session"
)

// Request configures in-process CLI argv and injectable absolute dirs.
// Parallel-safe: paths from d.DOCTEST_CASE; no Setenv/Chdir.
type Request struct {
	// Args is the primary RunCLI argv after the `sync` subcommand.
	Args []string

	// PreArgs are optional RunCLI invocations run in order before Args
	// (e.g. two inits before list). Each entry is one argv slice.
	PreArgs [][]string

	// Injectable absolute dirs (root Setup fills defaults under DOCTEST_CASE).
	StoreDir     string
	UnisonDir    string
	SSHConfigDir string

	// FocusPair is the pair name whose profile is loaded into Response after Run.
	// Empty → no profile read (or first pair if store has exactly one — leaves set explicitly).
	FocusPair string

	// SeedPairsJSON, when non-empty, is written to {StoreDir}/pairs.json before PreArgs/Args.
	SeedPairsJSON string

	// LocalPath / RemotePath helpers for leaves that need abs workspace dirs.
	// Root Setup creates empty dirs when non-empty paths are set by leaves, or leaves set full abs.
	LocalPath  string
	RemotePath string
}

// Response captures CLI I/O, error, and on-disk side effects.
type Response struct {
	Stdout string
	Stderr string
	RunErr string // final Args RunCLI error string; empty if nil

	// PreErr is the first PreArgs error if any (Args still may run only if Pre ok — harness stops on PreErr).
	PreErr string

	// Store
	PairsJSON       string
	PairsJSONExists bool
	Config          *synccmd.Config
	LoadErr         string

	// Profile for FocusPair
	ProfilePath    string
	ProfileContent string
	ProfileExists  bool
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	_ = d

	resp := &Response{}

	if req.SeedPairsJSON != "" {
		if err := os.MkdirAll(req.StoreDir, 0o755); err != nil {
			return nil, err
		}
		path := filepath.Join(req.StoreDir, "pairs.json")
		if err := os.WriteFile(path, []byte(req.SeedPairsJSON), 0o644); err != nil {
			return nil, err
		}
	}

	// Ensure workspace path dirs exist when leaves set them (absolute under case).
	for _, p := range []string{req.LocalPath, req.RemotePath} {
		if p == "" {
			continue
		}
		if err := os.MkdirAll(p, 0o755); err != nil {
			return nil, err
		}
	}

	runOne := func(args []string) (stdout, stderr string, errStr string) {
		var outBuf, errBuf bytes.Buffer
		err := synccmd.RunCLI(args, synccmd.CLIOpts{
			StoreDir:     req.StoreDir,
			UnisonDir:    req.UnisonDir,
			SSHConfigDir: req.SSHConfigDir,
			Stdout:       &outBuf,
			Stderr:       &errBuf,
		})
		if err != nil {
			errStr = err.Error()
		}
		return outBuf.String(), errBuf.String(), errStr
	}

	var stdoutAll, stderrAll strings.Builder
	for _, pre := range req.PreArgs {
		o, e, es := runOne(pre)
		stdoutAll.WriteString(o)
		stderrAll.WriteString(e)
		if es != "" {
			resp.PreErr = es
			resp.Stdout = stdoutAll.String()
			resp.Stderr = stderrAll.String()
			// Still try to load store for debugging; do not run main Args.
			loadStore(req, resp)
			loadProfile(req, resp)
			return resp, nil
		}
	}

	args := req.Args
	if args == nil {
		args = []string{}
	}
	o, e, es := runOne(args)
	stdoutAll.WriteString(o)
	stderrAll.WriteString(e)
	resp.Stdout = stdoutAll.String()
	resp.Stderr = stderrAll.String()
	resp.RunErr = es

	loadStore(req, resp)
	loadProfile(req, resp)
	return resp, nil
}

func loadStore(req *Request, resp *Response) {
	path := filepath.Join(req.StoreDir, "pairs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			resp.LoadErr = err.Error()
		}
		return
	}
	resp.PairsJSONExists = true
	resp.PairsJSON = string(data)
	// Prefer library Load when available; fall back to json for RED-phase resilience.
	st := &synccmd.Store{Dir: req.StoreDir}
	cfg, loadErr := st.Load()
	if loadErr != nil {
		resp.LoadErr = loadErr.Error()
		// still try raw JSON
		var raw synccmd.Config
		if jerr := json.Unmarshal(data, &raw); jerr == nil {
			resp.Config = &raw
		}
		return
	}
	resp.Config = cfg
}

func loadProfile(req *Request, resp *Response) {
	name := req.FocusPair
	if name == "" {
		return
	}
	// ProfileFileName is part of public API.
	fname := synccmd.ProfileFileName(name)
	path := filepath.Join(req.UnisonDir, fname)
	resp.ProfilePath = path
	data, err := os.ReadFile(path)
	if err != nil {
		resp.ProfileExists = false
		return
	}
	resp.ProfileExists = true
	resp.ProfileContent = string(data)
}

// pairByName returns the pair from resp.Config or nil.
func pairByName(resp *Response, name string) *synccmd.Pair {
	if resp == nil || resp.Config == nil {
		return nil
	}
	for i := range resp.Config.Pairs {
		if resp.Config.Pairs[i].Name == name {
			p := resp.Config.Pairs[i]
			return &p
		}
	}
	return nil
}

// errText prefers RunErr then PreErr.
func errText(resp *Response) string {
	if resp == nil {
		return ""
	}
	if resp.RunErr != "" {
		return resp.RunErr
	}
	return resp.PreErr
}

// defaultIgnoreList is the documented init default ignore set.
func defaultIgnoreList() []string {
	return []string{
		"Name .DS_Store",
		"Name node_modules",
		"Name *.log",
		"Path tmp",
		"Path log",
	}
}
```
