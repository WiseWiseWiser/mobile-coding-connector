# Scenario

**Feature**: menu-bar pure label helpers for bookmarks submenu

```
# FormatEmptyBookmarksLabel / FormatBookmarkMenuTitle
Go helpers mirrored by Swift Bookmarks menu
```

## Preconditions

1. Mode label; pure functions in server/bookmarks (or shared package as documented).

## Steps

1. Leaf sets LabelKind.
2. Assert Label string.

## Context

Menu empty state and item titles.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "label"
	return nil
}
```
