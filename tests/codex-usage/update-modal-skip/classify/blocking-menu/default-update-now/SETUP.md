# Scenario

**Bug**: usage fetch blocks on Update available menu with default selection Update now

```
01-update-modal-default.snapshot.txt -> blocking menu, selection UPDATE_NOW, writable loading
```

## Preconditions

Signed fixture `01-update-modal-default.snapshot.txt` (SHA-256 in PROTOCOL.md).

## Steps

1. `FixtureFile=01-update-modal-default.snapshot.txt`.

## Context

PROTOCOL step `detect_modal`. Default selection is **1. Update now** — Enter without
Down would upgrade; production must detect before injecting `/status`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "01-update-modal-default.snapshot.txt"
	return nil
}
```
