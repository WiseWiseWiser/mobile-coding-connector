# Scenario

**Feature**: auto-Skip Codex blocking Update available menu; banner vs menu detection

```
# classify path
signed snapshot fixture -> IsBlockingUpdateMenu / UpdateMenuSelection / CheckWritable

# fetch path
CODEX_SHOW_STATUS_COMMAND fake TUI -> FetchStatus waitForPrompt auto-Skip -> /status -> service ready
```

## Preconditions

1. Nested root under `tests/codex-usage/update-modal-skip` (does not inherit parent
   `Request`/`Run`). Fixtures live at `../testdata/update-modal-skip/`.
2. Implementer adds production helpers on `github.com/xhd2015/agent-pro/pkgs/agenttty`:
   - `IsBlockingUpdateMenu(text string) bool`
   - `UpdateMenuSelection(text string) string` // `UPDATE_NOW` | `SKIP` | `SKIP_UNTIL_NEXT` | `""`
3. Implementer narrows `checkCodexWritable` so residual banner text containing
   “Update available” is **not** permanently `loading` with reason `codex update available`.
4. Implementer adds Skip protocol in `agent/codex/tty` `waitForPrompt`:
   CSI Down (`\x1b[B`) → verify selection on Skip → Enter (`\r`) → poll until menu gone.
5. Fake TUI scripts: `fake-tui-auto-skip.py`, `fake-tui-stuck-update-now.py`.

## Steps

1. Root `Setup` sets defaults (timeout, PATH strip off by default).
2. Leaf sets `Op`, fixture or fake command.
3. Root `Run` dispatches classify vs fetch-inprocess.
4. Leaf `Assert` checks menu predicates, writable reason, or service status.

## Context

REQUIREMENT-DESIGN-codex-update-modal-skip.md. Nested tree is intentionally
**RED** until production helpers + Skip protocol land (compile and/or runtime).

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.FetchTimeoutSecs <= 0 {
		req.FetchTimeoutSecs = 30
	}
	return nil
}
```
