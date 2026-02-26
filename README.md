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

Open http://localhost:23712 in your browser. On first launch, you'll see the [Initial Setup](#initial-setup) page to create a login credential.

To access from public domain:

```sh
# Option 1: Cloudflare Quick Tunnel
cloudflared tunnel --url http://localhost:23712

# Option 2: localtunnel
npx localtunnel --port 23712
```

## Initial Setup

On first launch, the server has no credentials and is in an **uninitialized** state. Any API request returns `{"error": "not_initialized"}`, and the frontend automatically shows the **Setup** page.

### Step 1: Create Credential

Open the browser and you'll see the Setup page. You have two options:

- **Generate Random**: Click the "Generate Random" button. The server generates a secure 64-character hex token (32 random bytes → SHA-256 hash).
- **Enter Manually**: Type your own credential into the input field.

> **Important**: Copy and save the credential somewhere safe before confirming. This is your login password — if you lose it, you'll need to manually edit the credentials file on the server.

Click **"Confirm & Continue"** to finalize. The credential is written to `.ai-critic/server-credentials` (permissions `0600`).

### Step 2: Log In

After setup, you're redirected to the **Login** page. Enter any username and paste the credential as the password. On success, the server sets an `ai-critic-token` cookie (valid for 1 year).

You can also authenticate via the `Authorization: Bearer <credential>` header for API access.

### Step 3: Configure AI Models (Optional)

To enable AI code review, configure at least one AI provider. Go to **Settings → AI Models** in the UI, or manually create `.ai-critic/ai-models.json`:

```json
{
  "providers": [
    {
      "name": "openai",
      "base_url": "https://api.openai.com/v1",
      "api_key": "sk-..."
    }
  ],
  "models": [
    {
      "provider": "openai",
      "model": "gpt-4o"
    }
  ],
  "default_provider": "openai",
  "default_model": "gpt-4o"
}
```

### Step 4: Generate Encryption Keys (Optional)

Encryption keys enable secure transmission of sensitive data (e.g., SSH private keys) from the browser to the server. Generate them from the UI (**Settings → Encryption**) or via CLI:

```bash
go run ./script/crypto/gen
```

This creates an RSA key pair at `.ai-critic/enc-key` and `.ai-critic/enc-key.pub`.

### Data Directory

All server state is stored under `.ai-critic/` in the working directory:

| File | Description |
|------|-------------|
| `server-credentials` | Login credentials (one token per line) |
| `enc-key` / `enc-key.pub` | RSA key pair for frontend encryption |
| `ai-models.json` | AI provider and model configuration |
| `server-project.json` | Server project directory setting |
| `agents.json` | Agent configurations |
| `terminal-config.json` | Terminal settings and extra PATH entries |
| `server-domains.json` | Domain/tunnel mappings |
| `projects.json` | Registered projects |

### Managing Credentials

You can add more credentials after setup:

- **UI**: Settings → Manage Credentials → Add
- **API**: `POST /api/auth/credentials/add` with `{"token": "your-token"}`
- **Manual**: Append a new line to `.ai-critic/server-credentials`

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
