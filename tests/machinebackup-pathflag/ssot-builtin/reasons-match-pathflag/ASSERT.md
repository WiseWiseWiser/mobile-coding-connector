## Expected

1. No Run error.
2. `ReasonMismatches` is empty.
3. Every shared catalog path has the pathflag/product golden reason.

## Side Effects

- None.

## Errors

- Any reason mismatch or missing path fails.

## Exit Code

- N/A

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(resp.ReasonMismatches) > 0 {
		t.Fatalf("BuiltinExclusionConfig reason mismatches:\n  %s", strings.Join(resp.ReasonMismatches, "\n  "))
	}
	if !resp.HasPath {
		t.Fatal("HasPath=false, want all reasons aligned")
	}
}
```
