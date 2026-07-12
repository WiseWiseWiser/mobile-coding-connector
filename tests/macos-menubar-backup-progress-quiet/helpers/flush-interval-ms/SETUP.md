# Scenario

**Feature**: sealed flush interval constant in 100–200 ms band

```
menubar.BackupProgressFlushIntervalMilliseconds -> 150  # or any in [100,200]
```

## Preconditions

Exported int constant on `macosapp/menubar`.

## Steps

1. Op=helper_flush_interval.

## Context

REQUIREMENT helper constant; documents Swift timer band.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "helper_flush_interval"
	return nil
}
```
