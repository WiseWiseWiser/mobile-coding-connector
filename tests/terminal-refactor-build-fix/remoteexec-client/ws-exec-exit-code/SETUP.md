# Scenario

**Feature**: client captures binary stdout and returns remote exit code

```
# exit code path
fake WS -> binary stdout -> {"type":"exit","code":42} -> client exit 42
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "ws-exec-exit-code"
	req.WSExecArgv = []string{"echo", "hi"}
	req.WSStdoutPayload = "hello from remote\n"
	req.WSExitCode = 42
	return nil
}
```