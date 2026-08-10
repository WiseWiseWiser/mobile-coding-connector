# Scenario

**Feature**: remote-agent agent-run resume (P3 library gates)

```
resume [ --open ] <session-id>
  -> live/unbound errors | inject success | --open → attach path
```

## Preconditions

- Resume uses server library path (agentrunapi) behind `ResumeSession` inject.
- No real grok / agent-run binary exec in L2.
- `--open` should invoke attach after successful resume when TTY ready.

## Steps

1. Leaf sets Seeds, ResumeInject, optional WantAttachOnResume, CLI Args.
