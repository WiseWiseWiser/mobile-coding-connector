# Script Index

This directory contains project maintenance scripts for building, running, debugging, release packaging, environment setup, and experiments.

Most scripts are run with:

```sh
go run ./script/<path>
```

Example: `go run ./script/server/run`

## Build and Run

- `build` - Build frontend (`ai-critic-react`) and then build the Go server.
- `release` - Build frontend once, then cross-compile release binaries for `linux/amd64` and `linux/arm64`.
- `run` - Start local dev mode (build server, start Vite, run server with `--dev`).
- `run/quick-test` - Start quick-test server workflow (auto-kill/restart, optional Vite control).
- `server/run` - Build and run backend only, proxying frontend requests to local Vite.
- `server/build` - Build Go server binary (default output `/tmp/ai-critic`).
- `server/build/for-linux-amd64` - Build frontend and cross-compile server for `linux/amd64`.
- `vite/run` - Run Vite dev server for frontend.
- `vite/build` - Build frontend static assets (`ai-critic-react/dist` by default).
- `vite/stop` - Kill process(es) bound to Vite default port `5173`.

## Debug and Inspection

- `debug-server-and-frontend` - Start quick-test server stack and launch browser debugging flow.
- `debug-port` - Puppeteer helper for scripted browser checks against a target local port.
- `browser-debug` - Interactive Chrome DevTools-based debug REPL (headers, screenshots, eval, API calls).
- `request` - Send authenticated HTTP requests to local ai-critic server endpoints.

## Security, Auth, and Tunnels

- `crypto/gen` - Generate RSA keypair used for encrypted SSH private key transport.
- `cloudflare/setup` - Configure/check Cloudflare tunnel setup from `.config.local.json`.

## Sandbox Environments

- `sandbox/create` - Create/start Debian podman sandbox container and open interactive shell.
- `sandbox/fresh-setup` - Build frontend+server, create sandbox container, copy binary, and run on port `8899`.

## Skill Sync Utilities

- `skills/sync/cursor` - Sync project `skills/` into `.cursor/skills/`.
- `skills/sync/opencode` - Sync project `skills/` into `.opencode/skills/`.

## Bug Replication Experiments

- `replicate-bugs/exec-debug/replicate-exec-bug` - Reproduces state-loss behavior around `syscall.Exec`.
- `replicate-bugs/exec-debug/replicate-exec-bug-proper` - Demonstrates expected state restoration by re-parsing args.
- `replicate-bugs/exec-debug/replicate-exec-bug-fixed` - Shows recoverability pattern and documents fix direction.

## Shared Script Library (Not Runnable Directly)

- `lib/build_server.go` - Common frontend/server build helpers.
- `lib/constants.go` - Shared script constants (ports, binary names).
- `lib/nodejs.go` - Node/NPM helper utilities (node_modules checks, Node 20 wrapper).
- `lib/podman.go` - Podman machine/container helpers.
- `lib/quicktest.go` - Quick-test orchestration helpers.
- `lib/run.go` - Port/PID probing and kill helpers.
- `lib/skill_sync.go` - Skill directory sync implementation.
- `lib/token.go` - Credential token loading helpers.

## Supporting Files

- `debug-port/debug.js` - Node/Puppeteer runtime script used by `debug-port`.
- `debug-port/package.json` - Node dependencies for `debug-port`.
- `browser-debug/README.md` - Detailed usage docs for `browser-debug`.
