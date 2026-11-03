# Management authentication Cookie names

Use `--auth-cookie-name NAME` when an ai-critic instance proxies another application
that uses the same login Cookie name. For example, configure the Pi management
instance as `pi-agent-token` while leaving the Mac instance at its default
`ai-critic-token`.

This changes **the Cookie name**, not a password/token, credential-file format,
Bearer authentication, or a mapping's optional authentication gate.

## Startup

```sh
# Direct server startup; retain all your other existing startup arguments.
ai-critic --port 23712 --auth-cookie-name pi-agent-token

# keep-alive forwards the setting to its child and subsequent restarts.
ai-critic keep-alive --port 23712 --auth-cookie-name pi-agent-token

# Read-only Linux/systemd preview.
ai-critic integration systemctl show-config \
  --port 23712 --auth-cookie-name pi-agent-token

# Explicitly write the unit and restart: run only during an approved deployment.
ai-critic integration systemctl start \
  --port 23712 --auth-cookie-name pi-agent-token
```

The default remains `ai-critic-token`; no option preserves existing browser
protocols. An explicitly empty value is an error rather than a default override.
Names must be valid HTTP Cookie names. The service-gate name `svc-auth-token` is
reserved. Browser security prefixes `__Host-`, `__Secure-`, and `__Http-` are rejected
because this server does not issue the required Secure cookies. These checks occur
before opening the service listener, starting a keep-alive child, or writing a unit.

The configured name is process-wide and set at startup, not a dynamic API or a
per-mapping override. No environment-variable or JSON-config alternative is added.
The durable source is the real startup definition (for example systemd ExecStart
or the keep-alive launcher argv). Shell/systemd arguments are quoted appropriately.

## Authentication and forwarding

For a Pi started with `--auth-cookie-name pi-agent-token`:

| Surface | Behavior |
| --- | --- |
| Pi login | Issues `pi-agent-token`; retains Path=/, HttpOnly, SameSite=Lax, and existing lifetime |
| Pi API auth/check/status | Reads only `pi-agent-token`, with no default-name fallback |
| Pi management Bearer | Same tokens and behavior as before |
| Pi origin proxy request | Removes `pi-agent-token`; preserves Mac's `ai-critic-token` and other app cookies |
| Pi origin proxy Bearer | Continues removing valid Pi management Bearer credentials and preserving unrelated app Bearers |
| Upstream Set-Cookie | Passes through unchanged; no proxy-side renaming or deletion |
| Optional mapping/service gate | Remains separate; its own cookies/consumed credentials are still stripped |
| Internal management HTTP clients | Use Bearer, not a hardcoded browser Cookie name |

The Mac mapping can remain `--clear-auth`: native Mac authentication protects the
application; the Pi does not add a second login. Conversely, changing the Pi's
management Cookie name does not remove DSH/eluc mapping authentication.

Names must actually differ: configuring both Pi and upstream to the same name
still causes that upstream Cookie to be filtered. This option is namespace
configuration, not an automatic origin/application detection mechanism.

The localhost management-application compatibility exception documented in
[port mapping targets](port-mapping-targets.md) remains narrow and unchanged.
A remote Mac using the same numeric port 23712 does not qualify for it.

## Existing browser sessions

Switching a name deliberately does not accept the old name as an alias. Pi users
with a default-name session must sign in again with their **existing** credentials.
The old name is ignored, not automatically expired: it might belong to another
application on the same host, and browser cookies are not isolated by port. Login
writes only this instance's configured Cookie. Neither the management handler nor
the origin proxy deletes another namespace; stale Pi cookies can be removed
manually in the browser if desired.

Cookie names are not secrets; startup diagnostics may print the name and a
non-fatal re-login reminder. Passwords, Cookie values, Bearer tokens, and login
bodies must not be printed. Changing a Cookie name does not revoke or rotate a
token; existing management CLI Bearers remain valid.

## Startup preservation and safe rollout

1. Build/test the new binary without modifying running services.
2. Verify the Pi's actual supervisor and recovery route. If keep-alive is used,
   update the management parent as well as its server child: an old parent still
   has old Cookie-based management calls.
3. Explicitly add the option to the persistent startup definition. Upgrading a
   binary alone retains its argv and does not invent a new Cookie name.
4. Perform an approved restart/switch. The Pi management entry and forwarded
   applications may briefly disconnect. Mac services do not need to restart.
5. Verify native Pi login, existing CLI Bearer, Mac browser login/refresh, and the
   existing DSH/eluc mapping gates.

Once a generated systemd unit contains `--auth-cookie-name`, invoking the generator's
`start` without that option fails **before changing the unit or running systemctl**.
Repeat the desired option explicitly, or use native `systemctl restart
ai-critic-server` to reuse the installed ExecStart. A default `show-config` preview
warns when it differs from an installed explicit Cookie setting; its printed
maintenance commands carry both port and Cookie choice. To deliberately return to
the default, pass `--auth-cookie-name ai-critic-token` explicitly.

Roll back the **binary and startup definition together**: older binaries cannot
parse the new option. No credential, encryption-key, or mapping-store migration is
needed, and no token resets should be used to work around a forwarding problem.

## Developer HTTP/browser tools

The HTTP request script and keep-alive management actions use Bearer. A browser
Cookie injection tool must know the destination instance's Cookie name:

```sh
(cd script/browser-debug && go run . --auth-cookie-name pi-agent-token URL)
```

The debug tool retains the old default when omitted and logs only the injected
Cookie's name, never its value. Normal Web UI login does not need this tool or a
frontend setting: the browser processes the server's Set-Cookie automatically.

## Verification

```sh
go test -race ./run ./run/daemon ./server/auth ./server/proxy/origin ./server/services/authproxy
go test -race ./server/config ./server/proxy/portforward
go test -race ./server -run 'Test(Restart|ShutdownWithoutRestart|ExecRestart)'
(cd script/browser-debug && go test -race ./...)
doctest vet ./tests/auth-cookie-name
doctest test --label-all ./tests/auth-cookie-name/...
```

The doctest's primary e2e leaf runs **two independent auth processes** using the
production auth/proxy handlers, different credential files, distinct browser
Cookie domains, and ephemeral loopback listeners. It tests login, refresh,
legacy-name rejection, management credential stripping, and Pi relogin without
expiring the Mac session. It does not start full-server background services or
prove a live Cloudflare/Pi deployment. Go regressions additionally cover the
systemd writer, keep-alive argv, Cookie combinations, internal management HTTP
requests, and the existing shared streaming transport.
