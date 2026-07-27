# Scenario

**Feature**: POST adds url under root

```
POST {parent_id:root,type:url,name,url} -> 2xx; GET shows node
```

## Preconditions

1. APIOp post with name/url.

## Steps

1. POST add Docs site.
2. Assert 2xx and FindNodeByName.

## Context

Create via HTTP.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.APIOp = "post"
	req.NodeType = "url"
	req.Name = "Docs"
	req.URL = "https://docs.example.com"
	req.ID = "bm_docs"
	return nil
}
```
