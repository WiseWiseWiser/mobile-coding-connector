# Scenario

**Feature**: Register base is host-owned (not hardcoded /api/wrk)

```
Register(mux, "/custom") -> GET /custom/projects -> 200
```

## Preconditions

Empty registry; base `/custom` (not `/api/wrk`).

## Steps

1. Set `Base=/custom`, `Method=GET`, `Path=/custom/projects`.
2. Empty projects registry.

## Context

REQUIREMENT scenario 11 — proves wrkserver does not hardcode `/api/wrk` as the
only mount prefix.

```go
import (
	"net/http"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	writeProjectsJSON(t, req.WrkHome, nil)
	req.Base = "/custom"
	req.Method = http.MethodGet
	req.Path = "/custom/projects"
	return nil
}
```
