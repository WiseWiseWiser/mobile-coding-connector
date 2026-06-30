# ai-critic ptywrap Adapter Regression Doctests

Behavior-preserving regression tests after refactoring
`ai-critic-terminal/server/terminal` to thin HTTP adapter over shared `ptywrap`.

# DSN (Domain Specific Notion)

**Participants**

- **ai-critic-server** — starts with test config; registers terminal routes via
  ptywrap adapter (SSH browser mode stays in ai-critic only, not exercised here).
- **Terminal adapter** — delegates `/api/terminal` WS and `/api/terminal/sessions`
  REST to `ptywrap` library.
- **HTTP/WS test clients** — assert legacy query params and JSON response shapes.

**Behaviors**

- WS connect with `name` + `cwd` query creates shell session (ai-critic legacy).
- `GET /api/terminal/sessions` returns same paginated JSON shape as before refactor:
  `sessions`, `page`, `page_size`, `total`, `total_pages` with per-session fields
  `id`, `name`, `cwd`, `created_at`, `status`, `connected`.

## Version

0.0.2

## Decision Tree

```
[ptywrap adapter in ai-critic]
 |
 +-- websocket/
 |    |
 |    +-- create-with-name-cwd/       (LEAF)  WS name+cwd creates session
 |
 +-- rest/
      |
      +-- list-sessions-shape/        (LEAF)  list JSON shape unchanged
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `websocket/create-with-name-cwd` | WS `name`+`cwd` creates session; `session_id` returned |
| 2 | `rest/list-sessions-shape` | List API JSON matches pre-refactor schema |

## How to Run

```sh
doctest vet ./ai-critic-terminal/tests/ptywrap-adapter
doctest test ./ai-critic-terminal/tests/ptywrap-adapter/...
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/tests/ptywrap-adapter/adaptertest"
)

type Request = adaptertest.Request
type Response = adaptertest.Response

func Run(t *testing.T, req *Request) (*Response, error) {
	return adaptertest.Run(t, req)
}

func startAICriticServer(t *testing.T) (base string, port int, cleanup func()) {
	return adaptertest.StartAICriticServer(t)
}
```