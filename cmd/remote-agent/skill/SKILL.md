---
name: remote-agent
description: >-
  Use the remote-agent CLI to operate an ai-critic server remotely: upload
  files, run commands, manage git repos, inspect proxies, build the next
  server binary, and restart the server.
---

# Remote Agent Skill

Use `remote-agent` when you need to control an `ai-critic` server over HTTP from
the terminal.

## Getting Started

Configure the target server and auth token:

```bash
remote-agent config
```

After that, `remote-agent` commands can use the saved default domain without
needing `--server` and `--token` each time.

## Commands

### Upload Files

Upload a local file to the remote server:

```bash
remote-agent upload ./ai-critic-server /tmp/ai-critic-server
remote-agent upload ./bundle.tar.gz /tmp/
```

If the local file is executable, the uploaded file is marked executable on the
remote server as part of the upload flow.

### Execute Remote Commands

Run a command on the remote server and stream stdout/stderr live:

```bash
remote-agent exec ls -la /tmp
remote-agent exec sh -c 'uname -a && whoami'
```

### Manage Remote Git Repositories

Clone, fetch, pull, or push repositories on the remote machine:

```bash
remote-agent git clone https://github.com/example/project.git
remote-agent git -C ~/project fetch
remote-agent git -C ~/project pull
remote-agent git -C ~/project push
```

### Server Management

Trigger the same server-management actions exposed by the Manage Server page:

```bash
remote-agent server build-next
remote-agent server build-next --project my-project-id
remote-agent server restart
```

`build-next` streams build logs from `/api/build/build-next`, and `restart`
streams restart progress from `/api/server/exec-restart`.

### Inspect Proxy Configuration

List proxy servers configured on the remote `ai-critic` server:

```bash
remote-agent proxy list
```