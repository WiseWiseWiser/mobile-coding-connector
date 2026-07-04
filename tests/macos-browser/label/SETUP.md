# Scenario

**Feature**: browser preference drives Open in Browser menu label

```
FormatOpenInBrowserLabel(browser) -> label string
```

## Preconditions

Leaf sets the stored browser preference value.

## Steps

1. Set `Browser` per requirement table.
2. Assert exact label from `FormatOpenInBrowserLabel`.

## Context

REQUIREMENT group: `label/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Leaves supply browser-specific inputs; reset shared field first.
	req.Browser = ""
	return nil
}
```