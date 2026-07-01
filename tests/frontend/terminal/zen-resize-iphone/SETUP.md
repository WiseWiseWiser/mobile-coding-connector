# Scenario

**Bug**: iPhone project terminal zen mode changes layout without sending backend resize

```
# browser opens the project terminal at an iPhone-sized viewport
Playwright -> /project/{name}/terminal -> TerminalManager

# terminal starts connected and sends its initial resize
TerminalManager -> usePureTerminal -> WebSocket /api/terminal <- resize

# zen entry and exit resize the visible terminal container
Zen button -> TerminalManager zen mode -> active PureTerminalView fit -> resize
Exit Zen button -> TerminalManager normal mode -> active PureTerminalView fit -> resize
```

## Preconditions

1. The quick-test server is healthy.
2. The leaf can add the current repository as a temporary project in the isolated quick-test home.
3. Browser WebSocket instrumentation is installed before the terminal connection opens.

## Steps

1. Set `Request.ScriptPath` to `script.js` and `Request.TimeoutSecs` to `150`.
2. Use a `390 x 844` browser viewport before navigation.
3. Add the repository root as a temporary project through `/api/projects`.
4. Navigate to `/project/{name}/terminal`.
5. Wait for an active terminal tab, a terminal WebSocket, and at least one initial resize message.
6. Click `Zen` and wait for an additional resize message.
7. Click `Exit Zen` and wait for another additional resize message.
8. Print a JSON result with resize counts, latest resize payload, URL, and diagnostics.

## Context

The bug is specific to layout changes caused by terminal zen mode. Initial
connection resize messages are only setup evidence; the assertions use resize
counts captured immediately before each toggle to prove that the visible layout
transition sent a fresh backend resize without reload or reconnect.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.TimeoutSecs = 150
	return nil
}
```
