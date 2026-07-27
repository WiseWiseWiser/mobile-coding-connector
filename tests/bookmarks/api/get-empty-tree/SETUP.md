# Scenario

**Feature**: GET full tree when store is default empty

```
GET /api/bookmarks -> 200 {version:1, roots:[root…]}
```

## Preconditions

1. No seeds; APIOp get.

## Steps

1. GET.
2. Assert 200 and default root.

## Context

List baseline.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.APIOp = "get"
	return nil
}
```
