# Scenario

**Feature**: parseable grok reset → comma-prefixed days suffix

```
FormatResetSuffix("July 9, 16:55 PT", now=Jul 6 16:55 PDT) -> ", left 3d"
```

## Preconditions

Same inputs as `relative-time/three-days`.

## Steps

1. Set grok reset and fixed `now`.

## Context

REQUIREMENT leaf: `dropdown/reset-suffix/grok-three-days`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 9, 16:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```