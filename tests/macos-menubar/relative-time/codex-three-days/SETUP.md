# Scenario

**Feature**: codex reset format ≥24h away → days unit

```
FormatTimeLeft("08:00 on 9 Jul", now=Jul 6 08:00 PDT) -> "left 3d"
```

## Preconditions

Codex reset string without timezone; uses `now.Location()`.

## Steps

1. Set reset and fixed `now`.

## Context

REQUIREMENT leaf: `relative-time/codex-three-days`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "08:00 on 9 Jul"
	req.NowRFC3339 = "2026-07-06T08:00:00-07:00"
	return nil
}
```