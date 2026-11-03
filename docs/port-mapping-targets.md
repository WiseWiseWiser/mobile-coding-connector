# HTTP(S) origin targets for port mappings

A mapping can publish an HTTP(S) service reachable **from the ai-critic server**, not only a service on its localhost. Example:

```
browser → Pi public hostname → Pi loopback origin hop → Mac Ethernet address:3080
```

The Mac does not need a public hostname. This feature does not configure Ethernet, change an application's bind address, relax trusted-host checks, or restart DSH.

## CLI

Both `remote-agent` and `local-agent` share these commands:

```sh
# Existing localhost commands still work (defaults: http + 127.0.0.1).
remote-agent port mapping add 3080 --hostname pi.example.com --provider owned

# Illustrative IPv4; configure/verify this address separately before use.
remote-agent port mapping add 3080 --target-host 192.168.1.1 \
  --target-scheme http --hostname mac.example.com --provider owned --require-auth

# DNS is resolved on the server, not on the CLI machine.
remote-agent port mapping add 3080 --target-host macbook.local \
  --hostname mac.example.com --require-auth

# A link-local IPv6 zone names the *Pi* interface. Do not pass Mac's %en7.
remote-agent port mapping add 3080 --target-host 'fe80::1234%eth0' \
  --hostname mac.example.com --require-auth

remote-agent port mapping list
remote-agent port mapping list --json
remote-agent port mapping update map_... --target-host macbook.local
remote-agent port mapping delete map_...
remote-agent port mapping delete map_... --forget

# Temporary visit; no durable mapping record.
remote-agent port visit 3080 --target-host macbook.local --idle 10m --detach
```

Use the actual ID shown by `mapping list`. A numeric update/delete selector is accepted only when it is unambiguous. In particular, Pi:3080 and Mac:3080 can coexist and must then be addressed by ID. Updating a target keeps its ID and preserves omitted hostname/provider/auth/target settings. Existing WebSocket/SSE clients disconnect when their mapping is replaced or removed and must reconnect.

JSON has no ANSI coloring and mapping tokens are hidden by default. To deliberately inspect saved tokens, use `mapping list --show-token` on an authenticated management connection; do not paste that output into logs. New network mappings should require authentication.

## Web

The port mapping form has target host and target protocol fields, a URL preview, and **Check target from server**. Non-local hosts turn on the recommended authentication default; users can explicitly turn it off. Cards show mapping ID and target separately from tunnel status. **Edit target / check** preserves existing authentication/provider; Stop and logs address the mapping by ID rather than port.

A green tunnel status means the public tunnel was established, **not** that the origin is healthy. The target check opens a TCP connection from the server; it does not validate HTTP behavior, TLS/application identity, Host/Origin policy, or WebSocket handshake success. An unreachable target may be saved and restored later; requests fail until routing/listening is fixed. HTTPS requests always verify the origin certificate. No insecure-TLS switch is provided.

## Transport behavior

Each explicit target mapping owns one persistent origin transport independent of the public provider's request sockets:

| Setting | Value |
| --- | --- |
| HTTP proxy environment | ignored (direct server→origin dial) |
| Idle connections / host | 8 |
| Maximum connections / host | 32 |
| Idle timeout | 90 seconds |
| DNS + connect deadline | 5 seconds |
| TCP keepalive | 30 seconds |
| TLS handshake | 10 seconds |
| Response headers | 30 seconds; no whole-stream timeout |

HTTP keep-alive is reused; HTTPS negotiates HTTP/2 when supported. WebSocket upgrades retain dedicated bidirectional connections; text/binary frames and subprotocols pass through. SSE flushes immediately and follows downstream cancellation. Active visit streams prevent idle expiry. Removing/updating a mapping closes its idle, active, and upgraded origin sockets. Non-idempotent requests are not blindly replayed.

The public HTTP Host and Origin are preserved. Configure the origin application's allowed/trusted hosts for the public hostname separately. Mapping login cookies and bearer credentials consumed by the mapping gate are stripped before the origin request; application cookies remain intact.

Management Cookie filtering uses this ai-critic process's configured name. If the Pi proxies another ai-critic with the default `ai-critic-token`, start the Pi with `--auth-cookie-name pi-agent-token` so its management Cookie does not collide with the Mac's native session. This does not add/remove a mapping gate or change Bearer tokens. See [management Cookie names](auth-cookie-name.md) for restart preservation and browser-session migration. Leaving both instances on the same name still causes the upstream Cookie to be filtered.

