# Scenario

**Feature**: add url bookmark under default parent root

```
Add type=url name+url -> child of root
```

## Preconditions

1. ParentID omitted → root.

## Steps

1. Add url Grafana https://grafana.example.com.
2. Assert child present under root.

## Context

Happy-path create url.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NodeType = "url"
	req.Name = "Grafana"
	req.URL = "https://grafana.example.com"
	return nil
}
```
