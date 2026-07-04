# Scenario

**Feature**: unknown browser value falls back to plain label

```
FormatOpenInBrowserLabel("safari") -> "Open in Browser"
```

## Preconditions

Stored value is not one of the supported preferences.

## Steps

1. Set `Browser` to `safari`.

## Context

REQUIREMENT leaf: `label/unknown`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Browser = "safari"
	return nil
}
```