Compatibility exception: an HTTP mapping to `127.0.0.1` on this ai-critic server's own management port, without an extra mapping auth gate, retains the legacy direct-loopback provider path when created or restored. Its management bearer/cookie is intended for the same management application and must not be stripped. Network origins, other local application ports, HTTPS, and auth-gated mappings do not receive this exception.

Cloudflare's edge proxy still uses its existing per-request socket protocol. Connection pooling here is specifically the Pi→Mac origin hop; it does not change that protocol or promise reuse of the browser→edge connection.

## Validation and persistence

HTTP/HTTPS only. The host field accepts a bare IPv4, DNS name, IPv6 address (optionally bracketed), or raw scoped IPv6 literal such as `fe80::1234%eth0`; it is not a URL, path, userinfo, or host:port field. Previews encode the zone as `%25eth0` in URLs. Link-local IPv6 requires a zone. Private/loopback/link-local addresses are allowed intentionally; unspecified, multicast, known metadata endpoints, and recursive origin-hop destinations are denied. DNS answers are revalidated at each new connection and dialing uses the validated IP, not a second lookup. DNS names may still depend on server-side DNS/mDNS configuration.

The versioned canonical store is `port-mappings-v2.json` alongside the existing mapping files, written atomically with mode 0600. First load imports the union of legacy names/auth/desired records as localhost mappings (`map_local_<port>`) and saves exact `.pre-target-v2` snapshots. Name-only legacy entries stay inactive. Old files are migration inputs only after v2 exists; the legacy HTTP names/auth APIs are adapters over the canonical store. A saved name is separate from the desired hostname so forgetting a name does not accidentally change an active tunnel. Deletes set desired=false; `--forget` atomically removes the record. A disabled record is not resurrected by stale legacy files.

### Recovery / downgrade

Back up the whole data directory before upgrading. Stop the server before manually restoring or editing its mapping store. To intentionally downgrade, restore the pre-target snapshots as a consistent set and archive the v2 store first. A legacy version cannot represent network targets or same-port multi-host mappings. Do not delete v2 while old desired-state files remain unless you intentionally want to re-import their pre-upgrade state.

## Verification

Target normalization, migration/snapshots/mode, ID ambiguity, partial update/auth preservation, restart restore and delete tombstones are covered by Go tests. Local HTTP fixtures verify that 10 sequential requests reuse one origin TCP connection; other fixtures cover TLS rejection/HTTP2, self-reference/metadata rejection, authenticated WebSocket binary/subprotocol roundtrip and cleanup, SSE flushing/cancellation, and local service ownership when a remote origin uses the same port. Browser fixtures exercise scoped IPv6 preview, server-side diagnostics, target editing and ID-safe removal.

```sh
go test -race ./server/auth ./server/config ./server/proxy/origin ./server/proxy/portforward ./server/services/authproxy
go test ./cmd/agentcli -run 'Test.*(Port|Target)'
go test ./server/services -run 'Test(EnsurePortForward|ServiceForward)'
cd ai-critic-react
npm run build
CHROME_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' \
  node --test --test-concurrency=1 tests/ui/port-target.test.mjs tests/ui/bootstrap-port.test.mjs
```

These fixture tests do not prove the actual Pi Ethernet route or a live Cloudflare hostname. A real deployment still needs an externally verified public URL, reachable Mac listener, correct scoped interface, and approved trusted-host configuration. No such network/application changes are performed by the feature or its tests.

### Broader regression caveats observed on the development Mac

The focused commands above pass. Broader package runs also encountered failures outside the target feature: `TestRunServiceUpgradeUploadsBeforeRemoteUpgrade` has an HTTP chunk-upload fixture but the upload client requests `/api/files/upload/ws`; `TestProcessGroupIDResolvesParentGroupForChild` and `TestStopProcessGroupKillsNonLeaderListenerPID` fail when their immediate `pgrep -P` finds no child process. Those test/upload/process-management paths were not changed to hide the failures. This is not a claim that the entire repository test suite is green. A broad `go build ./...` additionally hits the unchanged debug script `script/debug/openclaw-service-inspect` calling `ListServices(string)` against a no-argument client API; the server and both agent entrypoints are built separately.
