# Scenario

**Feature**: omitted labels default to the provider name, then to name plus id

```
add --kind commandcode (no label) -> "CommandCode"
add --kind commandcode (no label) -> "CommandCode cc-v2"
```

## Preconditions

1. `FailHome` is unset, so both registrations validate.

## Steps

1. Set `Op=add` and register two Command Code items without labels.

## Context

Two Command Code sandboxes must be tellable apart in the menu bar.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "add"
	req.AddItems = []usageitems.Item{
		usageItem("cc-v1", "", usageitems.KindCommandCode, "/tmp/commandcode-v1/.commandcode"),
		usageItem("cc-v2", "", usageitems.KindCommandCode, "/tmp/commandcode-v2/.commandcode"),
	}
	return nil
}
```
