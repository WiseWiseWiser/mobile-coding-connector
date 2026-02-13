# AI Critic

AI Critic is a developer tool that combines AI-powered code review with a mobile-friendly coding workspace featuring terminal access, port forwarding, and agent integration. It ships as a single Go binary with an embedded React frontend, designed to be accessed remotely via Cloudflare tunnels or other port-forwarding providers.

## Install

Download the latest release for your platform (Linux amd64/arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/WiseWiseWiser/mobile-coding-connector/master/install.sh | bash
```

This downloads the `ai-critic-server` binary to the current directory.

## Start

Start the server:

```bash
./ai-critic-server
```

Open http://localhost:23712 in your browser. On first launch, you'll be prompted to set up an initial credential to secure your server.

To access from public domain:

```sh
# Option 1: Cloudflare Quick Tunnel
cloudflared tunnel --url http://localhost:23712

# Option 2: localtunnel
npx localtunnel --port 23712
```

On initial login, you will be prompted to setup login password, the initial password will be generated using secure sha256sum, you should keep it somewhere else to get logged in.

## Run with Keep Alive Daemon
If the server panics, the process ends. To make it auto restart, add a `keep-alive` sub command:

```sh
# The keep-alive daemon must be run in the background with nohup to ensure it survives
# terminal disconnections and continues running after you log out.
nohup ./ai-critic-server keep-alive &
```

**Why `nohup` and `&`?**
- `nohup` (no hangup) prevents the process from being terminated when the terminal session ends (e.g., SSH logout)
- `&` runs the process in the background so your shell remains available

This ensures the keep-alive daemon persists across terminal sessions and can automatically restart the server if it crashes.

## Get Started with Docker

Quick demo with one command (Docker or Podman):

```bash
docker run -it --rm -p 23712:23712 ghcr.io/xhd2015/ai-critic
```

```bash
podman run -it --rm -p 23712:23712 ghcr.io/xhd2015/ai-critic
```

Then open http://localhost:23712 in your browser.

### Build from source

```bash
git clone https://github.com/WiseWiseWiser/mobile-coding-connector.git && cd mobile-coding-connector
go run ./script/build
```

The binary is built to `/tmp/ai-critic`. Run it with:

```bash
nohup /tmp/ai-critic keep-alive &
```
