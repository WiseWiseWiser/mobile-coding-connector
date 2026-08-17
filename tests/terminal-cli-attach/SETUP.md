# Scenario

**Feature**: `remote-agent terminal attach` types into a live session

```
# writer-held CLI attach
REST create shell
  -> WS hold writer (web-like)
  -> AttachWithIO(TerminalAttachConnectOptions)
  -> echo MARKER
```

## Preconditions

- Isolated `ptywrap.NewManager` per leaf (no process-global session map).
- No real user `HOME` / `AI_CRITIC_HOME`.

## Steps

1. Leaf sets `req.Phase` and `req.Marker`.
2. Harness starts an in-process terminal API, holds a writer, attaches with
   production CLI options, and types `echo <marker>`.
