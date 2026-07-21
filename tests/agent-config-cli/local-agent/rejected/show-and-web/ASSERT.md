## Expected

1. Non-zero exit without hanging on UI.
2. Error mentions show and web.
3. No Config UI banner.

## Side Effects

None.

## Errors

Exit 0 or hang.

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
		t.Fatalf("expected mutual-exclusion error; combined:\n%s", resp.Combined)
	}
}
```
