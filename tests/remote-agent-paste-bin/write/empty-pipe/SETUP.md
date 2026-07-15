# Scenario

**Feature**: empty pipe clears scratch

```
# stale scratch + empty stdin pipe -> PUT content "" -> saved 0 bytes
seed stale -> empty pipe -> scratch cleared
```

## Preconditions

Scratch pre-seeded with stale content.

## Steps

1. `seedScratch(req, staleScratchContent, seededMetaUpdatedAt)`.
2. `setWritePipe(req, []byte{})` — piped empty payload.

## Context

REQUIREMENT leaf: `write-empty-pipe`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedScratch(req, staleScratchContent, seededMetaUpdatedAt)
	setWritePipe(t, req, []byte{})
	return nil
}
```