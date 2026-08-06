# Scenario

**Feature**: Install --both both hooks called

```
Install(Scope=both, LocalEnsure+RemoteEnsure → "2.54.0")
  -> both OK; both versions; PreferredUnisonVersion pin; nil error
```

## Preconditions

- Scope `both`.
- Fake local/remote versions `2.54.0`.
- PreferredUnisonVersion constant must be `2.54.0`.

## Steps

1. Mode install, Scope both.
2. Assert both hooks called, both OK, versions, pin constant.

## Context

- Scenario 4 from P4 design; also locks PreferredUnisonVersion.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "install"
	req.Scope = "both"
	req.FakeLocalVersion = "2.54.0"
	req.FakeRemoteVersion = "2.54.0"
	return nil
}
```
