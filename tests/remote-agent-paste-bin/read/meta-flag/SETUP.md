# Scenario

**Feature**: read scratch with --meta

```
# seeded scratch + --meta -> gray updated at on stderr, content on stdout
seed scratch -> paste-bin --meta -> stderr timestamp + stdout content
```

## Preconditions

Scratch seeded with fixed `updated_at` for deterministic stderr assertion.

## Steps

1. `seedScratch(req, seededUTF8Content, seededMetaUpdatedAt)`.
2. `setReadTTY(req, "--meta")`.

## Context

REQUIREMENT leaf: `read-meta-flag`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedScratch(req, seededUTF8Content, seededMetaUpdatedAt)
	setReadTTY(t, req, "--meta")
	return nil
}
```