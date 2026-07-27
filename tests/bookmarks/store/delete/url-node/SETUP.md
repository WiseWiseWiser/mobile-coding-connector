# Scenario

**Feature**: delete a url node leaves siblings intact

```
Delete bm_gone; bm_stay remains under root
```

## Preconditions

1. Two url seeds.

## Steps

1. Delete bm_gone.
2. Assert gone missing, stay present.

## Context

Non-recursive leaf delete.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PreAdds = []SeedNode{
		{Type: "url", ID: "bm_gone", Name: "Gone", URL: "https://gone.example.com"},
		{Type: "url", ID: "bm_stay", Name: "Stay", URL: "https://stay.example.com"},
	}
	req.ID = "bm_gone"
	return nil
}
```
