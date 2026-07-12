## Expected

1. `QuietOrderFront` is true (`orderFrontRegardless` or non-makeKey `orderFront(`).

## Side Effects

- None (read-only source inspection).

## Errors

- Only `makeKeyAndOrderFront` without a quiet order-front API.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.QuietOrderFront {
		t.Fatalf("expected orderFrontRegardless or quiet orderFront( in BackupProgressWindow (source: %s)", resp.ProgressWindowSource)
	}
}
```
