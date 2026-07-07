# REQUIREMENT: `remote-agent machine analyse-files`

## Status

Approved — user said "go ahead" (2026-07-06).

## Summary

Add `remote-agent machine analyse-files` subcommand that scans the **remote
server's full `$HOME`**, streaming one completed entry block at a time, then a
server-rendered summary. Reuse `github.com/xhd2015/dot-pkgs/go-pkgs/git/scan_repo`
for per-entry git discovery and worktree aggregation.

## Data model

### Server package: `server/machineanalyse`

| Type | Purpose |
|------|---------|
| `EntryResult` | One top-level `~` child: name, kind (file/dir), children, semantic lines, aggregates |
| `ChildLine` | Immediate child: name, bytes, human size |
| `SemanticLine` | Enricher field: key, count, unit, bytes, human size, extra (e.g. lines) |
| `Aggregates` | GitRepos, LinkedWorktrees, NodeModulesDirs (recursive count) |
| `ScanSummary` | Global rollups for summary + `done` frame |

### Wire protocol

- Endpoint: `POST /api/remote-agent/machine/analyse-files/stream`
- Use `server/streaming/progress.Writer` (same as machine backup dry-run)
- Per finished entry: `EmitLog(block, verbatim=true)` — full text block as designed below
- Final summary: multiple `EmitLog` lines, `verbatim=true`
- `EmitDone` with structured JSON (`entries`, `git_repos`, `codex_sessions`, …)

### CLI

- `cmd/agentcli/machine.go`: add `analyse-files` subcommand
- `streamcmd.Run` with custom log printer (print verbatim blocks as-is)
- Help text under `machine` help

## Per-entry output format (canonical)

```
> <entry-name>

  > <child>         <human-size>     ← 1. immediate children (deep-aggregated bytes)
  ...

  <semantic-field>  <count> <unit>  <human-size>   ← 2. enricher only; NO prefix
  ...

  git-dirs          <N>              ← 3. aggregates; omit when 0
  worktrees         <N> linked       ← omit when 0
  node_modules      <N> dirs         ← omit when 0
```

### Top-level files

```
> <filename>
  size    <human-size>
  lines   <N>            OR  lines   (binary)
```

### Rules

- Scan **every** immediate child of `$HOME` (dirs + files); no skip list
- Deep walk for sizes; display immediate children only
- `node_modules N dirs` = recursive count of dirs named `node_modules` under entry (distinct from child `> node_modules`)
- Semantic + child overlap allowed (show both)
- Order: children → semantic → aggregates
- User-facing stdout ends with trailing `\n` after last line

## Semantic enrichers (top-level entry name → fields)

### `.codex`

| Field | Count | Size scope |
|-------|-------|------------|
| sessions | `rollout-*.jsonl` under `sessions/**` | `sessions/` |
| skills | top-level dirs under `skills/` | `skills/` |
| rules | files in `rules/` | `rules/` |
| plugins | top-level dirs under `plugins/` (0 if absent) | `plugins/` |
| cache | — | `cache/` |
| logs | sqlite `logs_*.sqlite` + wal/shm | those files |
| state | sqlite `state_*.sqlite` + wal/shm | those files |
| memories | `memories/` + `memories_*.sqlite` | combined |
| history | lines of `history.jsonl` | file |
| models-cache | — | `models_cache.json` |
| shell-snapshots | entries in `shell_snapshots/` | dir |

### `.grok`

| Field | Count | Size scope |
|-------|-------|------------|
| sessions | subdirs of `sessions/` | `sessions/` |
| projects | subdirs of `projects/` | `projects/` |
| skills | top-level under `skills/` | `skills/` |
| downloads | — | `downloads/` |
| logs | — | `logs/` |
| marketplace-cache | top-level under `marketplace-cache/` | dir |
| vendor | — | `vendor/` |
| active-sessions | entries in `active_sessions.json` if parseable | file |

### `.cursor`

| Field | Count | Size scope |
|-------|-------|------------|
| projects | subdirs of `projects/` | `projects/` |
| chats | subdirs of `chats/` | `chats/` |
| skills | items under `skills-cursor/` | dir |
| ai-tracking | — | `ai-tracking/` |

### `.knowledge-hub`

| Field | Count | Size scope |
|-------|-------|------------|
| knowledges | entries in `knowledges/` (excl `.git`) | `knowledges/` |
| conversations | items in `conversations/` | `conversations/` |

### `.knowledge-index`

| Field | Count | Size scope |
|-------|-------|------------|
| agents | subdirs of `agents/` | `agents/` |
| knowledge-base | entries in `knowledge_base/` | dir |
| conversations | items in `conversations/` | dir |

### `.openclaw`

