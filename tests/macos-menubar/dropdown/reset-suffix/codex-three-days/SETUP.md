# Scenario

**Feature**: parseable codex reset → comma-prefixed days suffix

```
FormatResetSuffix("08:00 on 9 Jul", now=Jul 6 08:00 PDT) -> ", left 3d"
```

## Preconditions

Same inputs as `relative-time/codex-three-days`.

## Steps

1. Set codex reset and fixed `now`.

## Context

REQUIREMENT leaf: `dropdown/reset-suffix/codex-three-days`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "08:00 on 9 Jul"
	req.NowRFC3339 = "2026-07-06T08:00:00-07:00"
	return nil
}
```