## Expected

1. `HasRemoteCronMenu` is true.
2. `RemoteAccessibilityID` is true (`cron-menu`).

## Side Effects

- None (read-only source inspection).

## Errors

- Cron menu missing on remote app, or wrong accessibility id.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasRemoteCronMenu {
		t.Fatalf("remote app missing Menu(\"Cron\") (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.RemoteAccessibilityID {
		t.Fatalf("remote app missing accessibility id cron-menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
