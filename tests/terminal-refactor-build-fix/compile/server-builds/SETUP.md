# Scenario

**Bug**: `go build ./` fails with undefined `terminal.ShellQuote`

```
# server compile
go build ./ -> exit 0 (run/* uses dot-pkgs ptywrap.ShellQuote)
```

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "server-build"
	return nil
}
```