## Expected

1. `StatusTitle` is exactly `Status: On · Running`.

## Errors

- Progress percent, spinner text, or missing On prefix.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: On · Running"
	if resp.StatusTitle != want {
		t.Fatalf("StatusTitle = %q, want %q", resp.StatusTitle, want)
	}
}
```
