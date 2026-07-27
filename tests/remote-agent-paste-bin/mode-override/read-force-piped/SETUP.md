# Scenario

**Feature**: --read overrides piped stdin write detection

```
# seed scratch + pipe junk + --read -> stdout seed, API unchanged
piped junk + --read -> GET scratch (PUT skipped)
```

## Preconditions

Scratch seeded; piped bytes must not overwrite seed.

## Steps

1. `seedScratch(req, forceReadSeedContent, seededMetaUpdatedAt)`.
2. `setReadForcePiped(req, []byte(forceReadIgnoredPipe))`.

## Context

REQUIREMENT leaf: `read-force-piped`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	seedScratch(req, forceReadSeedContent, seededMetaUpdatedAt)
	setReadForcePiped(t, req, []byte(forceReadIgnoredPipe))
	return nil
}
```