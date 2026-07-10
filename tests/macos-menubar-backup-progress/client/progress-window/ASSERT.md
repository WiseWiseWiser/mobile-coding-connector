## Expected

1. `HasProgressWindow` is true (`BackupProgressWindow` or openBackupProgress helper).

## Side Effects

- None (read-only source inspection).

## Errors

- Only reusing log-path LogStreamWindow without a backup progress session API.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasProgressWindow {
		t.Fatalf("missing BackupProgressWindow / open helper (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
