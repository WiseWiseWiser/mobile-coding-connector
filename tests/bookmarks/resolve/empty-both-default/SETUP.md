# Scenario

**Feature**: both empty resolve to default

```
ResolveBrowser("","") -> "default"
```

## Preconditions

1. Both empty.

## Steps

1. Resolve → default.

## Context

Final fallback.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.BookmarkBrowser = ""
	req.GlobalDefault = ""
	return nil
}
```
