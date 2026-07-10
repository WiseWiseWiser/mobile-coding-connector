## Expected

1. `ConvertOK` is false.
2. `ConvertErr` is non-empty.

## Errors

- Silently storing local expr as UTC; partial/wrong conversion of ranges.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ConvertOK {
		t.Fatalf("expected unsafe convert error, got %q", resp.ConvertedExpr)
	}
	if resp.ConvertErr == "" {
		t.Fatal("ConvertErr empty")
	}
}
```
