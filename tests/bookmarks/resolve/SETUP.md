# Scenario

**Feature**: pure ResolveBrowser effective preference

```
# bookmarkBrowser + globalDefault
ResolveBrowser -> effective default|chrome|firefox|opera
```

## Preconditions

1. Mode resolve; no I/O.

## Steps

1. Leaf sets BookmarkBrowser and GlobalDefault.
2. Assert EffectiveBrowser.

## Context

Menu bar and CLI open share this rule.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "resolve"
	return nil
}
```
