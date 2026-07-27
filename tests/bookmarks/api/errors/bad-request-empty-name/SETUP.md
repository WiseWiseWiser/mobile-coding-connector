# Scenario

**Feature**: POST empty name returns 400

```
POST name="" -> 400
```

## Preconditions

1. RawBody or Name empty with type url.

## Steps

1. POST empty name.
2. Assert 400.

## Context

Validation over HTTP.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.APIOp = "post"
	req.RawBody = map[string]any{
		"parent_id": "root",
		"type":      "url",
		"name":      "",
		"url":       "https://example.com",
	}
	return nil
}
```
