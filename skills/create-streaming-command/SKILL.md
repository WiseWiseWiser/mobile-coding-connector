---
name: create-streaming-command
description: >
  Add a new streaming remote-agent subcommand using streamcmd + progress.Writer.
  Use when adding a remote-agent command that prints output incrementally,
  migrating a buffered JSON endpoint to SSE, or when the user mentions
  streaming command, streamcmd, progress framework, or /create-streaming-command.
---

# Create Streaming remote-agent Command

Guide for adding a **streaming-favored** `remote-agent` subcommand: server emits
SSE events as work completes; CLI prints immediately via `streamcmd.Run`.

## When to stream

| Use streaming | Keep JSON |
|---------------|-----------|
| Multi-step or slow work (checks, builds, startups) | Fast CRUD (status, config get) |
| User waits on a blank terminal today | Single small payload |
| Several independent results to show as they land | No incremental value |

Existing references:

- **Progress checks** — `cmd/remote-agent/wsproxy_doctor.go` → `GET /api/ws-proxy/doctor/stream`
- **Log lines** — `cmd/remote-agent/wsproxy.go` start (legacy `StreamSSEWithDone`; migrate to `streamcmd`)
- **Server writer** — `server/streaming/progress/writer.go`, `server/proxy/wsproxy/doctor.go` (`DoctorStream`)

---

## Wire protocol (SSE)

Endpoint returns `Content-Type: text/event-stream`. Each frame is `data: <json>\n\n`.

| `type` | Purpose | Key fields |
|--------|---------|------------|
| `log` | Status line (build/start style) | `message` |
| `progress` | One completed step/check | `id`, `layer`, `name`, `status`, `detail`, `hint` |
| `section` | Section header | `message` |
| `meta` | Banner / context | `message`, `try_url`, `server_status`, … |
| `done` | Terminal success summary | command-specific + e.g. `healthy`, `checks_total` |
| `error` | Fatal; stream ends | `message` |

**Rules**

- Emit each result **when it is ready** — do not buffer until the end.
- Always end the stream with a terminal frame: `done` (normal completion, including
  "completed but unhealthy") or `error` (aborted before meaningful work).
- Keep `GET /api/.../thing` JSON endpoint if callers depend on it; add `GET|POST /api/.../thing/stream` for CLI.

---

## Server (3 steps)

### 1. Stream handler on your manager/service

Use `server/streaming/progress.Writer`. **Check every emit return value** — never
`_ = pw.Emit…`.

#### Error handling model

| Situation | Server action | Return from `*Stream` |
|-----------|---------------|------------------------|
| `NewWriter` is nil (no `http.Flusher`) | — | `return fmt.Errorf("streaming not supported")` → handler writes JSON API error |
| Request invalid before any SSE frame | `EmitError(msg)` | `return nil` (client reads `type=error`) |
| Step/check failed but stream should finish | `EmitProgress` with `status: fail` | continue, then `EmitDone` with summary |
| Work aborted mid-run (cannot continue) | `EmitError(msg)` | `return nil` |
| Any `Emit*` returns error (write/flush failed) | — | `return err` |
| Happy path | emit items as they complete | `return pw.EmitDone(summary)` |

Pattern references: `Manager.StartStream` (fatal → `SendError`), `Manager.DoctorStream`
(check failures → `progress` + `done` with `healthy: false`).

#### Emit callback returns `error`

Refactor step runners to accept a callback that propagates write failures:

```go
type ProgressEmitter func(progress.Item) error
```

#### Example handler

```go
import (
    "fmt"
    "net/http"

    "github.com/xhd2015/ai-critic/server/streaming/progress"
)

func (m *Manager) MyCommandStream(w http.ResponseWriter, args Args) error {
    pw := progress.NewWriter(w)
    if pw == nil {
        return fmt.Errorf("streaming not supported")
    }

    if err := args.Validate(); err != nil {
        if emitErr := pw.EmitError(err.Error()); emitErr != nil {
            return emitErr
        }
        return nil
    }

    if err := pw.EmitSection("My command"); err != nil {
        return err
    }

    checks, runErr := m.runSteps(args, func(item progress.Item) error {
        return pw.EmitProgress(item) // emit immediately; abort stream on write error
    })
    if runErr != nil {
        if emitErr := pw.EmitError(runErr.Error()); emitErr != nil {
            return emitErr
        }
        return nil
    }

    healthy, failed := aggregateHealth(checks)
    return pw.EmitDone(map[string]any{
        "success":       healthy,
        "checks_total":  len(checks),
        "checks_failed": failed,
    })
}
```

Refactor long synchronous functions to accept `ProgressEmitter` (see
`serverDoctorChecks` in `server/proxy/wsproxy/doctor.go`). The emitter must return
the error from `pw.EmitProgress` so a broken client connection stops work promptly.

For **log-only** streams (like ws-proxy start), use `sse.Writer` directly:
`SendError` for fatals, `SendLog` for progress lines, `SendDone` at the end — same
return rules (fatal → `SendError` + `return nil`; write failure → `return err`).

### 2. Register route

In your package `api.go` (pattern from `server/proxy/wsproxy/api.go`):

```go
mux.HandleFunc("GET /api/my-feature/action/stream", handleMyStream)

func handleMyStream(w http.ResponseWriter, r *http.Request) {
    args, err := parseMyStreamArgs(r)
    if err != nil {
        writeAPIErr(w, newError(ErrBadRequest, err.Error()))
        return
    }
    if err := GetManager().MyCommandStream(w, args); err != nil {
        // Only reached before SSE headers are committed (e.g. no Flusher).
        writeAPIErr(w, toAPIError(err))
    }
}
```

