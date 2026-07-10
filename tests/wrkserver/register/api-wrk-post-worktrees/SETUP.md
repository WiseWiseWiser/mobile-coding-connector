# Scenario

**Feature**: Register with ai-critic base exposes POST worktrees

```
Register(mux, "/api/wrk") -> POST /api/wrk/worktrees -> not 404
```

## Preconditions

Base `/api/wrk`; body `{}` (missing project_path) to prove route hits handler.

## Steps

1. Set `Base=/api/wrk`, `Method=POST`, `Path=/api/wrk/worktrees`.
2. Default empty JSON body from Run.

## Context

REQUIREMENT scenario 10 (POST half). Validation 4xx proves the handler ran
(not mux 404).

```go
import (
	"net/http"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Base = "/api/wrk"
	req.Method = http.MethodPost
	req.Path = "/api/wrk/worktrees"
	return nil
}
```
