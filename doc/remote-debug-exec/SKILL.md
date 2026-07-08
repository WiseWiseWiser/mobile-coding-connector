---
name: remote-debug-exec
description: >-
  Debug a remote ai-critic deployment when the HTTP API is down: run shell commands
  via ./local-debug-remote-exec.sh, inspect keep-alive and server logs, and deploy with
  remote-agent upload/server/service. Use for Cloudflare 502, keep-alive restart loops,
  remote log forensics, or binary deploy when remote-agent ping fails.
---

# Remote debug exec + remote-agent deploy

Two paths to operate a remote ai-critic instance:

| Path | When to use |
|------|-------------|
| **`./local-debug-remote-exec.sh`** | API unreachable (502), need OS logs/processes/ports, keep-alive down |
| **`remote-agent`** | API up; upload, restart, services, tail logs over HTTP |

Try `remote-agent ping` first. If it fails, use `local-debug-remote-exec.sh`.

---

## 1. Setup: local remote-exec wrapper

The repo ships a **template only**. Your machine-specific entry command stays local and gitignored.

```bash
cp local-debug-remote-exec.sh.example local-debug-remote-exec.sh
chmod +x local-debug-remote-exec.sh
# Set REMOTE_EXEC to your non-interactive remote command wrapper (machine-specific)
```

`local-debug-remote-exec.sh` is in `.gitignore`. Never commit it.

**Use non-interactive remote exec.** Do not pipe commands into an interactive remote login — prompts (device auth, cert registration) consume stdin lines before your command runs (`echo` answers a prompt, `yes` runs forever).

The wrapper calls `REMOTE_EXEC` with `"$@"` appended so each invocation runs one remote command directly.

### Demo

```bash
./local-debug-remote-exec.sh echo yes
```

For pipelines or `&&` chains, use a single `sh -c '…'` argument.

### More examples

```bash
# Confirm remote cwd
./local-debug-remote-exec.sh sh -c 'cd /root && pwd'

# Smoke test when healthy
./local-debug-remote-exec.sh sh -c 'ss -tlnp | grep 23712'
./local-debug-remote-exec.sh sh -c 'curl -s http://127.0.0.1:23712/ping'

# Log forensics
./local-debug-remote-exec.sh sh -c 'cd /root && grep -c server_ready ai-critic-server-keep-alive.log'
```

### Rules learned the hard way

1. **Always `cd /root` first** — default cwd may not be the ai-critic home; data lives under `/root`.
2. **Run `pwd` on the first debug command** — confirm cwd before reading logs.
3. **Args pass through to remote** — `./local-debug-remote-exec.sh grep -c pattern file` works; use `sh -c` for `|`, `&&`, or multiple commands.
4. **Keep commands short** — avoid `tail` on multi-GB files; use `tail -n 50` or `grep` with date filters.
5. **Logs may be rotated/deleted** — capture excerpts during investigation.

Expected when healthy:

- Port **23712** — ai-critic server (`/ping` → `pong`)
- Port **23312** — keep-alive management API

---

## 2. Remote filesystem layout (`/root`)

| Path | Purpose |
|------|---------|
| `/root/ai-critic-server-vN` | Versioned server binaries |
| `/root/ai-critic-server` | Keep-alive daemon binary |
| `/root/ai-critic-server-keep-alive.log` | Keep-alive log (spawn, port checks, kills) |
| `/root/ai-critic-server.log` | Managed server stdout/stderr |
| `/root/.ai-critic/` | Config: domains, tunnels, services |
| `/root/.ai-critic/cloudflare-tunnel-gen-core.yml` | Tunnel ingress → `localhost:23712` |
| `/root/nohup.out` | Often huge — do not `tail` without `-n` |

### Cloudflare 502 checklist (via remote exec)

