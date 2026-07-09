# LOOP: fix openclaw service bash startup failure

**Created:** 2026-07-08  
**Slug:** `fix-openclaw-service-bash`  
**Dry-run status:** PASS (see Dry-run log)

## Goal

After building and deploying `ai-critic-server-linux-amd64` to the remote host, the
managed **openclaw** service starts successfully: status `running`, a live PID, and
service logs show gateway startup — **not** repeated
`failed to start: fork/exec /bin/bash: no such file or directory`.

## Prerequisites

| Requirement | Probe | Healthy signal |
|-------------|-------|----------------|
| `go` | `which go` | path printed |
| `remote-agent` | `which remote-agent` | path printed |
| Remote config | `remote-agent config` | server URL printed |
| API reachable | `remote-agent ping` | `ok (pong …)` |
| openclaw service defined | `remote-agent service list` | `Name: openclaw` present |

**Note:** `local-debug-remote-exec.sh` is optional when `remote-agent ping` succeeds.
Use `remote-agent exec` for remote shell commands.

**Known pitfall:** `fork/exec /bin/bash: no such file or directory` is often Go's
misleading error when the service **WorkingDir** does not exist (openclaw uses
`/root/my-openclaw`). Create the directory before restart (see Run step).

## 1. Build

Compile the Linux server bundle (frontend + backend):

```sh
cd "$(git rev-parse --show-toplevel)"
go run ./script/bundle/for-linux/
```

**Verify:** artifact exists and is non-empty:

```sh
test -s ./ai-critic-server-linux-amd64 && ls -lh ./ai-critic-server-linux-amd64
```

Expected: file size > 1 MB, exit code 0.

## 2. Deploy / Update

Upload the new binary and restart the managed server (keep-alive picks up the new
version):

```sh
remote-agent server upload-next ./ai-critic-server-linux-amd64
remote-agent server restart
```

**Verify:** server is reachable after restart (may take up to ~60s; 502 is transient):

```sh
for i in $(seq 1 20); do remote-agent ping && break; sleep 5; done
remote-agent server status
```

Expected: ping `ok`; server status shows running / ready.

## 3. Run

Ensure the openclaw working directory exists, then restart the service:

```sh
remote-agent exec sh -c 'mkdir -p /root/my-openclaw'
remote-agent service restart openclaw
```

**Verify:** service reports running with a PID:

```sh
remote-agent service list 2>&1 | awk '/Name:.*openclaw/,/^$/' | grep -E 'Status:|PID:'
```

Expected: `Status: running` and `PID:` > 0 (not `unavailable`).

## 4. Inspect / Feedback

Run the inspect script (single source of truth for loop success):

```sh
go run ./script/debug/openclaw-service-inspect/
```

**Verify:** exit code 0 and stdout contains `PASS:`.

Manual log spot-check (optional):

```sh
remote-agent exec sh -c 'tail -n 30 /root/.ai-critic/services/svc-1782644588356650870.log'
```

Expected signals in log tail:

- `[gateway] starting` or similar gateway startup lines
- **No** `failed to start: fork/exec /bin/bash`
- **No** immediate `service exited with error` after the latest start

Sustained health (optional, from reference debug doc):

```sh
# 10 quick pings — all must succeed
for i in $(seq 1 10); do remote-agent ping || exit 1; sleep 2; done
```

## 5. Fix

When inspect fails, use this decision tree:

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `fork/exec /bin/bash` in openclaw log only | Missing `WorkingDir` (`/root/my-openclaw`) | `mkdir -p /root/my-openclaw`; consider validating dir in `server/services/services.go` before `cmd.Start()` |
| `exit status 127` / command not found | `my` or `openclaw` not on PATH | Install openclaw CLI on remote; check `remote-agent openclaw doctor` |
| Service `error` after gateway starts | App crash / port conflict | Read full service log; check port 18789 with `remote-agent exec ss -tlnp` |
| Deploy step fails | API down / auth | `remote-agent auth status`; fall back to `local-debug-remote-exec.sh` per `doc/remote-debug-exec/SKILL.md` |
| Bash truly missing on host | Minimal container image | Change `server/services/services.go` to use `sh -lc` or absolute shell path |

After any code fix, return to **step 1** (rebuild → redeploy → restart → inspect).

## Pitfalls & blockers

| Blocker | Detection | Unblock |
|---------|-----------|---------|
| No `remote-agent` config | `remote-agent ping` → "no server specified" | `remote-agent config` — add default domain |
| Interactive SMC upload | `spl smc upload` prompts | Use `echo -ne $'n\nn\n' \| spl smc upload …` or `remote-agent server upload-next` |
| Wrong remote host | `service list` empty or no openclaw | Fix `~/.ai-critic/remote-agent-config.json` server URL |
| API down during incident | ping timeout | Use `./local-debug-remote-exec.sh` (copy from `.example`) |
| Misleading bash error | bash exists at `/bin/bash` but start fails | Check `WorkingDir` exists: `remote-agent exec ls -la /root/my-openclaw` |

## Aux scripts

| Path | Purpose |
|------|---------|
| `script/debug/openclaw-service-inspect/main.go` | Automated inspect: status, PID, log tail, PASS/FAIL |

## Dry-run log

| Step | Time (UTC+8) | Result | Evidence |
|------|--------------|--------|----------|
| Prereq audit | 2026-07-08 23:52 | PASS | `remote-agent ping` → ok; openclaw in `service list` with bash error |
| 1 Build | 2026-07-08 23:58 | PASS | `ai-critic-server-linux-amd64` 18M; `go run ./script/bundle/for-linux/` exit 0 |
| 2 Deploy | 2026-07-08 23:59 | PASS | `upload-next` → v1; `server restart` → pong 401ms (brief 502 during handoff) |
| 3 Run | 2026-07-09 00:00 | PASS | `mkdir -p /root/my-openclaw` + restart → status running PID 26863 |
| 4 Inspect | 2026-07-09 00:00 | PASS | `go run ./script/debug/openclaw-service-inspect/` → PASS; log shows `[gateway] ready` |
| 5 Fix | 2026-07-08 23:57 | PASS | Root cause: missing `/root/my-openclaw` WorkingDir → misleading bash error |

## Handoff

Loop is agent-runnable. To iterate on a durable code fix (better error message,
auto-create WorkingDir, or `sh` fallback), run:

```
/loop-workflow openclaw service starts without bash fork/exec errors after deploy
```

Use this LOOP's inspect script as the verification gate.