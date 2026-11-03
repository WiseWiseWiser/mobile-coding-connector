# Scenario

A custom-name Pi proxy forwards an unchanged default-name Mac's browser login Cookie.

## Preconditions

The fixture uses production auth handlers in two separate processes, private credentials, and random loopback ports.

## Steps

Login, refresh, reject the Pi legacy Cookie, preserve Bearer behavior, filter the Pi Cookie, and relogin to Pi without expiring the Mac session.

## Context

Real process boundary is required by per-process auth configuration. This does not verify Chrome UI or the live public Pi mapping.

