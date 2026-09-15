# Scenario

**Feature**: a failed provider validation warns and still registers the item

```
add --home /tmp/dead -> warning: cc-dead fetch failed: Session expired (exit 0)
```

## Preconditions

1. `FailHome=/tmp/dead` makes the provider fixture fail.

## Steps

1. Set `Op=add` and register one item pointing at the failing home.

## Context

A user typo in `--home` must not lose the registration.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "add"
	req.FailHome = "/tmp/dead"
	req.AddItems = []usageitems.Item{
		usageItem("cc-dead", "CC dead", usageitems.KindCommandCode, "/tmp/dead"),
	}
	return nil
}
```
