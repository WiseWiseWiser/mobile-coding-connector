## Expected

1. `HasLocalCronMenu` is true.
2. `LocalAccessibilityID` is true (`cron-menu`).

## Side Effects

- None (read-only source inspection).

## Errors

- Cron menu missing on local app, or wrong accessibility id.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasLocalCronMenu {
		t.Fatalf("local app missing Menu(\"Cron\") (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.LocalAccessibilityID {
		t.Fatalf("local app missing accessibility id cron-menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
