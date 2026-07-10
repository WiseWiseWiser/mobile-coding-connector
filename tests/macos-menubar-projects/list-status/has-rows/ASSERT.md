## Expected

1. `Label` is exactly empty string `""` (show project menus, not a status row).

## Errors

- Replacing existing rows with Loading… / empty / failed when count > 0.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "" {
		t.Fatalf("Label = %q, want empty (count>0 shows project menus)", resp.Label)
	}
}
```
