# Scenario

**Feature**: the registry file that holds every menu-bar usage item

```
missing file -> Seed() (grok + codex, rotating) -> ~/.ai-critic/usage-items.json
```

## Preconditions

1. The service seeds grok and codex when no registry file exists yet.

## Steps

1. Set `Op=registry` or `Op=list`.

## Context

Today's behavior is the starting point: two providers, rotation on, grok shown first.
