## Expected

1. `FlushIntervalInBand` is true (timer/interval token in 100–200 ms band).

## Side Effects

- None (read-only source inspection).

## Errors

- Immediate per-event UI append with no flush interval.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.FlushIntervalInBand {
		t.Fatalf("expected flush timer/interval in 100–200ms band in BackupProgressWindow (source: %s)", resp.ProgressWindowSource)
	}
}
```