Parse/validate what you can **before** calling `*Stream`. Once streaming starts,
signal failures with `EmitError` / `EmitProgress`+`EmitDone`, not `writeAPIErr`.

Naming: prefer `/api/<feature>/<action>/stream` (matches `/api/ws-proxy/doctor/stream`).

### 3. Test hooks (if integration tests need determinism)

Add hooks in `testhooks.go` (see `SetTestStubNetworkChecks`, `SetTestUpstreamFetchDelay`
in `server/proxy/wsproxy/testhooks.go`) so doctests avoid real network without
changing production behavior.

---

## Client CLI (minimal — use `streamcmd`)

Package: `cmd/remote-agent/streamcmd`

### A — declarative `Print` flags (default)

```go
import (
    "net/http"
    "net/url"

    "github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
)

return streamcmd.Run(getClient, streamcmd.Spec{
    Method: http.MethodGet,
    Path:   "/api/my-feature/action/stream",
    Query:  url.Values{"foo": {foo}},
    Print:  streamcmd.Logs, // or ProgressChecks | Sections | Meta (bitwise OR)
    After:  myAfterHook,    // optional; see below
})
```

| Flag | Prints |
|------|--------|
| `streamcmd.Logs` | `type=log` → `  message` |
| `streamcmd.ProgressChecks` | `type=progress` → `[ok] name: detail` |
| `streamcmd.Sections` | `type=section` → `Title:` |
| `streamcmd.Meta` | `type=meta` → banner / try_url / server_status |

### B — override one printer

```go
Print: streamcmd.Logs,
Printer: streamcmd.Printer{
    Log: func(ev client.StreamEvent) error {
        fmt.Fprintf(os.Stderr, "[remote] %s\n", ev.Message)
        return nil
    },
},
```

Non-nil `Printer.*` replaces the default for that type only; other types still use `Print` defaults.

### `After` — hybrid commands (server stream + local work)

Doctor pattern: server checks stream over SSE; client-only checks run after `done`:

```go
After: func(done map[string]any) error {
    // read summary from done payload
    for _, localCheck := range runLocalChecks(done) {
        _ = streamcmd.PrintProgress(localCheck) // reuse same formatter
    }
    return nil // or error for non-zero CLI exit
},
```

`streamcmd.Run` handles: client resolve, `Accept: text/event-stream`, event loop,
`error` propagation, `done` capture, stdout flush after each print.

### Wire up subcommand

1. Add handler in the appropriate `cmd/remote-agent/<topic>.go` switch.
2. Parse flags with `less-gen/flags` before calling `streamcmd.Run`.
3. Return `error` from `Run`/`After` so `main` exits non-zero on failure.

---

## Log-only command (migrate from `StreamSSEWithDone`)

**Before** (boilerplate in every command):

```go
result, err := c.StreamSSEWithDone(url, nil, func(ev client.ServerStreamEvent) {
    if ev.Type == "log" && ev.Message != "" {
        fmt.Printf("  %s\n", ev.Message)
    }
})
```

**After**:

```go
return streamcmd.Run(getClient, streamcmd.Spec{
    Method: http.MethodPost,
    Path:   "/api/ws-proxy/start/stream",
    Print:  streamcmd.Logs,
    After: func(done map[string]any) error {
        publicURL, _ := done["public_url"].(string)
        // ...
        return nil
    },
})
```

---

## Tests

Add doctests under the appropriate tree (see `REQUIREMENT-DESIGN-streaming-progress.md`):

| Layer | Tree | What to assert |
|-------|------|----------------|
| Server unit | `server/.../tests/<feature>/` | SSE frame order, `progress` before `done`, content-type |
| Client unit | `client/tests/streaming/` | `client.Stream` / event decode |
| CLI unit | `cmd/remote-agent/streamcmd/tests/` | `Print` defaults and `Printer` overrides |
| Integration | `tests/streaming/` | Real `ai-critic-server` + `remote-agent` subprocess; stdout incremental |

Integration harness (`tests/streaming/SETUP.md`):

1. Build `ai-critic-server` + `remote-agent`
2. Temp `AI_CRITIC_HOME` + `script/lib.WriteTestCredentials`
3. Start server, poll `/ping`
4. Run `remote-agent --server http://127.0.0.1:PORT --token testpassword <cmd>`
5. Read stdout pipe line-by-line

```sh
doctest vet ./tests/streaming
doctest test ./tests/streaming/...
```

**Doctest harness rule:** avoid method receivers in `DOCTEST.md` Go blocks — use
package-level helpers (doctest codegen limitation).

---

## Verify

```sh
go run ./script/build
doctest test ./server/.../tests/<your-feature>/...
doctest test ./tests/streaming/...   # if integration added
go test ./server/streaming/... ./client/... ./cmd/remote-agent/... -count=1
```

Manual smoke:

```sh
remote-agent --server http://HOST:PORT --token TOKEN my-subcommand
# output should appear line-by-line, not all at once at the end
```

---

## Checklist

- [ ] Server: emit events as work completes; terminal `done` or `error`
- [ ] Server: every `Emit*` checked; fatals use `EmitError`+`nil`; write failures returned
- [ ] Server: emit callback returns `error` from `EmitProgress`
- [ ] Route: `.../stream` registered; JSON endpoint kept if needed
- [ ] CLI: `streamcmd.Run` with correct `Print` flags
- [ ] Hybrid: local post-work in `After`, reuse `streamcmd.PrintProgress` if applicable
- [ ] Tests: unit + integration (local server + remote-agent) for streaming guarantee
- [ ] `go run ./script/build` passes