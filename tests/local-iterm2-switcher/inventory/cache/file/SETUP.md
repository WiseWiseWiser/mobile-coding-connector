# Scenario

**Feature**: disk last-good inventory file (`Handler.CachePath`)

```
# complete last-good only (not seed / prefix frames)
Handler -> storeCache complete -> iterm-inventory-cache.json (temp CachePath)

# new process / new Handler, empty RAM
SeedCacheJSON | prior write -> CachePath
new Handler -> load file into RAM -> stream/GET from_cache (no Capture)

# cold file states
missing | corrupt CachePath -> cold seed then Capture (same as RAM-cold)

# iTerm down must not wipe durable last-good
ITermRunning=false -> GET/stream empty live -> file still has sess-a
```

## Context

Harness always sets `CachePath` under `t.TempDir()` (never `$HOME`).
`SeedCacheJSON` writes that path before constructing the Handler.
`NewHandlerAfterWrite` GETs on handler A then measures on a new Handler with the same path.
`DoGETAfterStream` runs GET without refresh after stream (shared CaptureCalls).
`CacheFile*` response fields probe the disk file after the request.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ITermRunning = true
	req.WindowSpace = 1
	return nil
}
```
