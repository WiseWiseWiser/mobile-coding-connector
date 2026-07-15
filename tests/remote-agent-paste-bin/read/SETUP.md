# Scenario

**Feature**: paste-bin read mode (TTY stdin)

```
# scratch.json on server -> remote-agent paste-bin -> content on stdout
GET /api/file-transfer/scratch <- remote-agent (TTY) -> stdout bytes
```

## Preconditions

Leaves in this group use TTY stdin (`PipedStdin` unset) unless a leaf adds flags.

## Steps

1. Seed or delete `scratch.json` under `configHome/file-transfer/`.
2. Run `remote-agent paste-bin` with optional `--json` or `--meta`.
3. Assert stdout/stderr match read-mode contracts.

## Context

Read mode: GET scratch, emit `content` verbatim on stdout; empty scratch is silent.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```