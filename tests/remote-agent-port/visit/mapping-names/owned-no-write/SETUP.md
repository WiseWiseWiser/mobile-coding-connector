# Scenario

**Feature**: owned ad-hoc visit does not write mapping-names

```
seed names file -> Start(owned) -> file unchanged
```

## Steps

1. Seed mapping {"9999":"keep.example.com"}; Start owned.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-mapping"
	req.Port = defaultTestPort
	req.Provider = "owned"
	enableOwnedQuick(req, true, true)
	req.Idle = 10 * time.Minute
	req.SeedMappingNames = map[string]string{
		"9999": "keep.example.com",
	}
	return nil
}
```
