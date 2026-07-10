## Expected

1. `NewCronTaskAtBottom` is true (New Cron Task… after Divider / last in Cron menu).

## Side Effects

- None (read-only source inspection).

## Errors

- New Cron Task… at top or interleaved with per-task menus.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.NewCronTaskAtBottom {
		t.Fatalf("New Cron Task… not at bottom of Cron menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
