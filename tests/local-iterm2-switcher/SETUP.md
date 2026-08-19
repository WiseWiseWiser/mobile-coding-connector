# Scenario

**Feature**: local iTerm switcher inventory / focus / notes APIs

```
Capture + spaces + notes -> BuildInventory
POST focus -> Switch then Focus
PUT notes -> upsert / delete
```

## Preconditions

`server/localiterm2` exports Inventory, BuildInventory, FindLiveSession, NoteStore,
and Register mounts inventory/focus/notes. No live iTerm. Fixture snap may inject
`Session.Agent`; notes JSON may be a v1 map or a v2 `items` list.

## Steps

1. Root Setup validates request pointer.
2. Leaf Setup sets Op and fixture flags.
3. Root Run dispatches; leaf Assert checks inventory / HTTP / store.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req == nil {
		t.Fatal("nil request")
	}
	return nil
}
```
