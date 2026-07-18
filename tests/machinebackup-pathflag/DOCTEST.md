# machinebackup → pathflag integration contract

Classic TDD for wiring `server/machinebackup` exclusions to
`github.com/xhd2015/bak-files/pathflag` as the catalog SSoT. Snapshot today uses
a hand-maintained `builtinExclusionEntries` table and does **not** import
pathflag or require bak-files in `go.mod`. These doctests lock the
**post-refactor** contract: module dependency, package import, catalog skip via
public exclusion APIs, policy overrides, and BuiltinExclusionConfig alignment.

Out of scope: bak-files CLI, archive format, dry-run Owner UX, real upstream
ai-critic (snapshot only).

Module: `github.com/xhd2015/ai-critic`  
Package under change: `server/machinebackup`  
External catalog: `github.com/xhd2015/bak-files/pathflag`

# DSN (Domain Specific Notion)

**machinebackup exclusions** decide which home-relative paths are skipped when
building a machine backup plan. After the pathflag refactor, catalog matching is
driven by pathflag attributes; user/CLI include/exclude and binary content
detection remain policy layers on top.

**Participants**

| Participant | Role |
|-------------|------|
| Caller / doctest harness | Builds a `Request` with Op, relative path, and optional CLI include/exclude |
| MergeExclusions | Merges builtin catalog (+ user config) with CLI `--exclude` / `--include` into `ExclusionRules` |
| ExclusionRules | Public `IsExcluded(rel)` and `ReasonFor(rel)` for path-level policy |
| BuiltinExclusionConfig | Enumerates default exclude path rules + reasons for config JSON / show-config |
| pathflag.Classify | Pure catalog classifier (Rule, Reason, Flags, Owner); skip when Flags ∩ DefaultSkipMask ≠ 0 |
| go.mod | Declares `require github.com/xhd2015/bak-files` and `replace => ../..` from this snapshot |
| machinebackup package | Must import pathflag after refactor so Classify drives catalog skip |

**Behaviors**

1. **Module dependency** — ai-critic `go.mod` requires bak-files and replaces it
   to the parent monorepo root (`../..` from this snapshot).
2. **Package import** — `github.com/xhd2015/ai-critic/server/machinebackup`
   imports `github.com/xhd2015/bak-files/pathflag`.
3. **Catalog skip** — with empty user/CLI overrides, representative paths that
   pathflag would mark with DefaultSkipMask attributes are `IsExcluded`;
   ordinary config/dotfiles are not.
4. **Policy overrides** — CLI include removes a builtin exclude; CLI exclude
   adds a custom path; include of a specific `.log` keeps that file while other
   logs still skip.
5. **Builtin SSoT** — `BuiltinExclusionConfig` lists pathflag catalog rules
   (plus synthetic `**(binary)`), with reasons aligned to pathflag reasons for
   shared path rules.

## Version

0.0.2

## Decision Tree

```
tests/machinebackup-pathflag/              [Request{Op, RelPath, Exclude, Include, RulePath}]
│                                          Run: machinebackup public API + go.mod / import inspect
├── module/                                # dependency + import contract (RED today)
│   ├── go-mod-bak-files/                  # require + replace bak-files
│   └── imports-pathflag/                  # package Imports contain pathflag
├── catalog-skip/                          # MergeExclusions(nil,nil,nil) path outcomes
│   ├── cache/                             # .cache/x excluded (.cache)
│   ├── codex-tmp/                         # .codex/.tmp/plug excluded
│   ├── codex-config-included/             # .codex/config.toml not excluded
│   ├── node-modules/                      # foo/node_modules/x excluded
│   ├── upload-chunks/                     # a/upload-chunks/1 excluded
│   ├── log-suffix/                        # .ai-critic/service.log excluded (**/*.log)
│   └── bashrc-included/                   # .bashrc not excluded
├── policy/                                # include / exclude overrides
│   ├── include-cache/                     # --include .cache → not excluded
│   ├── exclude-docker/                    # --exclude .docker → excluded
│   └── include-keep-log/                  # keep.log included; other .log still skipped
└── ssot-builtin/                          # BuiltinExclusionConfig catalog
    ├── has-catalog-paths/                 # pathflag rules + **(binary) present
    └── reasons-match-pathflag/            # shared path rules share pathflag reasons
```

**Significance order:** integration contract (module/import) → pure catalog skip
→ policy overrides → config SSoT listing.

## Test Index

