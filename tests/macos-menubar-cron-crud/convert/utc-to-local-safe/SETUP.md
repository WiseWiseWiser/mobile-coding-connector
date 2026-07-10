# Scenario

**Feature**: safe UTC cron converts to local for editor display

```
# TZ=Etc/GMT-8 (UTC+8), stored UTC "0 1 * * *"
ConvertUTCCronToLocal -> "0 9 * * *" (1 + 8 = 9)
```

## Preconditions

1. Edit-open path displays local wall time when conversion is safe.
2. Fixed-offset zone; simple `M H * * *` is safe.
3. Reverse of local-to-utc-safe.

## Steps

1. Set UTC expr and TZName.

## Context

REQUIREMENT: on edit open UTC→local when safe; pass-through if unsafe
(covered by UI when convert errors).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConvertLeaf = "utc-to-local-safe"
	req.UTCExpr = "0 1 * * *"
	req.TZName = "Etc/GMT-8"
	return nil
}
```
