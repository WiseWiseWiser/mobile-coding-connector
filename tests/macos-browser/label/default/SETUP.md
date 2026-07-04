# Scenario

**Feature**: default browser preference shows plain label

```
FormatOpenInBrowserLabel("default") -> "Open in Browser"
```

## Preconditions

Default preference selected.

## Steps

1. Set `Browser` to `default`.

## Context

REQUIREMENT leaf: `label/default`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Browser = "default"
	return nil
}
```