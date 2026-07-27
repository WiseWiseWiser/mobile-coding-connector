# Scenario

**Feature**: PATCH renames existing bookmark

```
PATCH ?id=bm_p {name:Renamed} -> 2xx; GET shows Renamed
```

## Preconditions

1. Seed bm_p; APIOp patch.

## Steps

1. PATCH name Renamed.
2. Assert name change.

## Context

Update via HTTP.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{{
		Type: "url", ID: "bm_p", Name: "Orig", URL: "https://p.example.com",
	}}
	req.APIOp = "patch"
	req.ID = "bm_p"
	req.Name = "Renamed"
	return nil
}
```
