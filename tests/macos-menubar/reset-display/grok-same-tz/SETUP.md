# Scenario

**Feature**: grok PT reset displayed in same timezone (PDT)

```
FormatResetDisplay("July 9, 17:55 PT", now=Jul 6 16:55 PDT) -> "July 9, 17:55"
```

## Preconditions

`now` is in Pacific (PDT); reset wall clock matches local display.

## Steps

1. Set grok reset and PDT `now`.

## Context

REQUIREMENT scenario: PT reset in PDT strips timezone suffix, keeps clock.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 9, 17:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```