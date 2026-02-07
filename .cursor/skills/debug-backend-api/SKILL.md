---
name: debug-backend-api
description: Debug and test backend API endpoints using the request CLI tool. Use when the user wants to test an API endpoint, debug a backend handler, check API responses, or investigate server-side issues.
---

# Debug Backend API

## Quick Start

Use `go run ./script/request` to send authenticated HTTP requests to the local server (port 23712).

```bash
# GET request
go run ./script/request /api/checkpoints?project=myproject

# POST request with JSON body
go run ./script/request /api/checkpoints '{"project_dir":"/path/to/project","name":"test","file_paths":["file.txt"]}'
```

The tool automatically reads `.server-credentials` for auth (cookie: `ai-critic-token`).

## Usage

```
go run ./script/request <path> [body]
```

- **No body** → GET request
- **With body** → POST request with `Content-Type: application/json`
- Status line printed to stderr, response body to stdout
- Pipe to `jq` for formatted output: `go run ./script/request /api/xxx | jq .`

## Common Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/auth/check` | GET | Verify auth is working |
| `/api/checkpoints?project=NAME` | GET | List checkpoints |
| `/api/checkpoints?project=NAME` | POST | Create checkpoint |
| `/api/checkpoints/ID?project=NAME` | GET | Get checkpoint detail |
| `/api/checkpoints/current-changes?project=NAME&project_dir=DIR` | GET | Get current changes |
| `/api/checkpoints/diff?project=NAME&project_dir=DIR` | GET | Get current diffs |
| `/api/files?dir=DIR` | GET | List files in directory |
| `/api/files/content?dir=DIR&path=FILE` | GET | Read file content |
| `/api/port-forwards` | GET | List port forwards |
| `/api/agents` | GET | List agents |
| `/api/terminal/sessions` | GET | List terminal sessions |

## Debugging Workflow

1. **Identify the endpoint**: Check the frontend API call in `ai-critic-react/src/api/` to find the endpoint path and parameters.
2. **Send a test request**: Use `go run ./script/request` with the endpoint.
3. **Inspect the response**: Check status code and response body.
4. **Find the handler**: Backend handlers are registered in `server/server.go` (main routes) or in sub-packages like `server/checkpoint/`, `server/portforward/`, `server/terminal/`.
5. **Add debug logging**: Add `fmt.Printf("DEBUG ...")` lines in the handler, rebuild with `go run ./script/server/run`, and re-test.

## Important Notes

- Server must be running (`go run ./script/server/run`) before sending requests.
- The `.server-credentials` file must exist if auth is enabled; each line is a valid token.
- Always limit output when piping: `go run ./script/request /api/xxx | head -c 4096`.
