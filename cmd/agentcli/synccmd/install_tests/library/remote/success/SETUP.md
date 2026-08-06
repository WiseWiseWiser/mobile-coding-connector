# Scenario

**Feature**: Install --remote success via RemoteEnsure hook

```
Install(Scope=remote, RemoteEnsure(targetPath) → "2.54.0")
  -> RemoteOK true; RemoteVersion 2.54.0; LocalEnsure not called; nil error
```

## Preconditions

- Scope `remote`.
- FakeRemoteVersion `2.54.0`.
- RemoteTargetPath non-empty (root default).

## Steps

1. Mode install, Scope remote.
2. Assert RemoteEnsure called with RemoteTargetPath; LocalEnsure not; RemoteOK.

## Context

- Scenario 3 from P4 design (library equivalent of `--remote`).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "install"
	req.Scope = "remote"
	req.FakeRemoteVersion = "2.54.0"
	return nil
}
```
