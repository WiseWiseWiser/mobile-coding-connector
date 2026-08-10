# Scenario

**Feature**: ring of 200 drops oldest after overflow

```
# capacity 200, publish 250
NewHub(200) -> Publish x250 -> Recent(250) len==200 (newest retained)
```

## Steps

1. RingSize=200, PublishCount=250, RecentN=250.
2. Run hub-publish bulk path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.RingSize = 200
	req.PublishCount = 250
	req.RecentN = 250
	return nil
}
```
