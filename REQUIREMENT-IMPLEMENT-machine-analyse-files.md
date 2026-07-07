# REQUIREMENT-IMPLEMENT: `remote-agent machine analyse-files`

## Context

Design requirement: `REQUIREMENT-DESIGN-machine-analyse-files.md`

**Tests are sealed — do not modify** `./tests/remote-agent-machine-analyse-files/` unless a test is provably wrong per spec.

## Feature summary

Implement `remote-agent machine analyse-files`:

1. **Server** — new package `server/machineanalyse`:
   - `RegisterAPI(mux)` → `POST /api/remote-agent/machine/analyse-files/stream`
   - Scan all immediate children of `$HOME` (no skips)
   - Per entry: format block (children `>` first, semantic second, aggregates last)
   - Stream each block via `progress.EmitLog(verbatim=true)` as entry completes
   - Server-rendered summary at end + `EmitDone` JSON
   - Use `scan_repo` from `github.com/xhd2015/dot-pkgs/go-pkgs/git/scan_repo` per entry for git-dirs/worktrees
   - Semantic enrichers: `.codex`, `.grok`, `.cursor`, `.knowledge-hub`, `.knowledge-index`, `.openclaw`, `.opencode`
   - Codex skills = top-level dirs under `skills/`
   - Files: size + lines; binary → `lines (binary)`
   - node_modules aggregate: `node_modules N dirs` (recursive); distinct from child `> node_modules`
   - Summary topic-present rule: show tool lines only when indicator dir exists

2. **CLI** — `cmd/agentcli/machine.go`:
   - Add `analyse-files` subcommand + help
   - `streamcmd.Run` → `/api/remote-agent/machine/analyse-files/stream`
   - Print verbatim log blocks; trailing newline after output

3. **Wire-up** — register API in `server/server.go` (or via machineanalyse.RegisterAPI like machinebackup)

## Test tree (7 leaves)

```
tests/remote-agent-machine-analyse-files/stream/
  basic/           home + summary + entry blocks
  codex-semantic/  .codex children before semantic; sessions/skills; summary codex lines
  file-lines/      text lines N; binary lines (binary)
  git-dirs/        git-dirs 1 when repo present; omitted when 0
  node-modules/    > node_modules child + node_modules N dirs aggregate
  entry-order/     alphabetical entry blocks
  topic-absent/    no grok summary lines when .grok absent
```

## Designer questions — implementer decisions

1. `.ai-critic` under serverHome is a real scanned entry (harness creds dir). Treat as normal entry.
2. Assert `plugins  0 plugins` in codex-semantic if tests expect it; read ASSERT.md.
3. Flexible size regex in tests is fine.

## Verify command

```sh
go run ./script/build
doctest test -v ./tests/remote-agent-machine-analyse-files/...
go test ./server/machineanalyse/... -count=1
git diff ./tests/remote-agent-machine-analyse-files   # must be empty
```

All 7 doctests must pass GREEN.