# Scenario

**Feature**: grok PT reset crosses calendar day in JST

```
FormatResetDisplay("July 9, 17:55 PT", now=Jul 6 16:55 JST) -> "July 10, 09:55"
```

## Preconditions

`now` is in Asia/Tokyo (JST, UTC+9); PT July 9 17:55 PDT → July 10 09:55 JST.

## Steps

1. Set grok reset and JST `now` via RFC3339 offset `+09:00`.

## Context

REQUIREMENT scenario: PT reset date rollover when displayed in JST.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 9, 17:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00+09:00"
	return nil
}
```