# Scenario

**Feature**: project terminal browser automation through quick-test

```
# quick-test serves the project-scoped MobileCodingConnector route
Run -> quick-test server + Vite -> /project/{name}/terminal

# TerminalManager creates an active terminal tab backed by /api/terminal
MobileCodingConnector -> TerminalManager -> usePureTerminal -> WebSocket /api/terminal

# leaf script records WebSocket sends and checks terminal behaviour
Playwright -> instrument WebSocket.send -> resize JSON -> Assert
```

## Preconditions

1. Quick-test server and Vite dev server are started by the root `Run` function.
2. The browser script can create a temporary project record through `/api/projects`.
3. The project terminal route renders `MobileCodingConnector` with a persistent `TerminalManager`.

## Steps

1. Each child leaf navigates to a project-scoped terminal route.
2. Each child leaf instruments browser WebSocket sends before navigation.
3. Each child leaf asserts terminal resize messages emitted by the page.

## Context

This grouping covers the tabbed project terminal, not the standalone `/terminal`
route. It verifies behaviours where `TerminalManager` layout state changes must
reach the backend through `usePureTerminal` resize messages.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.ScriptPath == "" {
		req.ScriptPath = "script.js"
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 120
	}
	return nil
}
```
