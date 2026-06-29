## Expected

- Exit code 0
- Stdout is exactly `No dirty projects found.`

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
	want := "No dirty projects found."
	if strings.TrimSpace(resp.Stdout) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(resp.Stdout), want)
	}
}
```