## Expected

1. `FlushIntervalMs` is between 100 and 200 inclusive (canonical 150).
2. Constant name sealed: `BackupProgressFlushIntervalMilliseconds` in `macosapp/menubar`.

## Side Effects

- None (read-only menubar source inspection until pure API is callable).

## Errors

- Missing constant, or value outside the band (`FlushIntervalMs == -1` means missing).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FlushIntervalMs < 0 {
		t.Fatalf("BackupProgressFlushIntervalMilliseconds missing in macosapp/menubar (checked: %v)", resp.MenubarSourcesChecked)
	}
	if resp.FlushIntervalMs < flushIntervalMinMs || resp.FlushIntervalMs > flushIntervalMaxMs {
		t.Fatalf("FlushIntervalMs=%d, want in [%d,%d]", resp.FlushIntervalMs, flushIntervalMinMs, flushIntervalMaxMs)
	}
}
```
