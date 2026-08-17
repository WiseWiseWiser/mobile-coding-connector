# Scenario

**Feature**: live CLI attach joins in place (no alt-screen reset, no scrollback dump)

```
writer held -> AttachWithIO(TerminalAttachConnectOptions)
  -> first frame must not contain \e[?1049l or \e[2J
  -> typed echo still appears
```

## Steps

1. Same writer-held attach as `echo-reaches-pty`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "writer-held-echo"
	req.Marker = "CLI_ATTACH_INPLACE"
	return nil
}
```
