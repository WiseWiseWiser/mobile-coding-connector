# Scenario

**Feature**: NSScrollView + NSTextView (documentView) still used

```
BackupProgressWindow
  -> NSScrollView
  -> scroll.documentView = NSTextView
```

## Preconditions

Tokens `NSScrollView`, `NSTextView`, and `documentView` present in
`BackupProgressWindow.swift`.

## Steps

1. ClientLeaf=scroll-and-text-view.

## Context

REQUIREMENT #10; expected GREEN on current layout.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "scroll-and-text-view"
	return nil
}
```
