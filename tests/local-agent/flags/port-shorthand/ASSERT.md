## Expected

1. Exit code 0.
2. Stdout reports `Status: ok` and references `http://localhost:` with `resp.ServerPort`.

## Side Effects

Server started on ephemeral port; `local-agent` targeted that port via `--port`.

## Errors

- Ping failure or wrong server URL in output.

## Exit Code

0.

```go
import (
	"fmt"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	wantHost := fmt.Sprintf("http://localhost:%d", resp.ServerPort)
	if !strings.Contains(resp.Stdout, wantHost) {
		t.Fatalf("stdout should show server %s; got:\n%s", wantHost, resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "Status: ok") {
		t.Fatalf("expected successful ping; stdout:\n%s", resp.Stdout)
	}
}
```