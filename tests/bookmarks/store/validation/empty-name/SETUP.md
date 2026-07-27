# Scenario

**Feature**: reject empty name on add

```
Add name="" -> error
```

## Preconditions

1. Name empty string; url present.

## Steps

1. Add url with empty name.
2. Assert ErrMsg.

## Context

Name non-empty rule.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NodeType = "url"
	req.Name = ""
	req.URL = "https://example.com"
	req.ID = "bm_noname"
	return nil
}
```
