# Scenario

**Bug**: `go build ./` fails with undefined `terminal.ShellQuote`

```
# server compile
go build ./ -> exit 0 (run/* uses dot-pkgs ptywrap.ShellQuote)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "server-build"
	return nil
}
```