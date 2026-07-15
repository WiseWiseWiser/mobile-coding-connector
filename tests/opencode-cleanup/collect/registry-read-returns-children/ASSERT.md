## Expected

1. `CollectErr` is nil.
2. `CollectedPIDs` contains the fixture PID (`FixturePID` / current process).
3. `RegistryRaw` is non-empty and mentions `headless-agent`.

## Errors

- Missing helper or unread registry fails the test.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.CollectErr != nil {
		t.Fatalf("CollectOpencodeServePIDs error: %v", resp.CollectErr)
	}
	want := req.FixturePID
	found := false
	for _, pid := range resp.CollectedPIDs {
		if pid == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("CollectedPIDs %v does not contain fixture pid %d", resp.CollectedPIDs, want)
	}
	if !strings.Contains(resp.RegistryRaw, "headless-agent") {
		t.Fatalf("registry raw missing headless-agent: %q", resp.RegistryRaw)
	}
}
```
