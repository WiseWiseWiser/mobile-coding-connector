# Scenario

**Feature**: empty bookmarks menu label is No bookmarks

```
FormatEmptyBookmarksLabel() -> "No bookmarks"
```

## Preconditions

1. LabelKind empty.

## Steps

1. Call formatter.
2. Exact match.

## Context

Requirement empty UX.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.LabelKind = "empty"
	return nil
}
```
