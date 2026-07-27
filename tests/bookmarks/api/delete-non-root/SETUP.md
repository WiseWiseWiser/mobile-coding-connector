# Scenario

**Feature**: DELETE removes non-root node

```
DELETE ?id=bm_d -> 2xx; GET omits bm_d
```

## Preconditions

1. Seed bm_d.

## Steps

1. DELETE.
2. Assert absent.

## Context

Delete via HTTP.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{{
		Type: "url", ID: "bm_d", Name: "Del", URL: "https://d.example.com",
	}}
	req.APIOp = "delete"
	req.ID = "bm_d"
	return nil
}
```
