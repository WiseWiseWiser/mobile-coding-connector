# Scenario

**Feature**: server-scoped backup paths and archive filenames

```
server URL / home / UTC time -> ServerNameFromURL | BackupDir | BackupArchiveFilename
```

## Preconditions

`Op` is one of `path_server_name`, `path_backup_dir`, `path_archive_filename`.

## Steps

1. Leaf supplies URL, home+serverName, or UTC time.

## Context

REQUIREMENT: paths/naming scenarios 8–10.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Op set by each leaf under path/
	if req.Op == "" {
		req.Op = "path_server_name"
	}
	return nil
}
```
