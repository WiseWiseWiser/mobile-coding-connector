---
name: remote-agent/server
description: >-
  remote-agent server build-next and restart (Manage Server page equivalents).
---

# Server

Same actions as the Manage Server UI, streamed over HTTP.

```bash
remote-agent server build-next
remote-agent server build-next --project my-project-id
remote-agent server restart
```

| Command | Stream |
|---------|--------|
| `build-next` | `/api/build/build-next` |
| `restart` | `/api/server/exec-restart` |

Replacing a **managed service** binary (not the ai-critic core) → topic **service**
(`service upgrade`), not `server restart`.
