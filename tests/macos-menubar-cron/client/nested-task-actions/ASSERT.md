## Expected

1. `HasNestedTaskMenu` is true.
2. `HasRunNow` is true.
3. `HasEnableDisable` is true.
4. `HasViewLogs` is true.
5. `HasHistoryDisabled` is true (History placeholder present and disabled).

## Side Effects

- None (read-only source inspection).

## Errors

- Flat list without nested per-task Menu; missing Run/Enable/Disable/Logs/History.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasNestedTaskMenu {
		t.Fatalf("missing nested per-task Cron Menu (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.HasRunNow {
		t.Fatalf("missing Run Now action (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.HasEnableDisable {
		t.Fatalf("missing Enable/Disable toggle (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.HasViewLogs {
		t.Fatalf("missing View Logs (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.HasHistoryDisabled {
		t.Fatalf("missing disabled History placeholder (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
