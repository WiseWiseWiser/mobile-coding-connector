# Scenario

**Bug**: broken go build targets after terminal refactor

```
# compile checks
buildfixtest -> go build -o /dev/null <pkg> -> exit code + combined output
```

## Preconditions

- Module root contains `go.mod` and broken call sites until implementer fixes them.

## Steps

1. Leaf sets `req.Phase` to `remote-agent-build` or `server-build`.
2. Harness runs `go build -o /dev/null` for the corresponding package.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/tests/terminal-refactor-build-fix/testdata/buildfixtest"
)

func Setup(t *testing.T, req *Request) error {
	root := buildfixtest.ModuleRoot(t)
	t.Logf("compile checks run against module root %s", root)
	return nil
}
```