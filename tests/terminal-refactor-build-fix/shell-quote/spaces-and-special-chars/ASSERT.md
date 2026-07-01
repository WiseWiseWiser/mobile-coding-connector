## Expected

- Both inputs produce distinct quoted forms.
- Each quoted form round-trips via `sh -c` and resists adjacent-token injection.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range req.ShellQuoteInputs {
		quoted, ok := resp.ShellQuoteOutputs[input]
		if !ok {
			t.Fatalf("missing quoted output for %q", input)
		}
		if quoted == "" {
			t.Fatalf("empty quoted output for %q", input)
		}
	}
}
```