# Scenario

**Feature**: read scratch with --json

```
# seeded scratch -> paste-bin --json -> JSON on stdout
GET scratch -> --json -> {"content","updated_at"} on stdout
```

## Preconditions

Scratch seeded with known UTF-8 content.

## Steps

1. `seedScratch(req, seededUTF8Content, seededMetaUpdatedAt)`.
2. `setReadTTY(req, "--json")`.

## Context

REQUIREMENT leaf: `read-json-flag`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedScratch(req, seededUTF8Content, seededMetaUpdatedAt)
	setReadTTY(t, req, "--json")
	return nil
}
```