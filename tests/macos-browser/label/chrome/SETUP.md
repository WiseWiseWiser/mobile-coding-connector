# Scenario

**Feature**: Chrome preference shows browser suffix

```
FormatOpenInBrowserLabel("chrome") -> "Open in Browser(Chrome)"
```

## Preconditions

Chrome selected in Settings.

## Steps

1. Set `Browser` to `chrome`.

## Context

REQUIREMENT leaf: `label/chrome`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Browser = "chrome"
	return nil
}
```