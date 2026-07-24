# Security Policy

## Supported versions

Blip is a single-branch project. Only the latest commit on `main` receives security fixes; there are
no maintained release branches and no backports to older tags.

## Reporting a vulnerability

**Do not open a public issue for a security problem.**

Report privately through GitHub's
[private security advisories](https://github.com/maltemindedal/blip/security/advisories/new) on this
repository. That is the intended channel — there is no separate security mailing address.

Include:

- A description of the issue and the component it affects.
- Steps to reproduce, ideally with a minimal request or client script.
- The impact you believe it has, and any configuration required to trigger it.
- The commit or version you tested against.

## What to expect

This is a personally maintained project, not a staffed product, so response times are best effort:

- An acknowledgement that the report was received.
- An assessment of whether the issue is reproducible and in scope.
- A fix on `main` for confirmed issues, and credit in the advisory unless you ask otherwise.

If you do not hear back within a couple of weeks, feel free to follow up on the advisory thread.

## Scope

In scope: anything reachable through the HTTP and WebSocket surface described in the
[API reference](docs/reference/api.md) — the origin check, the message size limit, the per-connection
rate limiter, the shutdown path, and the built-in test page at `/test`.

Out of scope, because they are documented properties of the design rather than defects:

- **No authentication or authorization.** Blip has none by design; enforce it in front of `/ws`. See
  [Security hardening](docs/guides/security-hardening.md).
- **No TLS.** The server speaks plain HTTP and expects a reverse proxy to terminate TLS. See
  [Deploying to production](docs/guides/deploying-to-production.md).
- **No message persistence, no sender identity, and no delivery guarantee.** See
  [API reference](docs/reference/api.md#delivery-semantics).
- **`/test` being publicly reachable.** It is a development aid with no disable flag; block it at the
  proxy in production.
- **Denial of service from a single-instance design.** The hub lives in process memory and the
  process is not clustered.

## Related

- [Security hardening](docs/guides/security-hardening.md) — the controls and how to configure them
- [Configuration reference](docs/reference/configuration.md) — defaults and parsing rules
