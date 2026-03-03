# Plan: Cursor Agent Chat Mode & ACP (Agent Communication Protocol)

## Research Findings

### ACP is a Real, Well-Known Protocol

The **Agent Communication Protocol (ACP)** is an open protocol for AI agent interoperability, now part of **A2A under the Linux Foundation**. It addresses the fragmentation problem where modern AI agents are built in isolation across different frameworks. It is NOT something we need to invent -- it already exists.

- **Website**: https://agentcommunicationprotocol.dev/
- **Spec**: https://github.com/agntcy/acp-spec
- **SDKs**: Python and TypeScript (`@agentclientprotocol/sdk`)
- **Core concepts**: Agent manifests, message structure, stateful agents, distributed sessions, REST API with OpenAPI spec

### Cursor Agent + ACP

**Cursor-agent does NOT natively speak ACP.** It uses a custom event stream format. However, community adapters bridge the gap:

1. **cursor-acp** (github.com/roshan-c/cursor-acp) - The main adapter
   - Translates between ACP (NDJSON over stdio) and cursor-agent's custom event stream
   - Works with Zed, JetBrains, Neovim, Emacs, and other ACP clients
   - Uses `cursor-agent --print --output-format stream-json` flags
   - Supports multi-turn conversations via `--resume <resumeId>`
   - Maps cursor tool calls to ACP tool types:
     - `readToolCall` → `read_files`
     - `writeToolCall` → `edit_files`
     - `bashToolCall` → `run_bash_command`
     - `grepToolCall`/`globToolCall` → `search_files`

2. **cursor-agent-acp-npm** (github.com/blowmage/cursor-agent-acp-npm) - Another adapter (80+ stars)

### Key Insight: cursor-agent has structured JSON output

The `--output-format stream-json` flag on cursor-agent produces structured JSON events, which means we **don't need to parse terminal ANSI output**. We can directly consume structured events.

---

## Motivation

Currently, the cursor agent (`cursor-agent`) is defined as `Headless: false` (terminal-only) in our system. Users can only interact with it through the terminal tab. Other agents like `opencode` support a headless server mode with HTTP endpoints that enable a rich chat UI.

The goal is to:
1. **Leverage the existing ACP ecosystem** rather than inventing our own protocol
2. **Enable cursor agent to support chat mode** via an ACP adapter
3. **Allow seamless switching** between terminal and chat modes for any agent

## Architecture

### Option A: Use existing cursor-acp adapter (Recommended)

Leverage the community `cursor-acp` npm package and expose it via our backend:

```
┌──────────────────────────────────────────────┐
│  Frontend (existing Chat UI)                 │
│  ├─ POST /session/{id}/prompt_async          │
│  ├─ GET /event (SSE)                         │
│  └─ GET /session/{id}/message                │
└──────────────┬───────────────────────────────┘
               │ HTTP (proxied via our backend)
┌──────────────▼───────────────────────────────┐
│  Our Go Backend                              │
│  ├─ Spawns cursor-acp adapter process        │
│  ├─ Bridges HTTP ↔ NDJSON stdio              │
│  └─ Manages lifecycle                        │
└──────────────┬───────────────────────────────┘
               │ NDJSON over stdio
┌──────────────▼───────────────────────────────┐
│  cursor-acp (Node.js process)                │
│  ├─ ACP ↔ cursor-agent translation           │
│  └─ Spawns cursor-agent child process        │
└──────────────┬───────────────────────────────┘
               │ custom event stream
┌──────────────▼───────────────────────────────┐
│  cursor-agent                                │
│  --print --output-format stream-json         │
└──────────────────────────────────────────────┘
```

### Option B: Direct cursor-agent integration (Alternative)

Build our own Go adapter that directly consumes cursor-agent's `stream-json` output:

```
┌──────────────────────────────────────────────┐
│  Frontend (existing Chat UI)                 │
│  (Same HTTP endpoints as opencode)           │
└──────────────┬───────────────────────────────┘
               │ HTTP
┌──────────────▼───────────────────────────────┐
│  Our Go Backend                              │
│  ├─ CursorAgentAdapter (Go)                  │
│  ├─ Spawns cursor-agent with stream-json     │
│  ├─ Parses JSON events → chat messages       │
│  ├─ Exposes same HTTP API as opencode        │
│  └─ SSE event broadcasting                   │
└──────────────┬───────────────────────────────┘
               │ stream-json
┌──────────────▼───────────────────────────────┐
│  cursor-agent                                │
│  --print --output-format stream-json         │
└──────────────────────────────────────────────┘
```

### Option C: Use `@blowmage/cursor-agent-acp` npm package (Recommended)

Leverage the most popular community adapter (`github.com/blowmage/cursor-agent-acp-npm`, 92 stars) which provides a production-ready, full-featured ACP adapter:

```
┌──────────────────────────────────────────────┐
│  Frontend (existing Chat UI)                 │
│  ├─ POST /session/{id}/prompt_async          │
│  ├─ GET /event (SSE)                         │
│  └─ GET /session/{id}/message                │
└──────────────┬───────────────────────────────┘
               │ HTTP (proxied via our backend)
┌──────────────▼───────────────────────────────┐
│  Our Go Backend                              │
│  ├─ Spawns cursor-agent-acp adapter          │
│  ├─ Bridges HTTP ↔ NDJSON stdio              │
│  └─ Manages lifecycle                        │
└──────────────┬───────────────────────────────┘
               │ NDJSON over stdio
┌──────────────▼───────────────────────────────┐
│  @blowmage/cursor-agent-acp (Node.js)       │
│  ├─ Full ACP protocol implementation         │
│  ├─ Session management & persistence         │
│  ├─ Tool system (filesystem, terminal, etc.) │
│  ├─ Security framework                       │
│  └─ cursor-agent integration                 │
└──────────────┬───────────────────────────────┘
               │ custom event stream
┌──────────────▼───────────────────────────────┐
│  cursor-agent CLI                            │
└──────────────────────────────────────────────┘
```

