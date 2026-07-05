# macOS Launch Environment Doctests

Pure-function tests for `macosapp/launchenv` — the Go spec mirrored by Swift
`DaemonManager` when spawning `ai-critic keep-alive` from the menu-bar app.

# DSN (Domain Specific Notion)

**Participants**

- **KeepAliveEnv (`macosapp/launchenv`)** — builds child-process environment for
  keep-alive: suppress browser auto-open only (usage fetch is in-process; no
  bundled `codex-show-status` / `grok-show-status` paths).
- **DaemonManager (Swift)** — mirrors the same env keys when launching keep-alive.
- **Test harness** — invokes `KeepAliveEnv(binaryDir)` with leaf-provided bundle paths;
  no subprocess spawn.

**Behaviors**

- `KeepAliveEnv(dir)` always sets `AI_CRITIC_NO_OPEN_BROWSER=1`.
- `KeepAliveEnv(dir)` does **not** set `CODEX_SHOW_STATUS_BIN` or `GROK_SHOW_USAGE_BIN`.

## Version

0.0.2

## Decision Tree

```
[launch env]
 |
 +-- keep-alive/                      (GROUP)  keep-alive child env
      +-- no-open-browser/            (LEAF)   AI_CRITIC_NO_OPEN_BROWSER=1
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `keep-alive/no-open-browser` | Env contains `AI_CRITIC_NO_OPEN_BROWSER=1` |

## Parameter Coverage

| Leaf | binaryDir | Asserted keys |
|------|-----------|---------------|
| no-open-browser | `/app/Contents/MacOS` | `AI_CRITIC_NO_OPEN_BROWSER` |

## How to Run

```sh
doctest vet ./tests/macos-launch-env
doctest test ./tests/macos-launch-env/...
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/launchenv"
)

type Request struct {
	BinaryDir string
}

type Response struct {
	Env map[string]string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	return &Response{
		Env: launchenv.KeepAliveEnv(req.BinaryDir),
	}, nil
}
```