| # | Leaf | Description | Expected pre-impl |
|---|------|-------------|-------------------|
| 1 | `module/go-mod-bak-files` | go.mod require + replace bak-files | **RED** |
| 2 | `module/imports-pathflag` | machinebackup imports pathflag | **RED** |
| 3 | `catalog-skip/cache` | `.cache/x` excluded | GREEN likely |
| 4 | `catalog-skip/codex-tmp` | `.codex/.tmp/plug` excluded | GREEN likely |
| 5 | `catalog-skip/codex-config-included` | `.codex/config.toml` included | GREEN likely |
| 6 | `catalog-skip/node-modules` | nested `node_modules` excluded | GREEN likely |
| 7 | `catalog-skip/upload-chunks` | nested `upload-chunks` excluded | GREEN likely |
| 8 | `catalog-skip/log-suffix` | `*.log` excluded via public IsExcluded | **RED** (path-only API today) |
| 9 | `catalog-skip/bashrc-included` | `.bashrc` included | GREEN likely |
| 10 | `policy/include-cache` | include `.cache` clears skip | GREEN likely |
| 11 | `policy/exclude-docker` | exclude `.docker` adds skip | GREEN likely |
| 12 | `policy/include-keep-log` | keep.log in; service.log out | **partial RED** (log via IsExcluded) |
| 13 | `ssot-builtin/has-catalog-paths` | catalog paths + `**(binary)` listed | GREEN likely |
| 14 | `ssot-builtin/reasons-match-pathflag` | reasons match pathflag catalog | GREEN likely |

## How to Run

From the ai-critic snapshot module root:

```bash
doctest vet ./tests/machinebackup-pathflag
doctest test ./tests/machinebackup-pathflag
```

Single leaf:

```bash
doctest test ./tests/machinebackup-pathflag/module/go-mod-bak-files
```

