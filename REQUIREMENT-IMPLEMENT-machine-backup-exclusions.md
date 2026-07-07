# Implement: Machine Backup Exclusion Enhancements

## Context

Tests are **sealed** — do not modify `./tests/remote-agent-machine-backup` or
`REQUIREMENT-DESIGN-machine-backup-exclusions.md`.

Design doc: `REQUIREMENT-DESIGN-machine-backup-exclusions.md`

7/9 new doctest leaves are RED; `keep-sqlite` and `keep-images` already GREEN
(assets must stay included).

## Feature summary

1. Bump `exclusionConfigVer` to `"1.1"`
2. Add 6 path entries to `builtinExclusionEntries` in `server/machinebackup/exclusions.go`
3. Add special rules: `**/upload-chunks`, `**/*.log`, `**(binary)` (synthetic entry in ExcludedList)
4. Add `includedPaths` to `ExclusionRules` for per-file `--include` overrides
5. Extend `ReasonFor` / `IsExcluded` for upload-chunks segment and .log suffix
6. In `walk.go`, after path rules: check include override → log suffix → `IsExecutableBinary`
7. Add `IsExecutableBinary(path) (bool, string, error)` in `dot-pkgs-with-critic/go-pkgs/file/detect/` — ELF/Mach-O/PE only
8. Wire `--include` from API/CLI through to `MergeExclusions` / `BuildPlan`
9. Unit tests in `server/machinebackup/` and `dot-pkgs-with-critic/go-pkgs/file/detect/`

## Evaluation order

1. includedPaths exact match → include
2. path prefix / full-tree → exclude
3. **/node_modules segment → exclude
4. **/upload-chunks segment → exclude
5. **/*.log suffix → exclude
6. IsExecutableBinary → exclude
7. else → include

## New builtin path entries

| Path | Reason |
|------|--------|
| `.local/share/cursor-agent/versions` | Cursor agent version cache |
| `.opencode/bin` | OpenCode binary (reinstallable) |
| `.config/confluence-fetch-skill/data` | confluence-fetch-skill data cache |
| `.codex/.tmp` | Codex temporary plugin cache |
| `.local/share/opencode/repos` | OpenCode repo clone cache |
| `.local/share/opencode/log` | OpenCode application logs |

## Special rule entries in exclude_paths

| Path | Reason |
|------|--------|
| `**(binary)` | executable binaries (reinstallable) |
| `**/*.log` | log files |
| `**/upload-chunks` | incomplete upload temp state |

## Verify (all must pass)

```sh
go test ./server/machinebackup/... -count=1
go test ./dot-pkgs-with-critic/go-pkgs/file/detect/... -count=1
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
git diff ./tests/remote-agent-machine-backup   # must be clean
```