## Expected

1. `StatusTitle` is exactly `Status: On · last 12m ago · next in 48m`.

## Errors

- Missing On, wrong relative units, or using hyphen/en-dash instead of ` · `.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: On · last 12m ago · next in 48m"
	if resp.StatusTitle != want {
		t.Fatalf("StatusTitle = %q, want %q", resp.StatusTitle, want)
	}
}
```
