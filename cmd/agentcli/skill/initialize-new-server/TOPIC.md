---
name: remote-agent/initialize-new-server
description: >-
  Bootstrap ai-critic-server on a new Linux host over SSH: detect OS/arch,
  cross-build, upload, systemd or keep-alive, setup token; optional domain
  and remote-agent alias; cloudflared login is a human gate.
---

# Initialize a new server

First `ai-critic-server` on a host that has **no** HTTP API yet. Transport is
**SSH** (or equivalent: remote exec + file upload). `remote-agent exec` /
`upload` / `server upgrade` need a live server — use topic **server** then.

## When / not

| Use | Not |
|-----|-----|
| Bare Linux, `ping`/`remote-agent ping` fails | Host already serves `/ping` → topic **server** |
| Need systemd/keep-alive + first credential | App on an **existing** ai-critic → **deploy-service** |
| Cross-build the **server** binary | Cross-build some other CLI onto PATH → **install** |

Ask once (required / optional). Missing optional → skip that step; do not invent.

| Ask | Required | Skip if omitted |
|-----|----------|-----------------|
| SSH target (`user@host`, key/askpass) | yes | — |
| Public domain | no | step **Domain** |
| Alias (`xxx-agent`) | no | step **Alias** |

## Detect (SSH)

```bash
ssh "$HOST" 'uname -s; uname -m; command -v systemctl; sudo -n true; command -v cloudflared'
```

| `uname -m` | Build |
|------------|--------|
| `x86_64` / `amd64` | `GOOS=linux GOARCH=amd64` (`go run ./script/bundle/for-linux/` ok) |
| `aarch64` / `arm64` | `GOOS=linux GOARCH=arm64` |
| `armv7l` | `GOOS=linux GOARCH=arm GOARM=7` — **no** `install.sh` / release asset |

Always `CGO_ENABLED=0`. Output: `ai-critic-server-<goos>-<goarch>` from the
ai-critic module (frontend `ai-critic-react/dist` must exist; build it if missing).

## Upload + run

Dest: `/root/servers/ai-critic/ai-critic-server` (workdir = dirname of binary).

`scp user@fe80::addr%iface` often fails. Prefer:

```bash
ssh "$HOST" 'cat > /tmp/ai-critic-server && chmod +x /tmp/ai-critic-server' < ./ai-critic-server-linux-arm
ssh "$HOST" 'sudo mkdir -p /root/servers/ai-critic && sudo mv /tmp/ai-critic-server /root/servers/ai-critic/ai-critic-server'
```

| Remote has | Start |
|------------|--------|
| `systemctl` + sudo | `sudo /root/servers/ai-critic/ai-critic-server integration systemctl start` |
| else | `nohup /root/servers/ai-critic/ai-critic-server keep-alive &` |

Verify on the host: `curl -fsS -m 5 http://127.0.0.1:23712/ping` → `pong`.
Then the same URL from the Mac (IPv6 link-local: `http://[addr%iface]:23712/ping`).

## Credential

Do **not** `POST /api/auth/credentials/generate` first (can 401 while
uninitialized). Generate locally, setup, never print the token:

```bash
TOKEN=$(openssl rand -hex 32)
curl -fsS -X POST "$ORIGIN/api/auth/setup" \
  -H 'Content-Type: application/json' \
  -d "{\"credential\":\"$TOKEN\"}"
```

`{"status":"ok"}`. Later `GET /api/auth/status` is `initialized: true`.

## Domain (optional)

Only if the user named a domain. If `cloudflared` is missing or not logged in
(`cloudflared tunnel list` fails): **stop and ask the user to install/login**.
Do not automate login. After login, point the hostname at `http://127.0.0.1:23712`
(then topic **cloudflared** / **cloudflare-proxy** if switching backends).

## Alias (optional)

Only if the user named `xxx-agent`. IPv6 zone in the URL must be `%25`:

```bash
remote-agent alias add rpi --server 'http://[fe80::…%25en7]:23712'
printf '%s\n' "$TOKEN" | rpi-agent config set --token-stdin
rpi-agent ping
```

Then topic **alias**. Token stays in config; not in skill text.

## Wrong → correct

| Wrong | Correct |
|-------|---------|
| `remote-agent exec` / `upload` to create the first server | SSH (no HTTP API yet) |
| `install.sh` / `bundle/for-linux` on `armv7l` | `GOARCH=arm GOARM=7 CGO_ENABLED=0` |
| `POST /api/auth/credentials/generate` then setup | `POST /api/auth/setup` with a local hex token |
| `scp` to `user@fe80::addr%iface` | `ssh … 'cat > dest' < bin` |
| Alias URL `…%en7]:23712` | `…%25en7]:23712` |
| **deploy-service** for the core server | that topic is an **app** on a live ai-critic |

## Pitfalls

- Chicken-and-egg: this topic is Mac-side SSH; after `pong`, prefer `remote-agent`.
- `integration systemctl start` may return before `/ping` listens — retry a few seconds.
- Direct-cable IPv4LL vs a static `192.168.x` on the Pi: use the family that actually pings.
