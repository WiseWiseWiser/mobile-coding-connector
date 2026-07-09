# Scenario

**Feature**: resume skips files already complete locally

```
# local mirror has complete a.txt and sub/b.txt -> download skips both
pre-seeded local files (size match) -> remote-agent download -> skipped lines in stdout
```

## Preconditions

Remote tree at `uploads/mirror`; local `local-mirror/` pre-seeded with complete copies.

## Steps

1. Seed standard remote tree on server.
2. Pre-seed `local-mirror/a.txt` and `local-mirror/sub/b.txt` with matching content.
3. Args: `download uploads/mirror ./local-mirror`.

## Context

REQUIREMENT leaf #4 — dir-success/resume-skips-complete.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedStandardRemoteTree(req)
	setDownloadArgs(t, req, "uploads/mirror", "./local-mirror")
	req.LocalDir = localDirRel("uploads/mirror", "./local-mirror")
	req.LocalPreseedFiles = map[string]string{
		"a.txt":     "alpha\n",
		"sub/b.txt": "bravo\n",
	}
	return nil
}
```