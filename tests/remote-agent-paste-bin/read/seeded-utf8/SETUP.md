# Scenario

**Feature**: read seeded multiline UTF-8 scratch

```
# scratch.json with UTF-8 -> paste-bin -> exact bytes on stdout
seed multiline UTF-8 -> remote-agent paste-bin -> stdout content
```

## Preconditions

Scratch seeded with `line1\nline2\nemoji🎉`.

## Steps

1. `seedScratch(req, seededUTF8Content, "")`.
2. `setReadTTY(req)`.

## Context

REQUIREMENT leaf: `read-seeded-utf8`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedScratch(req, seededUTF8Content, "")
	setReadTTY(t, req)
	return nil
}
```