# Scenario

**Feature**: sidebar bookmarks

```
sidebar-bookmarks -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.SidebarID = "bookmarks"
	return nil
}
```
