## Expected

1. Exit code 0.
2. Stdout contains `Auth: OK`.

## Side Effects

Config file read; token not passed on CLI.

## Errors

- Unauthorized when saved token should match server credentials.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "Auth: OK") {
		t.Fatalf("expected Auth: OK using saved token; stdout:\n%s", resp.Stdout)
	}
}
```