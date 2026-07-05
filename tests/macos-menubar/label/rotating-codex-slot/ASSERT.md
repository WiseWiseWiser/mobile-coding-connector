## Expected

1. `Label` is exactly `Codex 58%`.

## Errors

- Rotating index 1 did not select codex slot.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Codex 58%" {
		t.Fatalf("label = %q, want %q", resp.Label, "Codex 58%")
	}
}
```