---
name: remote-agent
description: >-
  Operate ai-critic via remote-agent CLI (config, exec, upload, service, cron,
  git, server, request). Sealed SMC+SSH: remote-devbox. Triggers: remote-agent,
  remote service, remote upload/exec. Slash: /remote-agent.
  Multi-topic: remote-agent skill --show <topic>.
---

# remote-agent (CLI hub)

Control a configured ai-critic server over HTTP. **Sealed SMC + SSH** → skill
**remote-devbox** + `$AI/devbox/SETUP.md` §8 (link; do not restate pack flags).

This skill is an **index**. Load a topic:

```bash
remote-agent skill --show
remote-agent skill --show upload
remote-agent skill --show service
remote-agent skill service --show
remote-agent skill --list
```

## When to use

- `remote-agent` exec / upload / service / cron / git / server / request
- Long-lived remote processes; scheduled **remote** shell; Mac→remote files

## When not to use

- Pack / validate sealed bins → **remote-devbox**
- Create/register devbox SSH keys → **create-devbox-ssh**
- CodeLens watchdog ops → `$AI/knowledges/codelens/server/watchdog/TOPIC.md`

## Hard rules

| Rule | Detail |
|------|--------|
| Host of work | `exec` / `service` / `cron` run **on the remote**, not the Mac |
| Pack credentials | Pack Mac SMC/SSH on the **Mac**; ship with `upload` |
| Cron ≠ upload | `cron` cannot pack Mac tokens or run Mac `upload` |
| Transport | Prefer `remote-agent` over ad-hoc scp for this server |
| Secrets | No tokens/keys/app secrets in skill text, cron logs, or dumps |

## Topics

| Topic | Covers |
|-------|--------|
| `config` | Default domain / `config --web` |
| `exec` | Verbatim remote shell |
| `upload` | Gzip default, `--no-compress`, resume |
| `install` | Cross-build a local Go CLI onto the remote PATH |
| `service` | Lifecycle + **`service upgrade`** |
| `cron` | Remote schedules |
| `seal` | `remote-devbox refresh` + `/root/.smc` sync |
| `git` | Remote clone/fetch/pull/push |
| `server` | `build-next` / `restart` streams |
| `request` | Arbitrary API paths |
| `proxy` | List configured HTTP proxies |

## Command map

| Need | Use |
|------|-----|
| Default server | → topic **`config`** |
| One-shot shell | → **`exec`** |
| Mac→remote files | → **`upload`** |
| Cross-build a CLI onto remote PATH | → **`install`** |
| Long-lived / replace binary | → **`service`** |
| Scheduled remote shell | → **`cron`** |
| Token refresh | → **`seal`** |
| Remote git | → **`git`** |
| Build-next / restart | → **`server`** |
| Raw HTTP API | → **`request`** |

## Related

| Path | Role |
|------|------|
| **remote-devbox** | Pack / refresh / run-remote sealed SMC+SSH |
| `$AI/knowledges/codelens/server/watchdog/TOPIC.md` | CodeLens watchdog (consumer of `service upgrade`) |
