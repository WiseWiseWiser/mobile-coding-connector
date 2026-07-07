# Machine Backup: Large Dir Summary + Exclusion Revert

## Summary

1. **Remove** built-in exclusions: `.config/git-fetch-skill/data`,
   `.config/confluence-fetch-skill/data`, `.knowledge-index` (config version stays **1.1**)
2. **Summary phase** enhancements for `machine backup --dry-run`:
   - Sort DOT FILES and DOT DIRS by size descending (path tiebreak)
   - Flag included dirs with `Bytes > threshold` as `LARGE SIZE`
   - Default threshold **40 MB**; CLI `--large-dir-threshold SIZE` (50M, 50MB, 1G, 1GB, etc.)
   - `LARGE DIR DETAIL` blocks for each large included dir (analyse-files style via `file/analyse`)
3. **CLI**: red ANSI for `LARGE SIZE` when stdout is a TTY; plain when redirected
4. **Invariant**: real `machine backup` archives exactly what `--dry-run` reports as included
   (same walk/exclusions; `.backup/` meta still injected at pack time)

## Data Model

```go
// BackupStreamRequest / BackupRequest gain:
LargeDirThresholdBytes int64 `json:"large_dir_threshold_bytes,omitempty"` // 0 = default 40MB

const defaultLargeDirThresholdBytes = 40 * 1024 * 1024
```

`ExcludePathEntry` unchanged. Remove 3 entries from `builtinExclusionEntries` only.

Summary options passed to `formatBackupDryRunSummary(plan, opts)`:
- `LargeDirThresholdBytes int64`

Large dir detail: call `analyse` package on `filepath.Join(home, dirStat.Path)` for dirs over threshold;
emit `FormatEntryBlock`-style text under `LARGE DIR DETAIL:`.

## CLI

```
remote-agent machine backup [--dry-run] [--large-dir-threshold SIZE] ...
```

Parse SIZE case-insensitively: B, K/KB, M/MB, G/GB. Invalid → exit 1 with message.

## Output Format (summary verbatim logs)

```
  DOT DIRS (N dirs, F files, SIZE)
    DIR                     FILES     SIZE
    .big-dir                  100  50.00 MB  LARGE SIZE
    .small-dir                 10   1.00 KB

  LARGE DIR DETAIL:
  > .big-dir
    > child-a                    30.00 MB
    > child-b                    20.00 MB
    sessions  2 sessions         5.00 MB   ← semantic when enricher exists

  EXCLUDED (...)
  TOTAL: ...
```

Stream phase unchanged (no LARGE DIR DETAIL in stream; only summary).

User-facing stdout ends with `\n`.

## Test Strategy

Extend `tests/remote-agent-machine-backup`:

### New leaves

| Leaf | Purpose |
|------|---------|
| `backup/large-dir-summary` | Seed dir >40MB; dry-run shows LARGE SIZE, detail block, size-desc sort |
| `backup/large-dir-threshold` | `--large-dir-threshold 100MB` suppresses flag on 50MB dir |
| `backup/dry-run-matches-archive` | Same flags: dry-run included set == tar members (minus .backup meta) |
| `backup/included-fetch-skills` | After removal: git-fetch/confluence/knowledge-index paths in plan |

### Updates

| Leaf | Change |
|------|--------|
| `backup/extended-exclusions` | Remove 3 paths from expected list |
| `backup/path-exclusions` | confluence path now **included** not excluded |
| `backup/dry-run` | tolerate size-sorted DOT DIRS |

### Fixtures

- `seedLargeDirFixture`: `.big-test/` with padded files totaling >40MB
- `seedIncludedFetchSkills`: small files under removed-exclusion paths
- Threshold tests use 50MB dir + 100MB flag

### Unit tests (implementer)

- `ParseSizeFlag` / `ParseHumanSize`
- `formatBackupDryRunSummary` sort + LARGE SIZE + detail
- Exclusion list no longer has 3 paths

## Verification

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
go test ./cmd/agentcli/... -count=1  # if parser in shared pkg
```

## Approved

User `/doctest-tdd` after followup locked: no version bump, --large-dir-threshold, dry-run ≡ backup.