# Scenario

**Feature**: attach hop idle keepalive via WebSocket Ping

```
live hold attach + PingInterval inject (~50ms)
  -> client observes PingMessage while sending zero PTY bytes
```

## Preconditions

- Product sends WebSocket **control** Ping frames at Options.PingInterval
  (not written into PTY).
- L2 injects short interval so tests do not wait 30s.

## Steps

1. Leaf sets keepalive inject and hold FakeAttach.
