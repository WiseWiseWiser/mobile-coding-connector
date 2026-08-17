# Terminal CLI attach — writer-held input

`remote-agent terminal attach` must accept keystrokes on a live session even
when another client (the web UI) already holds the exclusive writer.

# DSN (Domain Specific Notion)

**Participants**

- **CLI attach handshake** — `agentcli.TerminalAttachConnectOptions` is what
  `remote-agent terminal attach` passes to `ptywrap/client.Attach`.
- **ptywrap session** — REST-created shell; one WebSocket claims `writer`
  (web-like); CLI attach is a second connection.
- **Harness** — in-process `httptest` + `ptywrap.RegisterAPIWithManager`
  (L2; no `ai-critic-server` binary).

**Behaviors**

- Attach to a running session whose writer is already held accepts typed
  `echo <marker>` (roleAttacher / `attach_mode=attach`).
- After that input, the session is still `running` and the original writer is
  still connected (CLI attach did not kill the shell).
- Live attach is in-place: no `\e[?1049l` / `\e[2J` first frame.
- `terminal attach` on an `exited` session errors (`<id> is exited`) and does
  not open a mute raw TTY.

## Version

0.0.2

## Decision Tree

```
tests/terminal-cli-attach/
├── DOCTEST.md
├── SETUP.md
├── writer-held/
│   ├── SETUP.md
│   ├── echo-reaches-pty/          (LEAF) typed echo appears while writer held
│   └── in-place-no-replay/        (LEAF) no 1049l/2J first frame
└── exited/
    ├── SETUP.md
    └── refuse-attach/             (LEAF) exited session is not attached
```

## Test Index

| # | Leaf | Description |
|---|---------|-------------|
| 1 | `writer-held/echo-reaches-pty` | Writer held; CLI attach options; `echo MARKER` in stdout; session still running |
| 2 | `writer-held/in-place-no-replay` | Live attach does not send alt-screen reset / clear |
| 3 | `exited/refuse-attach` | Exited session → `is exited` error; no Attach |

## How to Run

```sh
doctest vet ./tests/terminal-cli-attach
doctest test ./tests/terminal-cli-attach/...
# autonomous TTY gate (required before claiming attach done):
./script/verify-terminal-attach-e2e.sh
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/tests/terminal-cli-attach/testdata/attachtest"
	"github.com/xhd2015/doctest/session"
)

type Request = attachtest.Request
type Response = attachtest.Response

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	return attachtest.Run(t, d, req)
}
```
