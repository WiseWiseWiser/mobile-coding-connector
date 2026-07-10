## Expected

1. `ShowEnable` is `false` (menu offers **Disable**, not Enable).

## Errors

- Enable shown for an already-enabled cron task.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ShowEnable {
		t.Fatal("ShowEnable = true, want false for enabled cron task")
	}
}
```
