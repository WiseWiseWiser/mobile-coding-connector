# Scenario

**Feature**: server rejects repo/git-run requests before spawning git

```
# POST /api/remote-agent/git/run with invalid dir or denied args
remote-agent git -C <dir> <args> -> API/CLI error, git not executed
```

## Preconditions

Valid server credentials; request reaches server git handler.

## Steps

1. Leaf prepares `dir` and `args` that fail server-side validation or allowlist.
2. Remote-agent surfaces error on stderr with non-zero exit.

## Context

Reuses `dir is not a git repository` validation from fetch/pull/push handlers.

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