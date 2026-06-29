## Expected

1. Exit code 0.
2. Stdout shows `http://localhost:<ServerPort>` and successful ping.

## Side Effects

None beyond subprocess lifecycle.

## Errors

- Default resolution pointed at wrong host/port.

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	want := fmt.Sprintf("http://localhost:%d", resp.ServerPort)
	if !strings.Contains(resp.Stdout, want) {
		t.Fatalf("expected default URL %s in stdout:\n%s", want, resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "Status: ok") {
		t.Fatalf("expected pong; stdout:\n%s", resp.Stdout)
	}
}
```