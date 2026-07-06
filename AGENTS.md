# Agents

Guidelines for implementing and extending agent-facing remote commands in this repo.

## Streaming remote-agent commands

Use **SSE streaming** when a command performs multi-step work and users would
otherwise stare at a blank terminal. The CLI prints each result as it arrives via
`streamcmd.Run`; the server emits frames with `server/streaming/progress.Writer`.

### When to stream

| Stream (SSE + `streamcmd`) | Keep buffered JSON |
|----------------------------|-------------------|
| Dry-run plans with many entries | Fast CRUD / tiny payloads |
| Multi-step diagnostics | Single-field lookups |
| Long builds or startups | No incremental value |

References:

- **CLI hybrid pattern** — `cmd/agentcli/wsproxy_doctor.go` (server stream + local `After` hook)
- **Machine backup/restore dry-run** — `cmd/agentcli/machine.go` (custom progress printers + summary `After`)
- **Server writer** — `server/streaming/progress/writer.go`
- **Doctor stream handler** — `server/proxy/wsproxy/doctor.go` (`DoctorStream`)

### Wire protocol

Endpoint returns `Content-Type: text/event-stream`. Each frame is `data: <json>\n\n`.

| `type` | Purpose | Key fields |
|--------|---------|------------|
| `section` | Section header | `message` |
| `progress` | One completed item | `layer`, `name`, `status`, `detail`, `hint` |
| `meta` | Banner / context | `message`, `try_url`, `server_status`, … |
| `done` | Terminal success summary | command-specific summary fields |
| `error` | Fatal failure | `message` |

Rules:

- Emit each result **when it is ready** — do not buffer the full plan before printing.
- Always end with `done` (normal completion) or `error` (aborted).
- Keep non-stream JSON endpoints when other callers depend on them; add `.../stream` for CLI.

### Two-phase CLI output (machine commands)

`remote-agent machine backup --dry-run` and `restore --dry-run` use a **stream
phase** followed by a **summary phase**. Both phases are emitted by the **server**
as SSE frames; the CLI prints them verbatim.

1. **Stream phase** — server emits `section` + `progress` frames as entries are
   discovered (per-entry sizes for backup; skip/update/create lines for restore).
   CLI custom `progress` printers format these lines.
2. **Summary phase** — server emits `log` frames with `verbatim: true` containing
   the full human-readable rollup (e.g. `dry-run: machine backup plan`, section
   totals, `TOTAL:` line). CLI `printMachineStreamLog` prints them as-is.
3. **`done` frame** — structured JSON for tests/API callers; **not** the primary
   display source. The CLI must not reconstruct summary text from `done` in an
   `After` hook.

Real backup/restore (no `--dry-run`) keeps existing tar.xz upload/download paths;
restore apply still uses the JSON endpoint and prints skip lines from the returned plan.

### Implementing a new streaming command

1. **Server** — add `ThingStream(w http.ResponseWriter, …) error` using `progress.NewWriter`; check every `Emit*` return value.
2. **Route** — register `POST /api/.../thing/stream` beside any JSON endpoint.
3. **CLI** — call `streamcmd.Run` with appropriate `Print` flags and optional `Printer` overrides.
4. **Summary** — render summary text on the **server** and emit as `log` frames
   (`progress.EmitLog(msg, verbatim=true)`). Enable `streamcmd.Logs` with a
   verbatim log printer on the CLI. Reserve the `done` frame for structured data only.

Use `progress.EmitLog` for summary lines; use `section` / `progress` for incremental work.

See `skills/create-streaming-command/SKILL.md` for a full checklist.