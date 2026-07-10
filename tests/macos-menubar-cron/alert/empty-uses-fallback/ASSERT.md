## Expected

1. `AlertMessage` is exactly `Task updated`.

## Errors

- Empty alert body; alternate fallback wording.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Task updated"
	if resp.AlertMessage != want {
		t.Fatalf("alert = %q, want %q", resp.AlertMessage, want)
	}
}
```
