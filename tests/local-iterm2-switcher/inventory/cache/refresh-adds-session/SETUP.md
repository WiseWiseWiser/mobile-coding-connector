# Scenario

**Feature**: ?refresh=1 recaptures and can add a session

```
GET /inventory -> sess-a
GET ?refresh=1 -> second snap includes sess-b
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.DoRefreshGET = true
	req.SecondSnapB = true
	return nil
}
```
