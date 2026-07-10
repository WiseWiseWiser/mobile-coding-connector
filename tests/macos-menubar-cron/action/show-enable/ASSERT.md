## Expected

1. `ShowEnable` is `true` (menu offers **Enable**, not Disable).

## Errors

- Disable action shown for a disabled cron task.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShowEnable {
		t.Fatal("ShowEnable = false, want true for disabled cron task")
	}
}
```
