## Expected

1. `NoActivateInProgressWindow` is true.
2. `BackupProgressWindow.swift` contains neither `activate(ignoringOtherApps:` nor
   `NSApp.activate`.

## Side Effects

- None (read-only source inspection).

## Errors

- Any `NSApp.activate` / `ignoringOtherApps` in the progress window open path.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.NoActivateInProgressWindow {
		t.Fatalf("BackupProgressWindow must not call NSApp.activate / activate(ignoringOtherApps:) (source: %s)", resp.ProgressWindowSource)
	}
}
```
