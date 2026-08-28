# Scenario

**Feature**: injected success fetcher → service ready

```
FetchMode=success -> status ready + limits
```

## Steps

1. `FetchMode=success`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FetchMode = "success"
	return nil
}
```
