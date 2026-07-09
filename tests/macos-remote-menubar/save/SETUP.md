# Scenario

**Feature**: save remote-agent-config.json without wiping unrelated fields

```
Config + mutation -> remoteconfig.Save(path) -> file mode 0600, project_bindings kept
```

## Preconditions

`remoteconfig.Save` writes pretty JSON with mode `0600` and preserves
`project_bindings` when domains/default/token are updated.

## Steps

1. Set `Op=save`.
2. Leaf supplies `ConfigJSON` and optional update fields.

## Context

REQUIREMENT group: `save/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "save"
	return nil
}
```
