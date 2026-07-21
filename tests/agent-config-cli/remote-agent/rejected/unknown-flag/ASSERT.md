## Expected

1. Non-zero exit.
2. Error message about the unknown/invalid flag or usage; ideally points to help.
3. No Config UI.

## Side Effects

None.

## Errors

Exit 0 or starts UI.

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
	// Accept flag/unknown/unrecognized/help-style wording from flag.Parse or custom errors.
	ok := strings.Contains(lower, "flag") ||
		strings.Contains(lower, "unknown") ||
		strings.Contains(lower, "unrecognized") ||
		strings.Contains(lower, "not-a-real-flag") ||
		strings.Contains(lower, "help") ||
		strings.Contains(lower, "usage")
	if !ok {
		t.Fatalf("expected unknown-flag style error; combined:\n%s", resp.Combined)
	}
}
```
