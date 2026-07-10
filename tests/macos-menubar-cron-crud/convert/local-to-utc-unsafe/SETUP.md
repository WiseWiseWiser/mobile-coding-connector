# Scenario

**Feature**: unsafe local cron (ranges/lists) is rejected

```
# complex local expr with hour range + DOW: "0 9-17 * * 1-5"
ConvertLocalCronToUTC -> error (user must treat as UTC / fix expr)
```

## Preconditions

1. Expression `0 9-17 * * 1-5` is not safe (ranges).
2. Conversion must refuse regardless of TZ.

## Steps

1. Set unsafe local expr.

## Context

REQUIREMENT leaf: `convert/local-to-utc-unsafe` (optional).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConvertLeaf = "local-to-utc-unsafe"
	req.LocalExpr = "0 9-17 * * 1-5"
	req.TZName = "Etc/GMT-8"
	return nil
}
```
