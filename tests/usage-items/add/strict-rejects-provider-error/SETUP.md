# Scenario

**Feature**: --strict turns a failed validation into a rejection

```
add --strict --home /tmp/dead -> invalid: cc-dead fetch failed: Session expired
```

## Preconditions

1. `FailHome=/tmp/dead` makes the provider fixture fail.

## Steps

1. Set `Op=add`, `AddStrict=true`, and register one item pointing at the failing home.

## Context

Scripts that must not register an unreachable provider use `--strict`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "add"
	req.FailHome = "/tmp/dead"
	req.AddStrict = true
	req.AddItems = []usageitems.Item{
		usageItem("cc-dead", "CC dead", usageitems.KindCommandCode, "/tmp/dead"),
	}
	return nil
}
```
