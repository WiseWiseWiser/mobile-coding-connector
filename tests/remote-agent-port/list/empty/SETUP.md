# Scenario

**Feature**: port list with no listeners

```
empty /api/ports/local -> port list -> empty message
```

## Steps

1. LocalPorts empty; Args `port list`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "list")
	req.LocalPorts = nil
	return nil
}
```
