# Scenario

**Feature**: service rm unknown target

```
service rm does-not-exist-svc -> non-zero error
```

## Steps

1. Empty seed.
2. CLI: `service rm does-not-exist-svc`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "service", "rm", "does-not-exist-svc")
	return nil
}
```
