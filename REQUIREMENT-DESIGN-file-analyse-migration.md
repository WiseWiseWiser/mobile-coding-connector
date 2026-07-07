# REQUIREMENT-DESIGN: Migrate machineanalyse core → dot-pkgs/file/analyse

## Status

Approved — user said "go ahead" with OnEntry streaming (2026-07-06).

## Goal

Extract reusable home-directory analysis from `ai-critic/server/machineanalyse`
into `github.com/xhd2015/dot-pkgs/go-pkgs/file/analyse` (package name `analyse`).

`OnEntry` callback must be supported for streaming callers (emit each entry as
scan completes). ai-critic `server/machineanalyse` becomes thin HTTP/SSE adapter.

## Data model (file/analyse)

Move unchanged semantics from `server/machineanalyse/types.go`:

- `EntryKind`, `ChildLine`, `SemanticLine`, `Aggregates`, `EntryResult`, `ScanSummary`, `LargestEntry`

New API:

```go
package analyse

type Options struct {
    Home string
    // OnEntry is called after each top-level HOME child is fully scanned.
    // Return non-nil error to abort scan (propagated from Scan).
    OnEntry func(EntryResult) error
}

func Scan(ctx context.Context, opts Options) ([]EntryResult, ScanSummary, error)
func FormatEntryBlock(entry EntryResult) string
func FormatSummaryLines(summary ScanSummary) []string
func FormatSize(n int64) string
```

No persistent storage.

## Code to migrate (from server/machineanalyse)

| Source | Target |
|--------|--------|
| types.go | file/analyse/types.go |
| size.go | file/analyse/size.go |
| scan.go | file/analyse/scan.go (ScanHome→Scan, invoke OnEntry) |
| enrichers.go | file/analyse/enrichers.go |
| format.go | file/analyse/format.go |
| format_test.go | file/analyse/format_test.go |

## Stays in ai-critic (server/machineanalyse)

- api.go — route registration
- stream.go — progress.Writer; calls analyse.Scan with OnEntry→EmitLog

Delete migrated files after rewire.

## Dependencies (dot-pkgs)

- `git/scan_repo` — per-entry git aggregates (already in module)
- `file/detect` — optional for binary line detection (prefer reuse over duplicate)

## OnEntry streaming contract

1. Scan walks sorted top-level HOME children alphabetically.
2. After each entry is fully built, call `OnEntry(result)` if non-nil.
3. If OnEntry returns error, Scan aborts and returns that error.
4. Batch callers pass `OnEntry: nil`; still receive full `[]EntryResult` at end.

## Output format (unchanged)

Per entry block order: `>` children → semantic → aggregates.
Files: size + lines (or `lines (binary)`).
Summary: topic-present rule for tool lines.

## Test tree location

`dot-pkgs-with-critic/go-pkgs/file/analyse/tests/`

Mirror `git/scan_repo/tests/` harness style: temp fake HOME, direct `analyse.Scan`.

### Minimum leaves

| Leaf | Proves |
|------|--------|
| `scan/basic-dir` | Children sorted; deep sizes; dir entry kind |
| `scan/file-lines` | Text lines count; binary `lines (binary)` |
| `scan/codex-semantic` | Rollout sessions, top-level skill dirs, plugins 0 |
| `scan/git-dirs` | Aggregates.GitRepos == 1 with git init fixture |
| `scan/node-modules` | Child node_modules + NodeModulesDirs recursive count |
| `scan/entry-order` | Results alphabetical by entry name |
| `scan/on-entry` | OnEntry called once per entry in sorted order; abort on error |
| `format/entry-block` | Children before semantic before aggregates |
| `format/summary-topic` | Codex summary when HasCodex; grok omitted when absent |

### Unit tests

Move existing `format_test.go` tests to `file/analyse/format_test.go`.

## ai-critic regression (sealed, do not modify)

`tests/remote-agent-machine-analyse-files/` — all 7 stream leaves must stay GREEN after rewire.

## Verify commands

```sh
# dot-pkgs
cd dot-pkgs-with-critic/go-pkgs
go test ./file/analyse/... -count=1
doctest vet ./file/analyse/tests
doctest test -v ./file/analyse/tests/...

# ai-critic
doctest test -v ./tests/remote-agent-machine-analyse-files/...
```