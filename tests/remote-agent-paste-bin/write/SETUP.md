# Scenario

**Feature**: paste-bin write mode (piped stdin)

```
# stdin bytes -> remote-agent paste-bin -> PUT scratch + stderr saved/preview
piped stdin -> remote-agent paste-bin -> PUT /api/file-transfer/scratch
```

## Preconditions

Leaves attach piped stdin via `setWritePipe` (including empty pipe for clears).

## Steps

1. Reset scratch when the leaf models a fresh write target.
2. Pipe payload bytes into `remote-agent paste-bin` with optional output flags.
3. Assert stderr `saved N bytes`, optional stdout echo, and API scratch state.

## Context

Write mode: raw stdin bytes overwrite scratch; empty pipe clears content.

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