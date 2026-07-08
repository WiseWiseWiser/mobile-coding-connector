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

**The local script works.** If commands return quickly but logs/ports are missing,
the usual cause is **wrong target** (misconfigured `REMOTE_EXEC`), not a broken
wrapper. Do not abandon the script, retry `remote-agent` in parallel, or assume
the remote shell is down — verify you are on the correct host first.

---

## 0. Verify target before forensics (mandatory)

`REMOTE_EXEC` in `local-debug-remote-exec.sh` is machine-specific and **not**
validated against `~/.ai-critic/remote-agent-config.json`. A common failure mode:
the wrapper points at one SMC service while `remote-agent` targets a different
host (e.g. `consumerloan-codelensadmin-test-ph` vs the ai-critic box behind
`agent-fast-apex-nest-23aed.xhd2015.xyz`). Commands succeed on the wrong
container; every `grep` returns empty; investigators blame tooling instead of
config.

**Run this on every new incident before reading logs:**

```bash
# 1) Script smoke test — must return quickly with output
./local-debug-remote-exec.sh echo yes

# 2) Confirm you landed on the ai-critic host (not some other service cwd)
./local-debug-remote-exec.sh sh -c 'pwd; hostname; ls -la /root/ai-critic-server* 2>/dev/null | head -5'

# 3) Cross-check against remote-agent config
remote-agent config   # note the server URL / domain
```

| Check | Healthy signal | Wrong-target signal |
|-------|----------------|---------------------|
| `echo yes` | prints `yes` in <5s | hangs, auth prompt, empty |
| `pwd` | `/root` (typical) | unrelated app dir (e.g. `/consumerloan-codelensadmin`) |
| `ls /root/ai-critic-server*` | versioned binaries | `no such file` |
| `ss -tlnp \| grep 23712` | listener on 23712 | no match |
| `remote-agent ping` | responds (when API up) | timeout — expected during outage; **do not** spam parallel pings |

**If wrong-target:** fix `REMOTE_EXEC` in `local-debug-remote-exec.sh` to reach
the same machine that serves your `remote-agent` URL, then re-run §0. Do not
proceed to log forensics until `ai-critic-server*` exists under `/root`.

**If right-target but API down:** stick with `./local-debug-remote-exec.sh` only.
`remote-agent ping` / `server status` / `exec` will hang or 502 while keep-alive
is in a restart loop — that is the symptom, not evidence the local script failed.

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

1. **Verify target first (§0)** — `REMOTE_EXEC` must hit the same host as your
   `remote-agent` server URL. Empty `grep` on a fast response means wrong box,
   not broken script.
2. **The local script is the fallback when API is down** — it is reliable when
   configured correctly. Do not treat `remote-agent` timeouts as proof the
   wrapper failed.
3. **Always `cd /root` first** — default remote cwd is often an unrelated app
   dir; ai-critic data lives under `/root`.
4. **Run `pwd` + `ls /root/ai-critic-server*` on the first debug command** —
   confirm host and paths before reading logs.
5. **Args pass through to remote** — `./local-debug-remote-exec.sh grep -c pattern file` works; use `sh -c` for `|`, `&&`, or multiple commands.
6. **Keep commands short** — avoid `tail` on multi-GB files; use `tail -n 50` or `grep` with date filters.
7. **Logs may be rotated/deleted** — capture excerpts during investigation.
8. **One path at a time** — when `remote-agent ping` fails, use only
   `./local-debug-remote-exec.sh`; do not launch parallel `remote-agent` calls
   that will all hang.

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

2. ./local-debug-remote-exec.sh sh -c 'pwd; ls /root/ai-critic-server*'  (§0 — confirm target)

3. remote-agent ping
   ├─ YES → remote-agent server status / service logs / server restart
   └─ NO  → ./local-debug-remote-exec.sh only (§1–3); do not parallel remote-agent

4. On remote: ss 23712, curl /ping, grep keep-alive log

5. Recovery:
   a) Deploy fix + keep-alive --startup-timeout 60s
   b) Emergency: kill keep-alive; nohup ./ai-critic-server-vN --port 23712 &
   c) Truncate bloated logs to reduce I/O

6. Verify public URL + local /ping
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

## 9. Anti-pattern: blaming the wrapper

**Symptom:** investigator runs `./local-debug-remote-exec.sh`; commands return in
~2s but `grep ai-critic-server-keep-alive.log` is empty; `remote-agent ping`
hangs for 60s+; many parallel remote-agent calls launched.

**Actual cause:** `REMOTE_EXEC` pointed at the wrong SMC service. Remote cwd was
`/consumerloan-codelensadmin` (unrelated Node app), not `/root` (ai-critic). The
wrapper worked; the target was wrong. `remote-agent` hung because the real
ai-critic server was down — expected during a restart loop.

**Correct response:** run §0, fix `REMOTE_EXEC` to the ai-critic host, then grep
logs via the local script. Do not retry `remote-agent` until `/ping` recovers or
you need deploy actions that require HTTP.