# AI Critic

AI Critic is a developer tool that combines AI-powered code review with a mobile-friendly coding workspace featuring terminal access, port forwarding, and agent integration. It ships as a single Go binary with an embedded React frontend, designed to be accessed remotely via Cloudflare tunnels or other port-forwarding providers.

## Get Started

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
git clone https://github.com/xhd2015/ai-critic.git && cd ai-critic
docker build -t ai-critic . && docker run -it --rm -p 23712:23712 ai-critic
```
