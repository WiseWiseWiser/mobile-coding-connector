# Scenario

**Feature**: rotation walks enabled items in registry order and skips disabled ones

```
grok(enabled), codex(disabled), cc-v1(enabled) -> rotating title covers grok and cc-v1
```

## Preconditions

1. Codex is registered but disabled.

## Steps

1. Set `Op=list` with `SeedRotate=true` and a disabled codex entry.

## Context

The app cycles the enabled items every 60 seconds; disabled ones must not appear.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "list"
	req.SeedItems = []usageitems.Item{
		usageItem("grok", "Grok", usageitems.KindGrok, ""),
		{ID: "codex", Label: "Codex", Kind: usageitems.KindCodex, Enabled: false},
		usageItem("cc-v1", "CC v1", usageitems.KindCommandCode, "/tmp/commandcode-v1/.commandcode"),
	}
	req.SeedRotate = true
	return nil
}
```
