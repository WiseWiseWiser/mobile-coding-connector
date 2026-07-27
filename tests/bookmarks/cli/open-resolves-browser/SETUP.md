# Scenario

**Feature**: bookmarks open dry-run reports effective browser

```
# seed url browser=firefox; BOOKMARKS_OPEN_DRY_RUN=1
bookmarks open bm_open -> stdout mentions firefox and url
```

## Preconditions

1. Seed with browser firefox.
2. CLIEnv dry-run so no real /usr/bin/open.

## Steps

1. open bm_open.
2. Assert combined output includes firefox and the URL.

## Context

Open without GUI in CI.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{{
		Type: "url", ID: "bm_open", Name: "OpenMe",
		URL: "https://open.example.com", Browser: "firefox",
	}}
	req.CLIEnv = []string{"BOOKMARKS_OPEN_DRY_RUN=1"}
	req.CLIArgs = []string{"bookmarks", "open", "bm_open"}
	return nil
}
```
