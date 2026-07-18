# Scenario

**Feature**: go.mod requires bak-files and replaces it to monorepo root

```
# from ai-critic snapshot module root
go.mod -> require github.com/xhd2015/bak-files <version>
go.mod -> replace github.com/xhd2015/bak-files => ../..
```

## Preconditions

- Replace path is relative from this snapshot: parent of `external/` is bak-files root.

## Steps

1. Set Op to module_go_mod.
2. Expect both require and replace present.

## Context

- RED until implementer edits go.mod (designer must not).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpModuleGoMod
	return nil
}
```
