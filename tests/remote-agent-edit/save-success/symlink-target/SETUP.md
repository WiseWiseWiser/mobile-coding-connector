# Scenario

**Feature**: writing through a symlinked remote path preserves the link

```
# app.conf -> real/app.conf; edit app.conf -> target replaced, link kept, warning
serverHome app.conf -> real/app.conf -> remote-agent edit app.conf -> real/app.conf updated
```

## Preconditions

Remote `real/app.conf` contains `old\n`; `app.conf` is a symlink to `real/app.conf`.

## Steps

1. Seed the target file and the symlink; point the CLI at `app.conf`.
2. Editor writes `new\n`.
3. Assert exit 0, a `warning:` line naming the resolved target, the link still a
   symlink, and the target updated.

## Context

Happy-path leaf #7 — save-success/symlink-target (config files such as
`/etc/nginx/nginx.conf` are commonly symlinks; the link must not be replaced by a
regular file).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FileArg = "app.conf"
	req.RemoteRel = "app.conf"
	req.ServerPreseedFiles = map[string]string{"real/app.conf": "old\n"}
	req.ServerSymlinks = map[string]string{"app.conf": "real/app.conf"}
	req.Args = []string{"edit", "app.conf"}
	req.EditorWrite = "new\n"
	return nil
}
```
