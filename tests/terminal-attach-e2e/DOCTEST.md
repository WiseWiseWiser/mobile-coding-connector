# Terminal attach e2e (tty-watch)

Autonomous human journey: real `ai-critic-server` + `remote-agent` + `tty-watch`.
This is the gate that L2 httptest cannot replace. Default attach doctests stay
in `tests/terminal-cli-attach` (fast).

# DSN (Domain Specific Notion)

**Participants**

- **`script/verify-terminal-attach-e2e.sh`** — builds both binaries, starts an
  isolated server, drives `terminal new` / `attach` via tty-watch.
- **`tty-watch`** — real PTY (`run --detach`, `snapshot`, `send`, `kill`).

**Behaviors**

- `terminal new` shows a prompt.
- `terminal attach` of that running session snapshots a prompt (does not hang).
- Typed `echo VERIFY_ATTACH_MARKER` appears.
- First attach paint has no `\e[?1049l` / `\e[2J`.
- After writer close, attach errors `is exited` quickly.

## Version

0.0.2

## Decision Tree

```
tests/terminal-attach-e2e/
├── DOCTEST.md
├── SETUP.md
└── tty-watch-journey/     (LEAF) full script gate
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `tty-watch-journey` | `./script/verify-terminal-attach-e2e.sh` exit 0 |

## How to Run

```sh
./script/verify-terminal-attach-e2e.sh
doctest vet ./tests/terminal-attach-e2e
doctest test ./tests/terminal-attach-e2e --label e2e
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/tests/terminal-attach-e2e/testdata/e2etest"
	"github.com/xhd2015/doctest/session"
)

type Request = e2etest.Request
type Response = e2etest.Response

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	return e2etest.Run(t, d, req)
}
```
