# Scenario

**Feature**: remote-agent agent-run run (P5 library)

```
run [OPTIONS] ["prompt"] -> inject RunSession | validation | --new-terminal reject
```

## Preconditions

- L2 uses Options.RunSession inject (no real grok / agent-run exec).
- Prefer --detach success path (no interactive attach).
