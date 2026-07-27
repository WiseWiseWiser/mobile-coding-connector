# Scenario

**Feature**: Codex temporary plugin cache is excluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded(".codex/.tmp/plug") == true
  rule .codex/.tmp (Tmp|Cache)
```

## Preconditions

- Catalog prefix `.codex/.tmp`.

## Steps

1. RelPath under `.codex/.tmp`.
2. Expect excluded.

## Context

- Distinguishes from included `.codex/config.toml`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.RelPath = ".codex/.tmp/plug"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
