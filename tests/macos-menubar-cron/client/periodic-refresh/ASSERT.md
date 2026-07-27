## Expected

1. `HasPeriodicCronRefresh` is true — timer/sleep or Refresh path fetches cron tasks.

## Side Effects

- None (read-only source inspection).

## Errors

- Cron only loaded once at launch; missing from 30s poll / Refresh.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasPeriodicCronRefresh {
		t.Fatalf("cron missing from periodic/Refresh path (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
