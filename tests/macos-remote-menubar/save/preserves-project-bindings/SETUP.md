# Scenario

**Feature**: save preserves project_bindings when updating token

```
load config with project_bindings + update domain token -> Save -> Load -> bindings unchanged
```

## Preconditions

Initial config includes one domain and one project binding.

## Steps

1. Seed ConfigJSON with bindings and token `old`.
2. Set `UpdateToken=new-secret`.

## Context

REQUIREMENT leaf: `save/preserves-project-bindings`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://example.com",
  "domains": [
    {"server": "https://example.com", "token": "old"}
  ],
  "project_bindings": [
    {
      "server": "https://example.com",
      "remote_dir": "/home/u/proj",
      "local_path": "/Users/u/proj"
    }
  ]
}`
	req.UpdateToken = "new-secret"
	return nil
}
```
