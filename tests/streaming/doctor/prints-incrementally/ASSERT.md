## Expected

1. A line containing `configuration load` was recorded.
2. A line containing `upstream proxy fetch` was recorded.
3. Timestamp of `configuration load` is ≥150ms before `upstream proxy fetch`.
4. `configuration load` line index is less than `upstream proxy fetch` line index in `StdoutLines`.

## Side Effects

Upstream fetch delay hook cleared on cleanup.

## Errors

- Both lines appear only at process end (same timestamp / adjacent with no gap).
- `configuration load` line missing or after fetch line.

```go
import (
	"strings"
	"testing"
	"time"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	cfgTime, okCfg := lineTime(resp.LineRecords, "configuration load")
	fetchTime, okFetch := lineTime(resp.LineRecords, "upstream proxy fetch")
	if !okCfg {
		t.Fatalf("no timestamped line for configuration load; stdout:\n%s", resp.Stdout)
	}
	if !okFetch {
		t.Fatalf("no timestamped line for upstream proxy fetch; stdout:\n%s", resp.Stdout)
	}
	gap := fetchTime.Sub(cfgTime)
	if gap < 150*time.Millisecond {
		t.Fatalf("configuration load at %v, upstream fetch at %v (gap %v); want ≥150ms incremental gap",
			cfgTime, fetchTime, gap)
	}
	cfgIdx, fetchIdx := -1, -1
	for i, l := range resp.StdoutLines {
		if strings.Contains(strings.ToLower(l), "configuration load") {
			cfgIdx = i
		}
		if strings.Contains(strings.ToLower(l), "upstream proxy fetch") {
			fetchIdx = i
		}
	}
	if cfgIdx < 0 || fetchIdx < 0 || cfgIdx >= fetchIdx {
		t.Fatalf("line order wrong: configIdx=%d fetchIdx=%d; stdout:\n%s", cfgIdx, fetchIdx, resp.Stdout)
	}
}
```
