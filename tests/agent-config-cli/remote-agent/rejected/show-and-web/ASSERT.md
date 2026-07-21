## Expected

1. Non-zero exit within timeout (must not start UI and hang).
2. Error mentions both `--show` and `--web` and/or mutual exclusion.
3. No `Config UI running` banner.

## Side Effects

None.

## Errors

Exit 0, hang, or starts UI.

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExitNonZero(t, resp)
	assertNoConfigUI(t, resp)
	lower := combinedLower(resp)
	if !strings.Contains(lower, "show") || !strings.Contains(lower, "web") {
		t.Fatalf("expected mutual-exclusion error mentioning show and web; combined:\n%s", resp.Combined)
	}
}
```
