---
name: remote-agent
description: >-
  Operate ai-critic via remote-agent CLI (config, exec, upload, service, cron,
  git, server, request). Sealed SMC+SSH: remote-agent-manager devbox. Triggers: remote-agent,
  remote service, remote upload/exec, remote-agent exec, run on remote-agent.
  Slash: /remote-agent.
  Multi-topic: remote-agent skill --show <topic>.
---

# remote-agent (CLI hub)

Control a configured ai-critic server over HTTP. **Sealed SMC + SSH** → skill
**remote-agent-manager devbox** + `$AI/projects/remote-agent-manager/SETUP.md` §8 (link; do not restate pack flags).

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
- Go module cache via GOPROXY (`go` topic), not a mounted network FS

## When not to use

- Pack / validate sealed bins → **remote-agent-manager devbox**
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
| `upload` | Files: gzip + resume · dirs: tar.xz pack→merge, `--no-override` |
| `service` | Lifecycle + **`service upgrade`** |
| `install` | Cross-build a local Go CLI onto the remote PATH |
| `cron` | Remote schedules |
| `seal` | `remote-agent-manager devbox refresh` + `/root/.smc` sync |
| `git` | Remote clone/fetch/pull/push |
| `server` | `build-next` / `restart` streams |
| `request` | Arbitrary API paths |
| `proxy` | List configured HTTP proxies |
| `cloudflare-proxy` | Origin via edge `local-agent cloudflare-proxy` (no origin cloudflared) |
| `grok` | Start/resume remote Grok in this terminal tab |
| `deploy-service` | Recipe: install → upload → service add w/ domain → verify |
| `go` | GOPROXY on remote + local fast-fail relay |

## Command map

| Need | Use |
|------|-----|
| Default server | → topic **`config`** |
| One-shot shell | → **`exec`** |
| Mac→remote files | → **`upload`** |
| Long-lived / replace binary | → **`service`** |
| Cross-build a CLI onto remote PATH | → **`install`** |
| Scheduled remote shell | → **`cron`** |
| Token refresh | → **`seal`** |
| Remote git | → **`git`** |
| Build-next / restart | → **`server`** |
| Raw HTTP API | → **`request`** |
| Remote Grok TUI (this tab) | → **`grok`** |
| Stand up a web service + public domain | → **`deploy-service`** |
| GOPROXY / module cache offload | → **`go`** |
| Origin behind CF proxy (no origin cloudflared) | → **`cloudflare-proxy`** |

## Related

| Path | Role |
|------|------|
| **remote-agent-manager devbox** | Pack / refresh / run sealed SMC+SSH |
| `$AI/knowledges/codelens/server/watchdog/TOPIC.md` | CodeLens watchdog (consumer of `service upgrade`) |
| `$AI/knowledges/ai-critic/event-bus-open-tty/TOPIC.md` | Remote detach → Mac `event-bus listen --open-tty` |
| `$AI/knowledges/ai-critic/remote-go-cache-fs/TOPIC.md` | Why GOPROXY instead of SSHFS/NFS/SMB for module cache |
| `$AI/knowledges/ai-critic/cloudflare-proxy/TOPIC.md` | Private fleet hosts for this proxy |
