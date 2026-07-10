# Scenario

**Feature**: backup directory under home for a server name

```
BackupDir("/Users/me", "foo.example.com") -> "/Users/me/.backup/ai-critic/foo.example.com"
```

## Preconditions

Home is an absolute path; server-name already resolved.

## Steps

1. Set `Op=path_backup_dir`, home `/Users/me`, server `foo.example.com`.

## Context

REQUIREMENT leaf: paths #9.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "path_backup_dir"
	req.HomeDir = "/Users/me"
	req.ServerName = "foo.example.com"
	return nil
}
```
