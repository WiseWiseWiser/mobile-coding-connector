# Scenario

**Feature**: remote-agent agent-run attach + idle keepalive (P2)

```
seed meta[+terminal_session_id] -> attach WS | CLI attach
  -> FakeAttach / ResolveTTY inject -> echo | pings | clear errors
```

## Preconditions

- Product mounts `GET /api/agent-run/sessions/{id}/attach` (WebSocket).
- L2 uses `RegisterAPIWithOptions` inject hooks (no real grok/codex TTY).
- CLI `agent-run attach` for help and validation; live path may be client WS.

## Steps

1. Leaf sets attach session id, TTYMode, and optional ping inject.
2. Run dials attach or drives CLI validation.
3. Assert exit / errors / ReceivedOutput / PingCount.
