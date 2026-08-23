---
name: remote-agent/request
description: >-
  Call arbitrary ai-critic API paths with remote-agent request.
---

# Request

Call HTTP API paths on the configured server (auth from saved config).

```bash
remote-agent request /api/services
remote-agent request /api/services/start?id=svc-123 '{}'
echo '{"name":"demo"}' | remote-agent request /api/some
```

Prefer dedicated subcommands (`service`, `git`, …) when they exist; use
`request` for ad-hoc or undocumented paths.
