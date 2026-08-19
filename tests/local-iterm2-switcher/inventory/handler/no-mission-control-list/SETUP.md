# Scenario

**Feature**: inventory source never calls space.List / Mission Control

```
read server/localiterm2/*.go -> no spacelib.List / space.List
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "no_list_source"
	return nil
}
```
