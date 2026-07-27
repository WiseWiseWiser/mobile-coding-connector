# Scenario

**Feature**: empty bookmark browser inherits global default

```
ResolveBrowser("","chrome") -> "chrome"
```

## Preconditions

1. Bookmark empty; global chrome.

## Steps

1. Resolve.
2. Expect chrome.

## Context

Inherit path.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.BookmarkBrowser = ""
	req.GlobalDefault = "chrome"
	return nil
}
```
