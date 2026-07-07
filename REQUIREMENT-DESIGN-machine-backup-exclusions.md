# Machine Backup Exclusion Enhancements

## Summary

Extend machine backup built-in exclusions with:

1. **Six new path-prefix exclusions** in `builtinExclusionEntries`
2. **Three new special rules** (like `**/node_modules`): `**/upload-chunks`, `**/*.log`, `**(binary)`
3. **`IsExecutableBinary`** in `dot-pkgs-with-critic/go-pkgs/file/detect/` (ELF/Mach-O/PE only)
4. **Per-file `--include` overrides** for log suffix and binary rules
5. **Config version bump** `1.0` → `1.1`

Deferred (not in this change): `.cursor/chats`, `.grok/sessions`, `.codex/sessions`.

## Data Model

### Exclusion config (`.backup/config.json`)

```json
{
  "version": "1.1",
  "exclude_paths": [
    {"path": "**(binary)", "reason": "executable binaries (reinstallable)"},
    {"path": "**/*.log", "reason": "log files"},
    {"path": "**/node_modules", "reason": "node_modules directories"},
    {"path": "**/upload-chunks", "reason": "incomplete upload temp state"},
    ...existing 12 entries...,
    ...6 new path entries...
  ]
}
```

### ExclusionRules evaluation order (walk)

1. `includedPaths` exact match → **include**
2. Path prefix / full-tree rule → exclude
3. `**/node_modules` segment → exclude
4. `**/upload-chunks` segment → exclude
5. `**/*.log` suffix (basename ends `.log`) → exclude
6. `IsExecutableBinary` → exclude
7. else → include

`--include PATH` adds exact paths to `includedPaths` (works for re-including a
specific `.log` file or executable binary even when tree exclusions don't apply).

### New builtin path entries

| Path | Reason |
|------|--------|
| `.local/share/cursor-agent/versions` | Cursor agent version cache |
| `.opencode/bin` | OpenCode binary (reinstallable) |
| `.config/confluence-fetch-skill/data` | confluence-fetch-skill data cache |
| `.codex/.tmp` | Codex temporary plugin cache |
| `.local/share/opencode/repos` | OpenCode repo clone cache |
| `.local/share/opencode/log` | OpenCode application logs |

### Explicitly NOT excluded

- `.live-and-love/imgs/` — user images kept
- `.local/share/opencode/opencode.db` — SQLite state DB kept (not executable binary)
- Session dirs (deferred)

### IsExecutableBinary

New function in `dot-pkgs-with-critic/go-pkgs/file/detect/`:

- Returns true only for ELF, Mach-O, PE executables
- Does NOT treat images, SQLite, archives, fonts as executable
- Reuse existing magic-detection helpers from `detect.go`

## Test Strategy

Extend existing doctest tree `tests/remote-agent-machine-backup` with new leaves
under `backup/` grouping. Harness seeds fixtures in `serverHome` and asserts via
`machine backup --dry-run` (stream output) and/or `machine backup` (archive members).

Also add unit tests in `server/machinebackup/` (implementer may add; not doctest
tree) for exclusion rule logic.

### New doctest leaves

```
backup/
  extended-exclusions/     (LEAF) show-config lists new rules + path entries (version 1.1)
  upload-chunks/           (LEAF) **/upload-chunks segment excluded; EXCLUDED lists rule
  log-suffix/              (LEAF) *.log excluded by default; listed in EXCLUDED
  include-log/             (LEAF) --include re-includes specific .log file
  binary-exclude/          (LEAF) ELF binary excluded; **(binary) in EXCLUDED
  include-binary/          (LEAF) --include re-includes specific executable
  keep-sqlite/             (LEAF) opencode.db (SQLite magic) remains included
  keep-images/             (LEAF) .live-and-love/imgs/*.jpg remains included
  path-exclusions/         (LEAF) new path prefixes exclude seeded trees from archive
```

### Fixture guidance (designer seeds in SETUP)

- Minimal ELF stub (small valid ELF header bytes) at e.g. `.ai-critic/bin/stub`
- Text config at `.ai-critic/config.json`
- Log at `.ai-critic/service.log` (excluded by default)
- Named log for include test: `.ai-critic/keep.log` with `--include .ai-critic/keep.log`
- `upload-chunks/` nested: `.live-and-love/upload-chunks/chunk-1`
- SQLite header file: `.local/share/opencode/opencode.db` with `SQLite format 3\0` prefix
- Tiny JPEG bytes in `.live-and-love/imgs/photo.jpg`
- Path exclusion trees: `.codex/.tmp/junk`, `.local/share/opencode/repos/foo`, etc.

### Expected outputs

- `--show-config` / EXCLUDED section lists all special rules with reasons
- Dry-run DOT DIRS omits excluded paths; EXCLUDED section mentions rules
- Archive tar members exclude matching paths/files
- `--include` leaves re-included paths in DOT FILES/DIRS and archive
- User-facing stdout ends with `\n` after last content line

### Update existing leaves

- `backup/dry-run` — may need to tolerate additional EXCLUDED entries (still lists `.cache`, `.npm`)
- `backup/show-config` — version `1.1`, more exclude_paths entries
- `restore/show-config-builtin` — version `1.1`

## Verification

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
go test ./dot-pkgs-with-critic/go-pkgs/file/detect/... -count=1
```

## Approved

User approved via `/doctest-tdd go ahead` after followup clarification sessions.