# Scenario

**Feature**: GET inventory cache / refresh / stream / disk last-good

```
GET /inventory -> last-good cache (no layout probe)
GET ?refresh=1 -> full deep recapture
GET /inventory/stream (cold) -> desktop seed then full capture
GET /inventory/stream (warm) -> last-good then incremental layout-diff
file/ -> complete last-good on CachePath; new Handler loads file
```

## Context

GET leaves keep `Op=inventory`. Stream leaves set `Op=stream`.
`DoSecondGET` warms daemon memory via GET before the measured request.
`FirstSnapAB` / `UseTwoWindowSnap` / `SecondSnapAOnly` / `SecondSnapB` select fixture layouts.
`LayoutCalls` vs `CaptureCalls` distinguish layout-diff from deep capture.
`SessionACwd` detects a full recapture of a known session ID.
`file/` leaves use `SeedCacheJSON` / `CacheFile*` / `DoGETAfterStream` for disk last-good.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = true
	req.WindowSpace = 1
	return nil
}
```