**Key features of @blowmage/cursor-agent-acp:**
- 100% ACP schema compliance (strict adherence to spec)
- Full session management with persistence
- Complete tool system: filesystem, terminal, and Cursor-specific tools
- Security framework: path validation, command filtering, access controls
- 200+ unit and integration tests
- <100ms average response time
- Cross-platform (macOS, Linux, Windows)
- Configurable via JSON config file
- Debug logging support

**Install:**
```bash
npm install -g @blowmage/cursor-agent-acp
cursor-agent-acp  # starts the ACP adapter on stdio
```

**Advantages over Option A (roshan-c):**
- More mature project (92 stars vs 11)
- Has tests (200+)
- Better documentation
- Configurable security framework
- Session persistence

**Advantages over Option B (custom Go adapter):**
- Already built and tested
- Full ACP compliance out of the box
- Handles edge cases (auth detection, cancellation, session sync)
- Active maintenance with community support

**Trade-offs:**
- Requires Node.js runtime
- Two child processes (cursor-agent-acp → cursor-agent)
- Less control over the translation layer

### Recommendation: Option C

Option C is preferred because:
- Most mature and well-tested adapter (92 stars, 200+ tests)
- Full ACP schema compliance
- Production-ready with security framework
- We only need to bridge NDJSON↔HTTP in our Go backend (thin layer)
- Future-proof: when cursor-agent natively supports ACP, we just drop the adapter

## Implementation Plan

### Phase 1: Cursor Agent JSON Event Parser (Go)

1. **Study cursor-agent's `stream-json` output format**
   - Run `cursor-agent --print --output-format stream-json` and capture events
   - Document the event types and their JSON structure
   - Reference the cursor-acp adapter's `mapCursorEventToAcp()` function for known event types

2. **Create `server/agents/cursor/parser.go`**
   - Parse cursor-agent's JSON event stream
   - Map events to our internal message types (text, tool_call, thinking, error)

### Phase 2: Cursor Agent HTTP Adapter (Go)

1. **Create `server/agents/cursor/adapter.go`**
   - Implement the same HTTP interface our frontend expects (matching opencode's API)
   - Session management (spawn cursor-agent per session with `--resume` support)
   - Prompt handling (write to stdin, read events from stdout)
   - SSE event broadcasting

2. **Update `server/agents/agents.go`**
   - Change cursor-agent's `Headless` to `true` (or add a new flag like `HasChatMode`)
   - When launching cursor-agent session, start our adapter instead of raw terminal
   - Route proxy requests through our adapter

### Phase 3: Frontend Integration

1. **Update agent definitions** to show cursor-agent as chat-capable
2. **Reuse existing chat UI** (AgentChat.tsx, ChatMessage.tsx) - no changes needed if the HTTP API matches
3. **Optional**: Add mode switching (terminal ↔ chat) in the agent picker

### Phase 4: Protocol Generalization

1. Extract common agent adapter interface in Go
2. Apply same pattern for claude-code and codex
3. Document the expected HTTP API as our internal "agent chat protocol"

## Cursor-Agent Event Types (from cursor-acp research)

Based on the cursor-acp adapter, cursor-agent emits these event types in stream-json mode:

| Event Type | Description | Maps To |
|------------|-------------|---------|
| Text content | Agent's text response | Chat message (assistant) |
| `readToolCall` | File read operation | Tool call: "read_files" |
| `writeToolCall` | File write/edit | Tool call: "edit_files" |
| `bashToolCall` / `shellToolCall` | Shell command execution | Tool call: "run_bash_command" |
| `grepToolCall` / `globToolCall` | File search | Tool call: "search_files" |
| Plan mode indicator | Agent switched to plan mode | Mode update notification |

## Timeline Estimate

| Phase | Duration | Priority |
|-------|----------|----------|
| Phase 1: JSON event parser | 1-2 days | High |
| Phase 2: HTTP adapter | 2-3 days | High |
| Phase 3: Frontend integration | 1 day | Medium |
| Phase 4: Generalization | 2-3 days | Low |

## Prerequisites

- `cursor-agent` CLI must be installed and authenticated (`cursor-agent login`)
- The `--output-format stream-json` flag must be available in the installed version

## Open Questions

1. **Exact JSON schema**: What is the precise JSON format of cursor-agent's stream-json output? (Need to capture and document)
2. **Stdin prompt format**: How does cursor-agent expect prompts via stdin in non-interactive mode?
3. **Resume mechanism**: Does `--resume <id>` work reliably for multi-turn?
4. **Concurrent sessions**: Can multiple cursor-agent processes run simultaneously?
5. **Authentication scope**: Does cursor-agent auth work headlessly (no browser)?

## Summary

The ACP protocol already exists as a Linux Foundation standard. Cursor-agent supports a structured JSON output format (`--output-format stream-json`) that can be parsed without ANSI terminal parsing. The recommended approach is to build a Go adapter in our backend that spawns cursor-agent, consumes its JSON events, and exposes them through our existing chat HTTP API -- enabling the cursor agent to work with our existing chat UI with minimal frontend changes.
