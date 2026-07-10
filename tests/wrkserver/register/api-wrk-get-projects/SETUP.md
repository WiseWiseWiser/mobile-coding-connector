# Scenario

**Feature**: Register with ai-critic base exposes GET projects

```
Register(mux, "/api/wrk") -> GET /api/wrk/projects -> 200
```

## Preconditions

Empty registry; base `/api/wrk`.

## Steps

1. Set `Base=/api/wrk`, `Method=GET`, `Path=/api/wrk/projects`.
2. Ensure empty projects registry.

## Context

REQUIREMENT scenario 10 (GET half).

```go
import (
	"net/http"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	writeProjectsJSON(t, req.WrkHome, nil)
	req.Base = "/api/wrk"
	req.Method = http.MethodGet
	req.Path = "/api/wrk/projects"
	return nil
}
```
