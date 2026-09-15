# Scenario

**Feature**: reading the cached response without refetching

```
SeedReady(reset instants) + fixed clock -> Get() -> fresh window countdowns
```

## Preconditions

1. `TestExported_SeedReady` and `TestExported_SetNow` inject the cache and clock.

## Steps

1. Set `Op=countdown` with two clock instants.

## Context

The menu bar polls every 30 seconds but the provider is fetched every 10
minutes, so the countdown must advance on its own.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "countdown"
	return nil
}
```
