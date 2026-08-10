# Scenario

**Feature**: local-only agent-run subcommands rejected by remote-agent

```
agent-run focus|web|assets|pty … -> Error: local-only / not available via remote-agent
```

## Preconditions

- Pure CLI path; no server inject required.
- Commands may be accepted by parser then rejected with clear message.
