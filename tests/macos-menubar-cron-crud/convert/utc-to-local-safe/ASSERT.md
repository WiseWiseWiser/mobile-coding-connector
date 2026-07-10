## Expected

1. `ConvertOK` is true.
2. `ConvertedExpr` is `0 9 * * *`.

## Errors

- Leaving UTC in form without conversion when safe; wrong hour arithmetic.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ConvertOK {
		t.Fatalf("convert failed: %s", resp.ConvertErr)
	}
	got := strings.Join(strings.Fields(resp.ConvertedExpr), " ")
	want := "0 9 * * *"
	if got != want {
		t.Fatalf("ConvertedExpr = %q, want %q", resp.ConvertedExpr, want)
	}
}
```
