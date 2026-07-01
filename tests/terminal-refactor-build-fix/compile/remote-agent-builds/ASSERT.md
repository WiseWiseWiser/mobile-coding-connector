## Expected

- `go build -o /dev/null ./cmd/remote-agent` exits 0 with no compile errors.

## Exit Code

N/A (build subprocess exit 0).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("remote-agent build failed: %v\n%s", err, resp.BuildOutput)
	}
	if resp.BuildExitCode != 0 {
		t.Fatalf("expected build exit 0, got %d\n%s", resp.BuildExitCode, resp.BuildOutput)
	}
	if strings.Contains(resp.BuildOutput, "undefined:") {
		t.Fatalf("compile errors remain:\n%s", resp.BuildOutput)
	}
}
```