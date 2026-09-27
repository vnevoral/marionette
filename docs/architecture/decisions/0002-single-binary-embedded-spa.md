# ADR-0002: A single Go binary with an embedded SPA

- **Status**: Accepted
- **Date**: 2026-09-25

## Context

The target platform is a Raspberry Pi with Ubuntu (linux/arm64) and, in
general, a low-power/constrained environment where we do not want to
require installing Node.js, a web server (nginx) or any other runtime just
to run the UI (see FR-01, FR-02, NFR-02).

## Decision

The backend is written in Go and the build produces a single static binary.
The frontend (Vue 3 SPA) is built with Vite into `web/dist` and embedded
into the binary using `go:embed` (`internal/webui`). The HTTP server
(`internal/server`) serves both the API and the SPA's static assets from the
same process and port, with a fallback to `index.html` for client-side
routing (Vue Router).

## Considered alternatives

- A separate Node.js server for the UI + Go API — rejected, requires an
  extra runtime and process on the target (NFR-02).
- Serving the UI via nginx/Apache next to the Go API — rejected for the
  same reason, unnecessary operational complexity on a Raspberry Pi.
- SSR/Nuxt — rejected, unnecessary complexity for an internal dashboard
  with no SEO requirement.

## Consequences

- The release artifact is one file per platform (`bin/marionette`,
  `bin/marionette-linux-arm64`), installation = copy + run (+ systemd
  unit, see roadmap Phase 7).
- The build pipeline must always build the UI first (`make ui-build`)
  before `go build`, otherwise the embed fails or contains stale content —
  handled by the order of targets in the `Makefile`.
- Development mode (`ui-dev` + `backend-dev`) runs separately via a Vite
  proxy on `/api`; the production build is always unified into a single
  process.
