# Scenario

**Feature**: single-file upload dry-run streams chunk plan without server write

```
# copy hello.txt -> --dry-run upload uploads/hello.txt -> would upload chunk, server unchanged
local hello.txt -> remote-agent upload --dry-run -> plan only
```

## Preconditions

No remote pre-seed; destination path absent on server.

## Steps

1. Copy `testdata/hello.txt` into a temp local file.
2. Args: `upload --dry-run <local> uploads/hello.txt`.

## Context

REQUIREMENT-DESIGN-upload-download-dry-run.md — file-regression/dry-run-single-file.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	localDir := mkLocalWorkDir(t)
	localFile := filepath.Join(localDir, "hello.txt")
	// Fixture lives under sibling leaf single-file/testdata.
	fixture := filepath.Join(d.DOCTEST_CASE, "..", "single-file", "testdata", "hello.txt")
	copyFixture(t, fixture, localFile)
	setUploadDryRunArgs(t, req, localFile, "uploads/hello.txt")
	req.RemoteDir = "uploads"
	return nil
}
```