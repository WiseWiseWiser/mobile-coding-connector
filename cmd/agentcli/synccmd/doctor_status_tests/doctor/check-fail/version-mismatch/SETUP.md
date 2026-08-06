# Scenario

**Feature**: Doctor fails versions-match when local and remote differ

```
LocalVersion=2.54.0, RemoteVersion=2.51.0
  -> Doctor -> versions-match OK=false, AllOK false
```

## Preconditions

- Happy hooks except RemoteVersion.

## Steps

1. Inherit seeded pair + profile from parent.
2. Override RemoteVersion to `2.51.0`.

## Context

- Equality of version number strings (not semver range).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.RemoteVersion = fakeVersion("2.51.0")
	return nil
}
```
