# Scenario

**Feature**: BuiltinExclusionConfig lists pathflag catalog paths and binary row

```
BuiltinExclusionConfig().ExcludePaths
  ⊇ pathflag path rules + **/node_modules + **/upload-chunks + **/*.log + **(binary)
```

## Preconditions

- Golden path list mirrors pathflag attributeRules plus specials and synthetic binary.

## Steps

1. Op ssot_catalog_paths.
2. Expect MissingPaths empty.

## Context

- Ensures config surface enumerates the catalog implementer must generate from pathflag.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpSSOTCatalogPaths
	return nil
}
```
