# Scenario

**Feature**: remote-agent agent-run send (P4)

```
send [opts] <session-id> "message" -> msg id | error
```

## Preconditions

- Live session seed + SendInject for L2.
- No real TTY; inject enqueues.

## Steps

1. Leaf sets CLI Args and SendInject.
