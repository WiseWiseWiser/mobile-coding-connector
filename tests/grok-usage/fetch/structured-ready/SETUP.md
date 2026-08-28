# Scenario

**Feature**: bare-local success → structured reset_at / reset_display / time_left

```
FetchMode=success-no-tz -> ready + structured A+B fields
```

## Steps

1. Set `FetchMode=success-no-tz`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FetchMode = "success-no-tz"
	return nil
}
```
