# Scenario

**Feature**: Firefox preference shows browser suffix

```
FormatOpenInBrowserLabel("firefox") -> "Open in Browser(Firefox)"
```

## Preconditions

Firefox selected in Settings.

## Steps

1. Set `Browser` to `firefox`.

## Context

REQUIREMENT leaf: `label/firefox`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Browser = "firefox"
	return nil
}
```