```bash
./local-debug-remote-exec.sh sh -c 'cd /root && grep -A2 hostname .ai-critic/cloudflare-tunnel-gen-core.yml'
./local-debug-remote-exec.sh sh -c 'ss -tlnp | grep 23712'
./local-debug-remote-exec.sh sh -c 'curl -s http://127.0.0.1:23712/ping'
./local-debug-remote-exec.sh sh -c 'curl -s http://127.0.0.1:23312/api/keep-alive/status'
```

502 with tunnel healthy → backend on 23712 is down, not Cloudflare.

---

## 3. Keep-alive log forensics

Run on remote via wrapper (each line below = one `./local-debug-remote-exec.sh` invocation, or combine with `cd /root` first):

```bash
./local-debug-remote-exec.sh sh -c 'cd /root && grep -c phase=server_ready ai-critic-server-keep-alive.log'
./local-debug-remote-exec.sh sh -c 'cd /root && grep -c "failed to become ready" ai-critic-server-keep-alive.log'
./local-debug-remote-exec.sh sh -c 'cd /root && grep "connection refused" ai-critic-server-keep-alive.log | tail -20'
./local-debug-remote-exec.sh sh -c 'cd /root && grep server_ready ai-critic-server-keep-alive.log'
```

| Log pattern | Meaning |
|-------------|---------|
| `connection refused` for full 10s → `failed to become ready` | Port not open before timeout kill |
| `Server started` many seconds after `Starting` | Slow fork/exec under I/O load |
| `waited_ms=1001` on `server_ready` | Normal ~1s startup |
| No `core_listen` in server.log before kill | Process died before `net.Listen` |

```bash
./local-debug-remote-exec.sh sh -c 'cd /root && grep -E "core_listen|core_ready" ai-critic-server.log | tail -20'
```

---

## 4. remote-agent (HTTP API path)

```bash
remote-agent config && remote-agent ping && remote-agent auth status
```

### Deploy

```bash
go run ./script/bundle/for-linux/
remote-agent server upload-next ./ai-critic-server-linux-amd64
remote-agent server restart
remote-agent server status
```

Or: `go run ./script/deploy-remote/`

### Upload / services / exec

```bash
remote-agent upload ./local-file /root/remote-path
remote-agent service list
remote-agent service logs --lines 100 <name>
remote-agent service upgrade <name> ./binary
remote-agent exec -- tail -n 30 /root/ai-critic-server-keep-alive.log
```

Prefer `remote-agent exec` when the API works — no local wrapper setup.

---

## 5. Incident playbook (502)

```
1. curl -sI https://<domain>/service

2. remote-agent ping
   ├─ YES → remote-agent server status / service logs / server restart
   └─ NO  → ./local-debug-remote-exec.sh (§1–3)

3. On remote: ss 23712, curl /ping, grep keep-alive log

4. Recovery:
   a) Deploy fix + keep-alive --startup-timeout 60s
   b) Emergency: kill keep-alive; nohup ./ai-critic-server-vN --port 23712 &
   c) Truncate bloated logs to reduce I/O

5. Verify public URL + local /ping
```

---

## 6. Post-fix keep-alive (on remote)

```bash
./local-debug-remote-exec.sh sh -c 'cd /root && nohup ./ai-critic-server-vN keep-alive --startup-timeout 60s --forever >> ai-critic-server-keep-alive.log 2>&1 &'
```

---

## 7. Related docs

- `local-debug-remote-exec.sh.example` — wrapper template
- `cmd/agentcli/skill/SKILL.md` — remote-agent reference
- `script/deploy-remote/` — build → upload-next → restart
- `tests/keep-alive/slow-core/` — startup-timeout regression tests

---

## 8. Case study (Jul 2026 remote 502)

**Symptom:** tunnel URL → 502; Cloudflare tunnel OK.

**Via remote exec:** port 23712 down; keep-alive kill loop; log showed `connection refused` → `failed to become ready within 10s`; normal startup ~1s, under load exec took 16s+.

**Fix:** `--startup-timeout 60s`, TCP-only startup checks, restart backoff (`tests/keep-alive/slow-core/`).