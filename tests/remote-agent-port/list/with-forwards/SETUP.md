# Scenario

**Feature**: port list --forwards includes persistent forwards

```
seeded forward + listeners -> port list --forwards -> forward URL present
```

## Steps

1. Seed listener + forward; Args `port list --forwards`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "list", "--forwards")
	seedOneListener(req)
	seedOneForward(req)
	return nil
}
```
