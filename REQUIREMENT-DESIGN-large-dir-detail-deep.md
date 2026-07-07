# REQUIREMENT-DESIGN: LARGE DIR DETAIL deep scan

## Problem

`backup --dry-run` summary `LARGE DIR DETAIL` only lists **immediate children** of each
large top-level dot-dir (via `analyse.ScanDirEntry` → `immediateChildSizes`). Nested
directories ≥10 MB are not listed unless they are direct children of a LARGE SIZE dir.

## New behavior

### LARGE DIR DETAIL section

Recursively scan **all included (non-excluded) directories** under backup scope and emit a
**flat** list of every directory whose total size ≥ **10 MB**.

| Rule | Value |
|------|-------|
| Detail threshold | **10 MB** fixed (`10 * 1024 * 1024`) |
| Scope | Included dot-dirs/files tree only; skip excluded paths per `ExclusionRules` |
| Output | Flat lines: one row per qualifying dir (parent and child may both appear) |
| Sort | Size descending; path ascending tiebreak |
| Format | `  > <rel-path>  <human-size>` (align with analyse child line style) |
| Location | Summary only (after DOT DIRS, before EXCLUDED); stream unchanged |

### Unchanged

- `LARGE SIZE` flag on DOT DIRS rows still uses `--large-dir-threshold` (default 40 MB)
- `LARGE DIR DETAIL` appears when at least one included dir ≥ 10 MB exists (not only when LARGE SIZE dirs exist)
- Excluded trees (e.g. `.cache`) must not appear in detail scan

### Example output

```
  LARGE DIR DETAIL:
  > .big-test/child-a                  30.00 MB
  > .big-test                          50.00 MB
  > .big-test/child-b                  20.00 MB
  > .deep-test/nested-big              12.00 MB
```

(sorted by size desc; `.big-test` parent and children all listed)

## Implementation sketch (for implementer)

- New `collectLargeIncludedDirs(home, rules, minBytes)` walking included backup tree
- Replace or supplement `formatLargeDirDetailBlocks` to use flat sorted list
- Reuse `formatSize` / `analyse.FormatSize` for human sizes
- Unit test in `stream_summary_test.go`

## Scenarios to test

| Leaf | Description |
|------|-------------|
| `backup/large-dir-detail-deep` | **New**: nested `.deep-test/nested-big/` (12MB) + small sibling; detail lists nested path; excluded `.cache` absent |
| `backup/large-dir-summary` | **Update**: still expects `.big-test` children in detail but flat sorted format; parent `.big-test` also listed |

### Fixture (`SeedLargeDirDetailDeep`)

- Keep existing `SeedLargeDir` (`.big-test` 50MB)
- Add `.deep-test/nested-big/file` 12MB
- Add `.deep-test/small/tiny` 1KB
- Ensure `.cache` exists in default seed (builtin excluded) — must NOT appear in detail

## Verify

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
```