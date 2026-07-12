# Scenario

**Feature**: detect Skip selected after CSI Down (verify-before-Enter)

```
02-skip-selected.snapshot.txt -> blocking menu, selection SKIP
```

## Preconditions

Signed fixture `02-skip-selected.snapshot.txt` (≡ `02a-csi-down-x1`).

## Steps

1. `FixtureFile=02-skip-selected.snapshot.txt`.

## Context

PROTOCOL step `select_skip` assert-before-Enter: `›` on `2. Skip`, not on Update now.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "02-skip-selected.snapshot.txt"
	return nil
}
```
