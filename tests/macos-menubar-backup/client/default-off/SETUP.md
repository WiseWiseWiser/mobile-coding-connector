# Scenario

**Feature**: periodic backup defaults off; no auto-enable on launch

```
app launch / onAppear -> must not force backupEnabled=true
default task enabled=false
```

## Preconditions

Default OFF until the user chooses Enable.

## Steps

1. Set `ClientLeaf=default-off`.

## Context

REQUIREMENT #26 and runtime policy Default: Off.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "default-off"
	return nil
}
```
