# Scenario

**Feature**: reject non-absolute url on url node

```
Add url="not-a-url" -> error
```

## Preconditions

1. Relative or opaque non-absolute URL.

## Steps

1. Add with URL `not-a-url`.
2. Assert ErrMsg.

## Context

Absolute http(s) requirement.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NodeType = "url"
	req.Name = "Bad"
	req.URL = "not-a-url"
	req.ID = "bm_badurl"
	return nil
}
```
