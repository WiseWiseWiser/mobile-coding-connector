# REQUIREMENT-IMPLEMENT: file/analyse migration

## Context

Design: `REQUIREMENT-DESIGN-file-analyse-migration.md`

**Tests are sealed — do not modify** `dot-pkgs-with-critic/go-pkgs/file/analyse/tests/`

**ai-critic integration tests sealed — do not modify** `tests/remote-agent-machine-analyse-files/`

## Implement in dot-pkgs (`dot-pkgs-with-critic/go-pkgs/file/analyse/`)

Create package `analyse` with migrated code from `ai-critic/server/machineanalyse`:

- types.go, size.go, scan.go, enrichers.go, format.go, format_test.go
- `Scan(ctx, Options)` with `OnEntry func(EntryResult) error` — call after each entry; abort on error
- Use `git/scan_repo` for git aggregates
- Prefer `file/detect` for binary detection if straightforward

## Rewire ai-critic (`server/machineanalyse/`)

- Keep api.go, stream.go only (thin adapter)
- stream.go: `analyse.Scan` with `OnEntry` → `pw.EmitLog(analyse.FormatEntryBlock(e), true)`
- Delete: types.go, size.go, scan.go, enrichers.go, format.go, format_test.go from machineanalyse
- Import: `github.com/xhd2015/dot-pkgs/go-pkgs/file/analyse`

For local dev, ai-critic go.mod may need `replace` for dot-pkgs pointing to `../dot-pkgs-with-critic/go-pkgs` if not published yet.

## Verify

```sh
cd dot-pkgs-with-critic/go-pkgs
go test ./file/analyse/... -count=1
doctest test -v ./file/analyse/tests/...

cd /Users/xhd2015/.wrk/worktrees/ai-critic-master-2026-07-05-backup-server
go run ./script/build
doctest test -v ./tests/remote-agent-machine-analyse-files/...
git diff dot-pkgs-with-critic/go-pkgs/file/analyse/tests   # empty
git diff tests/remote-agent-machine-analyse-files          # empty
```

All 9 dot-pkgs doctests + 7 ai-critic integration doctests must pass GREEN.