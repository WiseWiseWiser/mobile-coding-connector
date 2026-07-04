# Scenario

**Feature**: menu-bar Open in Browser label formatting (Go spec for Swift client)

```
browser preference -> FormatOpenInBrowserLabel -> menu item label
```

## Preconditions

1. `macosapp/browser` exports `FormatOpenInBrowserLabel(browser string) string`.
2. No subprocess or HTTP — pure function calls.

## Steps

1. Leaf `Setup` sets `Browser`.
2. Root `Run` calls `FormatOpenInBrowserLabel`.
3. Leaf `Assert` checks exact label text.

## Context

Swift `OpenInBrowserLabelFormatter` mirrors this contract. UI behavior is manual.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```