| Field | Count | Size scope |
|-------|-------|------------|
| agents | subdirs of `agents/` | `agents/` |
| workspace | items in `workspace/` | `workspace/` |
| plugin-skills | items under `plugin-skills/` | dir |
| memory | — | `memory/` |
| logs | — | `logs/` |
| npm | — | `npm/` |

### `.opencode` (dot dir)

| Field | Count | Size scope |
|-------|-------|------------|
| bin | files in `bin/` | `bin/` |
| node_modules | child dir (size, not recursive count) | `node_modules/` child |

## Summary block (server-rendered)

```
analyse-files summary
  home:              <path>
  entries scanned:   <N>   (<dirs> dirs, <files> files)
  total size:        <human>

  git repos:         <N>
  linked worktrees:  <N>

  codex sessions:    <N>     ← only when ~/.codex exists (show 0 if empty)
  codex skills:      <N>     ← only when ~/.codex exists
  grok sessions:     <N>     ← only when ~/.grok exists
  grok projects:     <N>     ← only when ~/.grok exists
  grok skills:       <N>     ← only when ~/.grok exists
  cursor projects:   <N>     ← only when ~/.cursor exists
  cursor chats:      <N>     ← only when ~/.cursor exists
  knowledge-hub knowledges: <N>  ← only when ~/.knowledge-hub exists
  knowledge-index agents: <N>     ← only when ~/.knowledge-index exists
  openclaw agents:   <N>     ← only when ~/.openclaw exists

  node_modules:      <N> dirs

  largest entries:   (top 5)
    <name>           <size>
```

Topic-present rule: omit tool summary lines when indicator dir absent.

## Git integration

Per top-level dir entry `D`:

```go
scan_repo.Scan(ctx, scan_repo.Options{
    Roots:         []string{filepath.Join(home, D)},
    ListWorktrees: true,
    OnRepo:        ..., // count main repos + linked worktrees
})
```

- `git-dirs` = number of repos discovered (main + worktree checkout rows)
- `worktrees N linked` = count of non-main worktrees across repos in that entry

## Reuse

- `scan_repo` from dot-pkgs (already in go.mod)
- `progress.Writer` + `streamcmd` pattern from machine backup
- Display helpers: human size formatting (can mirror `machinebackup.formatSize` or tmpfiles pattern)
- Do NOT import disk-usage-analyser module; copy minimal logic into `server/machineanalyse`

## Test approach

Doctest tree: `./tests/remote-agent-machine-analyse-files/`

Mirror `tests/remote-agent-machine-backup/` harness:

- Isolated `serverHome` with `HOME=serverHome`
- Ephemeral ai-critic-server + remote-agent CLI
- Seed fixtures for: generic dir, top-level file (text + binary), `.codex` with sessions/skills, dir with nested git (use minimal git init fixtures), `node_modules` child vs recursive count

### Scenarios to cover (minimum)

| Leaf | What it proves |
|------|----------------|
| `stream/basic` | Exit 0; `home:` line; `analyse-files summary`; entry blocks with `>` headers |
| `stream/codex-semantic` | `.codex` block: children before semantic; sessions/skills counts; summary includes codex lines |
| `stream/file-lines` | Text file shows `lines N`; binary shows `lines (binary)` |
| `stream/git-dirs` | Entry with git repo shows `git-dirs 1`; entry without omits line |
| `stream/node-modules` | Entry with child `node_modules` AND `node_modules N dirs` aggregate |
| `stream/entry-order` | Blocks sorted alphabetically by entry name |
| `stream/topic-absent` | When `.grok` absent, summary omits grok lines |

Unit tests in `server/machineanalyse/` for enrichers and formatters (optional but encouraged).

## CLI example (fixture home)

```
remote-agent machine analyse-files
home: /tmp/server-home-xxx

> .codex
  > sessions                         12 KB
  > skills                           4 KB
  > cache                            1 KB
  sessions          2 rollouts       12 KB
  skills            1 skill           4 KB
  plugins           0 plugins         0 B
  cache             —                 1 KB
  git-dirs  1

> notes.txt
  size    18 B
  lines   2

> plain-dir
  > sub                                6 B

analyse-files summary
  home:              /tmp/server-home-xxx
  entries scanned:   3   (2 dirs, 1 files)
  ...
  codex sessions:    2
  codex skills:      1
```

## Non-goals (v1)

- macOS `Library/Application Support/Cursor` nested enricher inside `Library` entry
- Real network / remote-agent exec against production server in doctests
- JSON-only endpoint (stream only for CLI v1)

## Verify commands

```sh
go run ./script/build
doctest vet ./tests/remote-agent-machine-analyse-files
doctest test -v ./tests/remote-agent-machine-analyse-files/...
go test ./server/machineanalyse/... -count=1
```