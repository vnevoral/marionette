# Implementation block: Code review fixes — backend (same-origin behind a proxy, allowlist, store)

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: NFR-12, FR-30..33 (persistence), NFR-05
- **ADRs**: —
- **Status**: Done
- **Dependencies**: Blocks 0026, 0028 (persistence), 0034

## Goal

Once done, the NFR-12 protection also works behind a TLS-terminating reverse
proxy and the `MARIONETTE_ALLOWED_HOSTS` allowlist compares hosts with the
same rule as `Origin`; the config store does not falsely report `Dirty()`
after a failed write.

## Scope

- **In scope** (findings of the 2026-09-26 code review):
  1. `internal/server/origin.go`: the `Origin` × `Host` comparison does not
     depend on the connection's TLS state — the hostname is compared
     case-insensitively and the ports must match, or both be "default"
     (`80`, `443` or not specified). An explicit other port
     (`pi.local:8080`) must match.
  2. The `MARIONETTE_ALLOWED_HOSTS` allowlist uses the same normalization:
     `pi.local`, `pi.local:80` and `pi.local:443` in the list match the
     `Host` `pi.local`, `pi.local:80` and `pi.local:443`.
  3. `internal/config/store.go`: the rollback after an `OnChange` failure
     (`CreateCard`, `UpdateCard`, `DeleteCard`) reverts the `changes`
     counter, so `Dirty()` does not report a change that did not actually
     happen.
  4. `internal/config/types.go`: remove the `errorsAs` wrapper, call
     `errors.As` directly.
  5. Documentation: `docs/architecture/overview.md` (NFR-12 section), README
     (`MARIONETTE_ALLOWED_HOSTS`), NFR-12 in requirements (one sentence
     about the proxy).
- **Out of scope**:
  - trusting the `X-Forwarded-Proto` header / trusted proxy configuration;
  - authentication (NFR-01, outside the MVP);
  - frontend findings (block 0037).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a fix of the state ("plan and fix all the
  findings") after a code review of 18 commits on `main`. The rule
  "default ports are equivalent" is a deliberate compromise:
  `http://pi.local` and `https://pi.local` are both the operator's host;
  another service on the same hostname with its own port remains a foreign
  origin.

## Proposed solution

- `origin.go`: a new function `canonicalHost(host string) string` —
  lowercase, trim, `net.SplitHostPort`; port `80`/`443`/missing → hostname
  only (IPv6 without square brackets), otherwise `host:port`.
  `sameOrigin(origin, requestHost)` loses the `tls` parameter; the allowlist
  is stored and compared via `canonicalHost`. The comments of
  `requireSameOrigin`/`sameOrigin` describe the actual rule.
- `store.go`: in every rollback branch `store.changes--` instead of `++`
  (memory after the rollback matches the last saved state).
- `types.go`: `errors.As(err, &validation)` at the call site.

## Test plan

- `TestSameOriginComparison`: new rows — `https://pi.local` × `pi.local`
  → true, `http://pi.local` × `pi.local:443` → true, `https://pi.local:8443`
  × `pi.local` → false, `http://pi.local` × `pi.local:8080` → false.
- `TestAllowedHostsRestrictMutatingRequests`: allowlist `pi.local:80`
  accepts `Host: pi.local`, allowlist `pi.local` accepts
  `Host: pi.local:443`, `pi.local:8080` in the list does not accept
  `Host: pi.local`.
- `TestStoreRollbackKeepsDirtyHonest`: after an `OnChange` failure `Dirty()`
  is false if it was false before the attempt.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- a mutating request with `Origin: https://pi.local` and `Host: pi.local`
  without TLS is accepted;
- `docs/architecture/overview.md` no longer lists rejection behind a proxy
  as a known limitation.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (lint, `vue-tsc`, Prettier, Vitest,
  `vite build`, `go test -race`, `go vet`). `TestSameOriginComparison`
  extended with proxy cases (`https://pi.local` × `pi.local`,
  `pi.local:80`, `http://pi.local` × `pi.local:443` → match;
  `https://pi.local:8443` × `pi.local`, `http://pi.local` × `pi.local:8080`
  → no match), a new `TestCanonicalHost`,
  `TestAllowedHostsRestrictMutatingRequests` verifies `pi.local:80`
  in the list × `Host: pi.local`/`:443` and the rejection of `pi.local`
  without a port when only `pi.local:8080` is in the list. A new
  `TestStoreRollbackKeepsDirtyHonest` (after an `OnChange` failure on
  create, update and delete, `Dirty()` is false).
- **Deviations from the plan**: none. `sameOrigin` lost the `tls` parameter;
  `request.TLS` is no longer read in the middleware.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  (NFR-12 section: canonical host comparison, removed the "known limitation"
  behind a proxy), README (`MARIONETTE_ALLOWED_HOSTS`, paragraph about
  cross-site protection), `docs/requirements/requirements.md` NFR-12,
  roadmap.
