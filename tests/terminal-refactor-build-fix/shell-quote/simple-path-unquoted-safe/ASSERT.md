## Expected

- `ShellQuote("/tmp/ai-critic")` round-trips through `sh -c` unchanged.
- Quoted form is a valid shell token (may use `'...'` wrapping per agent-pro semantics).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShellRoundTripOK {
		t.Fatalf("shell round-trip failed for %q (quoted %q)", req.ShellQuoteInput, resp.ShellQuoteOutput)
	}
	if resp.ShellQuoteOutput == "" {
		t.Fatal("expected non-empty quoted output")
	}
}
```