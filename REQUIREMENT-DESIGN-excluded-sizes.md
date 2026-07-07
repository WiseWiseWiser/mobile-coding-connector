# Machine Backup EXCLUDED Section Sizes

## Summary

Enhance `remote-agent machine backup --dry-run` stream and summary output so the
EXCLUDED section shows:

1. Header with total paths, files, and bytes skipped
2. Per-rule table: RULE, FILES, SIZE, REASON
3. Sorted by SIZE descending (ties: path ascending)
4. Stream phase: one progress line per rule (not per file)

User approved via followup: FILES column yes, sort by size yes, one line per rule yes.

## Data Model

Extend `ExcludePathEntry` in `server/machinebackup/exclusions.go`:

```go
type ExcludePathEntry struct {
    Path   string `json:"path"`
    Reason string `json:"reason"`
    Files  int    `json:"files"`  // regular files skipped under this rule
    Bytes  int64  `json:"bytes"`  // aggregate bytes skipped
}
```

During walk/discover, when a path is skipped, attribute to the **first matching
rule** (same evaluation order as `shouldSkipPath`):

1. includedPaths → not skipped
2. path prefix / full-tree
3. **/node_modules segment
4. **/upload-chunks segment
5. **/*.log suffix
6. IsExecutableBinary → **(binary)

No double-counting. Symlinks skipped do not increment `Files` (dirs not counted).

Aggregate into `plan.Excluded` with stats populated; sort by `Bytes` desc, `Path` asc
before emit.

## Output Format

### Stream section header (SSE `section` frame)

```
EXCLUDED
```

### Stream progress lines (one per rule, `excluded` layer)

CLI prints:
```
  RULE                                    FILES       SIZE   REASON
  .cache                                      2      1.50 KB  temporary application cache
  **/*.log                                    1        512 B  log files
```

Or without header row in stream (header row only in summary)? **Include column
header row once after `EXCLUDED:` section in stream phase**, then one line per rule.

### Summary block (`log` frames, verbatim)

```
  EXCLUDED (N paths, F files, X.XX MB)
    RULE                                    FILES       SIZE   REASON
    .cache                                      2      1.50 KB  temporary application cache
    **/*.log                                    1        512 B  log files
    ...
```

Header totals must equal sum of per-rule Files and Bytes.

Use existing `formatSize()` for SIZE column.

User-facing stdout ends with `\n` after last content line.

## Test Strategy

Extend `tests/remote-agent-machine-backup`:

### New leaf: `backup/excluded-sizes`

Seed in serverHome (via SETUP helpers):

- `.cache/junk` (1024 B) — excluded by `.cache` prefix
- `.cache/nested/deep` (512 B) — same rule
- `.ai-critic/service.log` (512 B) — excluded by `**/*.log`
- `.npm/x/package.json` — under `.npm` tree (excluded)
- `.bashrc` — included (control)

Assert on `machine backup --dry-run`:

1. Exit 0
2. EXCLUDED header matches `EXCLUDED \(\d+ paths, \d+ files,` with size token
3. `.cache` row shows FILES >= 2 and SIZE >= 1 KB
4. `**/*.log` row shows FILES >= 1
5. Rows appear in descending SIZE order (`.cache` before `**/*.log` when cache larger)
6. `.cache/junk` not in included DOT FILES
7. Column header `RULE` and `FILES` present in EXCLUDED section

### Update: `backup/dry-run`

- Tolerate new EXCLUDED header format with paths/files/size totals
- Still requires `.cache` reason token

### Unit tests (implementer adds in `server/machinebackup/`)

- `TestExcludedStatsAttribution` — prefix beats log suffix
- `TestExcludedStatsSort` — bytes descending
- `TestFormatExcludedSection` — header totals match sum

## Verification

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/backup/excluded-sizes
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
```

## Approved

User: `/doctest-tdd go ahead` after followup locked FILES, size-desc sort, one line per rule.