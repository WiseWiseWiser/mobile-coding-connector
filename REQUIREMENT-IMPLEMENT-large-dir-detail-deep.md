# REQUIREMENT-IMPLEMENT: LARGE DIR DETAIL deep scan

## Context

LARGE DIR DETAIL only shows level-1 children via analyse.ScanDirEntry. Need recursive
flat scan of all included dirs ≥10MB, size-desc sorted.

Design: `REQUIREMENT-DESIGN-large-dir-detail-deep.md`

## Tests sealed — do not modify

`./tests/remote-agent-machine-backup/`

## Implementation

1. Constant `largeDirDetailMinBytes = 10 * 1024 * 1024`
2. `collectLargeIncludedDirs(home string, rules ExclusionRules, minBytes int64) []dirSizeEntry`
   - Walk included backup tree (dot entries under home)
   - Skip excluded paths per rules
   - Compute total size per directory
   - Collect dirs where size >= minBytes
3. `formatLargeDirDetailFlat(entries []dirSizeEntry) []string` — `  > path  size` lines sorted desc
4. Replace `formatLargeDirDetailBlocks` usage in `formatBackupDryRunSummary`:
   - LARGE DIR DETAIL when any dir ≥10MB (not only largeDirs from DOT DIRS threshold)
   - Detail threshold independent of `--large-dir-threshold`
5. Unit tests in `stream_summary_test.go`

## RED failures

- large-dir-summary: expects flat rows including `.big-test` parent path
- large-dir-detail-deep: expects `.deep-test/nested-big` nested path
- large-dir-threshold / persisted-threshold: detail still shows ≥10MB dirs even when LARGE SIZE suppressed

## Verify

```sh
go test ./server/machinebackup/... -count=1
doctest test ./tests/remote-agent-machine-backup/...
```