```go
import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/machinebackup"
)

// Op values for Request.Op.
const (
	OpIsExcluded       = "is_excluded"
	OpReason           = "reason"
	OpBuiltinHasPath   = "builtin_has_path"
	OpIncludeOverride  = "include_override"
	OpModuleGoMod      = "module_go_mod"
	OpPackageImports   = "package_imports"
	OpSSOTCatalogPaths = "ssot_catalog_paths"
	OpSSOTReasons      = "ssot_reasons"
)

// pathflagModule is the expected bak-files module path.
const pathflagModule = "github.com/xhd2015/bak-files"

// pathflagImport is the expected import path used by machinebackup.
const pathflagImport = "github.com/xhd2015/bak-files/pathflag"

// machinebackupPkg is the package under change.
const machinebackupPkg = "github.com/xhd2015/ai-critic/server/machinebackup"

// Request drives exclusion, builtin config, or module/import inspection.
// Setup fills Op and path/override fields; Run calls machinebackup public API
// or inspects go.mod / package source (no production code changes).
type Request struct {
	// Op selects the harness path (see Op* constants). Default: is_excluded.
	Op string
	// RelPath is the home-relative path for exclusion checks.
	RelPath string
	// Exclude is CLI --exclude paths passed to MergeExclusions.
	Exclude []string
	// Include is CLI --include paths passed to MergeExclusions.
	Include []string
	// RulePath is the exclude_paths entry path for builtin_has_path.
	RulePath string
	// WantExcluded is the expected IsExcluded outcome when set via Setup.
	// Zero value false is meaningful; use WantExcludedSet.
	WantExcluded    bool
	WantExcludedSet bool
	// WantReason is optional exact ReasonFor / builtin reason expectation.
	WantReason string
	// SecondaryRelPath is used by include-keep-log to check a second path.
	SecondaryRelPath string
	// WantSecondaryExcluded is expectation for SecondaryRelPath when set.
	WantSecondaryExcluded    bool
	WantSecondaryExcludedSet bool
}

// Response is assert-friendly output from Run.
type Response struct {
	Excluded          bool
	Reason            string
	SecondaryExcluded bool
	SecondaryReason   string
	HasPath           bool
	BuiltinReason     string
	// ModuleRequire is true when go.mod requires bak-files.
	ModuleRequire bool
	// ModuleReplace is true when go.mod replace points bak-files at ../..
	ModuleReplace bool
	// ImportsPathflag is true when machinebackup lists pathflag in Imports.
	ImportsPathflag bool
	// MissingPaths lists BuiltinExclusionConfig paths still absent (ssot).
	MissingPaths []string
	// ReasonMismatches lists "path: got vs want" for reason SSoT failures.
	ReasonMismatches []string
	// Detail is free-form diagnostic text for asserts / debugging.
	Detail string
	// Err is a non-fatal harness error message (empty on success).
	Err string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	op := req.Op
	if op == "" {
		op = OpIsExcluded
	}
	resp := &Response{}
	root := moduleRoot()

	switch op {
	case OpIsExcluded, OpReason, OpIncludeOverride:
		rules := machinebackup.MergeExclusions(nil, req.Exclude, req.Include)
		rel := req.RelPath
		resp.Excluded = rules.IsExcluded(rel)
		resp.Reason = rules.ReasonFor(rel)
		if req.SecondaryRelPath != "" {
			resp.SecondaryExcluded = rules.IsExcluded(req.SecondaryRelPath)
			resp.SecondaryReason = rules.ReasonFor(req.SecondaryRelPath)
		}
		return resp, nil

	case OpBuiltinHasPath:
		path := req.RulePath
		if path == "" {
			path = req.RelPath
		}
		cfg := machinebackup.BuiltinExclusionConfig()
		for _, e := range cfg.ExcludePaths {
			if e.Path == path {
				resp.HasPath = true
				resp.BuiltinReason = e.Reason
				break
			}
		}
		return resp, nil

	case OpModuleGoMod:
		requireOK, replaceOK, detail, err := inspectGoMod(root)
		if err != nil {
			return &Response{Err: err.Error()}, nil
		}
		resp.ModuleRequire = requireOK
		resp.ModuleReplace = replaceOK
		resp.Detail = detail
		return resp, nil

	case OpPackageImports:
		ok, detail, err := packageImportsPathflag(root)
		if err != nil {
			return &Response{Err: err.Error()}, nil
		}
		resp.ImportsPathflag = ok
		resp.Detail = detail
		return resp, nil

	case OpSSOTCatalogPaths:
		missing := missingBuiltinCatalogPaths()
		resp.MissingPaths = missing
		resp.HasPath = len(missing) == 0
		return resp, nil

	case OpSSOTReasons:
		mismatches := mismatchedBuiltinReasons()
		resp.ReasonMismatches = mismatches
		resp.HasPath = len(mismatches) == 0
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown Op %q", op)
	}
}

// moduleRoot returns the ai-critic module root (directory with go.mod).
// DOCTEST_ROOT is tests/machinebackup-pathflag → two levels up.
func moduleRoot() string {
	root, err := filepath.Abs(filepath.Join(DOCTEST_ROOT, "../.."))
	if err != nil {
		return filepath.Join(DOCTEST_ROOT, "../..")
	}
	return root
}

func inspectGoMod(root string) (requireOK, replaceOK bool, detail string, err error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false, false, "", err
	}
	text := string(data)
	// require may be single-line or in a require ( ) block.
	requireOK = goModHasRequire(text, pathflagModule)
	replaceOK = goModHasReplace(text, pathflagModule, "../..")
	var parts []string
	if requireOK {
		parts = append(parts, "require=ok")
	} else {
		parts = append(parts, "require=missing")
	}
	if replaceOK {
		parts = append(parts, "replace=ok")
	} else {
		parts = append(parts, "replace=missing")
	}
	return requireOK, replaceOK, strings.Join(parts, " "), nil
}

func goModHasRequire(text, mod string) bool {
	sc := bufio.NewScanner(strings.NewReader(text))
	inBlock := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		if line == "require (" {
			inBlock = true
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if strings.HasPrefix(line, mod+" ") || line == mod {
				return true
			}
			continue
		}
		if strings.HasPrefix(line, "require "+mod+" ") || line == "require "+mod {
			return true
		}
	}
	return false
}

func goModHasReplace(text, mod, wantDir string) bool {
	// Accept replace mod => ../.. or replace mod => ../.. // comment
	// also block form.
	wantDir = filepath.ToSlash(wantDir)
	sc := bufio.NewScanner(strings.NewReader(text))
	inBlock := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "replace (" {
			inBlock = true
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if replaceLineMatches(line, mod, wantDir) {
				return true
			}
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "replace "))
			if replaceLineMatches(rest, mod, wantDir) {
				return true
			}
		}
	}
	return false
}

func replaceLineMatches(line, mod, wantDir string) bool {
	// forms: mod => path | mod vX => path
	if !strings.Contains(line, mod) || !strings.Contains(line, "=>") {
		return false
	}
	parts := strings.SplitN(line, "=>", 2)
	if len(parts) != 2 {
		return false
	}
	left := strings.TrimSpace(parts[0])
	right := filepath.ToSlash(strings.TrimSpace(parts[1]))
	if !strings.HasPrefix(left, mod) {
		return false
	}
	return right == wantDir || strings.HasSuffix(right, "/"+strings.TrimPrefix(wantDir, "./"))
}

// packageImportsPathflag reports whether machinebackup imports pathflag.
// Prefers `go list`; falls back to parsing .go files under server/machinebackup.
func packageImportsPathflag(root string) (bool, string, error) {
	cmd := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, machinebackupPkg)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == pathflagImport {
				return true, "go list: pathflag import present", nil
			}
		}
		return false, "go list: pathflag import absent\n" + string(out), nil
	}
	// Fallback: parse source (e.g. if go list fails for unrelated reasons).
	dir := filepath.Join(root, "server", "machinebackup")
	entries, rdErr := os.ReadDir(dir)
	if rdErr != nil {
		return false, "", fmt.Errorf("go list: %v; read dir: %w", err, rdErr)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if perr != nil {
			continue
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if p == pathflagImport {
				return true, "source parse: pathflag import in " + name, nil
			}
		}
	}
	return false, fmt.Sprintf("go list failed (%v); source parse: pathflag import absent", err), nil
}

// pathflagCatalogPaths are non-segment path rules from pathflag attributeRules
// plus segment/suffix specials that BuiltinExclusionConfig must enumerate.
// Synthetic **(binary) is product-specific (content detect), not a Classify rule.
func pathflagCatalogPaths() []string {
	return []string{
		".bun",
		".grok/downloads",
		".grok/marketplace-cache",
		".grok/vendor",
		".grok/logs",
		".config/chromium",
		".cache",
		".npm",
		".cargo/registry",
		".codex/.tmp",
		".codex/skills/.system",
		".opencode/bin",
		".local/share/cursor-agent/versions",
		".local/share/opencode/repos",
		".local/share/opencode/snapshot",
		".local/share/opencode/log",
		".Trash",
		".local/share/Trash",
		".backup",
		"**/node_modules",
		"**/upload-chunks",
		"**/*.log",
		"**(binary)",
	}
}

// pathflagPathReasons maps catalog path rules to pathflag.Classify Reason strings
// (and the product binary synthetic reason). Used for SSoT reason alignment.
func pathflagPathReasons() map[string]string {
	return map[string]string{
		".bun":                               "Bun install cache",
		".grok/downloads":                    "Grok downloads cache",
		".grok/marketplace-cache":            "Grok plugin marketplace git cache",
		".grok/vendor":                       "Grok vendored dependencies cache",
		".grok/logs":                         "Grok application logs",
		".config/chromium":                   "Chromium profile cache",
		".cache":                             "temporary application cache",
		".npm":                               "npm cache",
		".cargo/registry":                    "Cargo registry cache",
		".codex/.tmp":                        "Codex temporary plugin cache",
		".codex/skills/.system":              "Codex system skills cache",
		".opencode/bin":                      "OpenCode binary (reinstallable)",
		".local/share/cursor-agent/versions": "Cursor agent version cache",
		".local/share/opencode/repos":        "OpenCode repo clone cache",
		".local/share/opencode/snapshot":     "OpenCode snapshot cache",
		".local/share/opencode/log":          "OpenCode application logs",
		".Trash":                             "macOS trash",
		".local/share/Trash":                 "Linux trash",
		".backup":                            "machine backup metadata (injected at pack time)",
		"**/node_modules":                    "node_modules directories",
		"**/upload-chunks":                   "incomplete upload temp state",
		"**/*.log":                           "log files",
		"**(binary)":                         "executable binaries (reinstallable)",
	}
}

func missingBuiltinCatalogPaths() []string {
	cfg := machinebackup.BuiltinExclusionConfig()
	have := make(map[string]bool, len(cfg.ExcludePaths))
	for _, e := range cfg.ExcludePaths {
		have[e.Path] = true
	}
	var missing []string
	for _, p := range pathflagCatalogPaths() {
		if !have[p] {
			missing = append(missing, p)
		}
	}
	return missing
}

func mismatchedBuiltinReasons() []string {
	cfg := machinebackup.BuiltinExclusionConfig()
	byPath := make(map[string]string, len(cfg.ExcludePaths))
	for _, e := range cfg.ExcludePaths {
		byPath[e.Path] = e.Reason
	}
	var mismatches []string
	for path, want := range pathflagPathReasons() {
		got, ok := byPath[path]
		if !ok {
			mismatches = append(mismatches, path+": missing (want reason "+want+")")
			continue
		}
		if got != want {
			mismatches = append(mismatches, fmt.Sprintf("%s: got %q want %q", path, got, want))
		}
	}
	return mismatches
}
```
