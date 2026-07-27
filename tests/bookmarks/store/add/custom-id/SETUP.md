# Scenario

**Feature**: client-supplied id is preserved on add

```
Add with id=bm_custom -> stored id equals bm_custom
```

## Preconditions

1. ID set by leaf.

## Steps

1. Add url with id bm_custom.
2. Assert FindNode bm_custom.

## Context

Stable ids for CLI/API callers.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NodeType = "url"
	req.ID = "bm_custom"
	req.Name = "Custom"
	req.URL = "https://example.com/custom"
	return nil
}
```
