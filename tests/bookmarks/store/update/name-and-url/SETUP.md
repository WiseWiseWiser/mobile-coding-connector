# Scenario

**Feature**: update bookmark name and url together

```
Update id bm_u name+url -> both fields change
```

## Preconditions

1. PreAdd bm_u old name/url.

## Steps

1. Update name New Name and url https://new.example.com.
2. Assert both fields.

## Context

Combined field patch.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PreAdds = []SeedNode{{
		Type: "url", ID: "bm_u", Name: "Old", URL: "https://old.example.com",
	}}
	req.ID = "bm_u"
	req.Name = "New Name"
	req.URL = "https://new.example.com"
	return nil
}
```
