# Scenario

**Feature**: window countdowns advance between fetches

```
SeedReady(5h reset = base+2h) -> Get@base: left 2h -> Get@base+1h: left 1h
```

## Preconditions

1. The cache is seeded with resets 2h and 2d6h after `base`.

## Steps

1. Set `Op=countdown` with `NowRFC3339=base` and `ThenRFC3339=base+1h`.

## Context

The app polls every 30 seconds but the provider is fetched every 10 minutes, so the
countdown has to come from the cached reset instants.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "countdown"
	req.NowRFC3339 = "2026-09-15T12:00:00Z"
	req.ThenRFC3339 = "2026-09-15T13:00:00Z"
	return nil
}
```
