# Scenario

**Feature**: scrollToEndOfDocument runs on flush only, not every raw line

```
# required
flushPending/flush* -> scrollToEndOfDocument once

# banned
appendOnMain: string += …; scrollToEndOfDocument  # every SSE line
```

## Preconditions

A flush-named path calls `scrollToEnd`; immediate `string +=` … `scrollToEnd`
pairing is absent.

## Steps

1. ClientLeaf=scroll-on-flush.

## Context

REQUIREMENT #7; RED on current appendOnMain which scrolls every line.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "scroll-on-flush"
	return nil
}
```
