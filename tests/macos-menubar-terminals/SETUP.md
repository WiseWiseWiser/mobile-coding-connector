# Scenario

**Feature**: macOS menu bar Terminals formatting, attach/new commands, remote domain switcher, and Swift contracts

```
# pure helpers
TerminalSession(name,id,status) -> FormatTerminalTitle / empty label / attach|new cmds
# exited status appends " [EXITED]" to base title; running/empty/unknown → base only
remote-agent-config domains -> SelectDefaultDomain -> default persisted + Resolve

# Swift apps
local AICriticApp  -> Terminals + New Terminal + Refresh (no Server switcher)
remote AICriticApp -> Terminals + level-1 Server switcher + Refresh + periodic poll
session click / New Terminal -> iTerm2 only (no Terminal.app fallback)
```

## Preconditions

1. `macosapp/menubar` exports `FormatTerminalTitle`, `FormatTerminalsEmptyLabel`,
   `BuildTerminalAttachCommand`, `BuildTerminalNewCommand`, `AgentBinaryForApp`,
   and `PeriodicRefreshInterval` (30s).
2. `macosapp/remoteconfig` exports `SelectDefaultDomain` which sets `default` for
   a matching domain server URL.
3. Go helper leaves are pure (or temp-dir config write for domain select) — no
   network or live iTerm.
4. Client leaves read Swift sources under `macos-ai-critic/ai-critic-macos/` and
   `macos-ai-critic/ai-critic-remote-macos/` (plus Shared/).

## Steps

1. Leaf `Setup` sets `Op` and inputs (or `ClientLeaf` for Swift).
2. Root `Run` dispatches by `Op` to helpers or source inspection.
3. Leaf `Assert` checks exact strings, resolved domain, or Swift contract flags.

## Context

Implements REQUIREMENT-DESIGN-macos-menubar-terminals.md and
REQUIREMENT-DESIGN-macos-menubar-terminal-exited-title.md. Primary logic lives in
Go (`macosapp/menubar`, `macosapp/remoteconfig`); Swift mirrors for UI.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```
