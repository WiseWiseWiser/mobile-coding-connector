# Scenario

**Bug**: `go build ./cmd/remote-agent` fails with missing WS helper symbols

```
# remote-agent compile
go build ./cmd/remote-agent -> exit 0 (exec.go delegates to dot-pkgs ptywrap/client)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "remote-agent-build"
	return nil
}
```