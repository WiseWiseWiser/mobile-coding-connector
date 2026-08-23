---
name: remote-agent/config
description: >-
  Configure remote-agent default server domain and token (config / config --web).
---

# Config

Set the target ai-critic server once; later commands use the saved default
without `--server` / `--token` each time.

```bash
remote-agent config                 # help
remote-agent config --show          # dump saved JSON (tokens visible)
remote-agent config --web           # local UI to manage domains
```

Prefer `config --web` for editing multiple domains. Do not commit tokens.
