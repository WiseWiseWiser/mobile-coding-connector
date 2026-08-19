# Scenario

**Feature**: sidebar desktop

```
sidebar-desktop -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.SidebarID = "desktop:1"
	req.SpaceIndex = 1
	return nil
}
```
