## Expected

1. Non-zero exit.
2. Error text indicates `--json` requires `--show` (or equivalent wording).
3. No Config UI banner; no successful full config dump as the only outcome.

## Side Effects

None.

## Errors

Exit 0 or silent success.

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
	if !strings.Contains(lower, "json") {
		t.Fatalf("error should mention json; combined:\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "show") {
		t.Fatalf("error should mention need for --show; combined:\n%s", resp.Combined)
	}
}
```
