## Expected

1. `BackupNowNotGatedOnEnabled` is true (disabled only endpoint/running; not `backupEnabled`).

## Side Effects

- None (read-only source inspection).

## Errors

- `.disabled(!state.backupEnabled || …)` or similar enable gate on Backup Now.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.BackupNowNotGatedOnEnabled {
		t.Fatalf("Backup Now appears gated on backupEnabled (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
