# Scenario

**Feature**: full new → attach → echo → exited-refuse journey

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "tty-watch-journey"
	return nil
}
```
