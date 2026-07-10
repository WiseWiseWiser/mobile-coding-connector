# Scenario

**Feature**: safe local cron converts to UTC on save

```
# TZ=Etc/GMT-8 (fixed UTC+8), local "0 9 * * *"
ConvertLocalCronToUTC -> "0 1 * * *" (9 - 8 = 1)
```

## Preconditions

1. Fixed-offset timezone without DST: `Etc/GMT-8` means UTC+8.
2. Simple expr `M H * * *` is safe to convert.
3. Expected UTC hour: 9 - 8 = 1 → `0 1 * * *`.

## Steps

1. Set local expr and TZName.

## Context

REQUIREMENT leaf: `convert/local-to-utc-safe` (optional).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConvertLeaf = "local-to-utc-safe"
	req.LocalExpr = "0 9 * * *"
	req.TZName = "Etc/GMT-8"
	return nil
}
```
