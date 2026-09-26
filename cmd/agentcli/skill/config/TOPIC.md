---
name: remote-agent/config
description: >-
  Configure remote-agent server domains and tokens (config / config set /
  config --web).
---

# Config

Set the target ai-critic server once; later commands use the saved default
without `--server` / `--token` each time.

```bash
remote-agent config                 # help
remote-agent config --show          # dump saved JSON (tokens visible)
remote-agent config --web           # local UI to manage domains
```

## config set

Scriptable token updates for **any** server, default or not:

```bash
remote-agent config set --server https://x.dev --token-stdin
printf '%s\n' "$TOKEN" | xdev-agent config set --token-stdin   # via an alias
remote-agent --alias xdev config set --default                 # switch default
remote-agent config set --alias xdev --clear-token             # drop a token
```

| Target (first match wins) | Example |
| --- | --- |
| subcommand `--server` / `--alias` | `config set --alias xdev …` |
| global `--server` / `--alias` | `remote-agent --alias xdev config set …` |

Conflicting targets (an alias and a different `--server`) are an error; retarget
an alias with `remote-agent alias update <name> --server URL` instead.

`--token` works but is visible in shell history and `ps`; prefer `--token-stdin`.
Tokens are never echoed back. See the `alias` topic for wrapper commands.

Prefer `config --web` for editing multiple domains. Do not commit tokens.
