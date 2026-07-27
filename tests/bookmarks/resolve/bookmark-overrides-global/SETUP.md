# Scenario

**Feature**: non-empty bookmark browser overrides global default

```
ResolveBrowser("firefox","chrome") -> "firefox"
```

## Preconditions

1. Bookmark firefox; global chrome.

## Steps

1. Call ResolveBrowser.
2. Expect firefox.

## Context

Per-bookmark preference wins.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.BookmarkBrowser = "firefox"
	req.GlobalDefault = "chrome"
	return nil
}
```
