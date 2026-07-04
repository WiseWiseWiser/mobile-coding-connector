# Scenario

**Feature**: Opera preference shows browser suffix

```
FormatOpenInBrowserLabel("opera") -> "Open in Browser(Opera)"
```

## Preconditions

Opera selected in Settings.

## Steps

1. Set `Browser` to `opera`.

## Context

REQUIREMENT leaf: `label/opera`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Browser = "opera"
	return nil
}
```