# Roadmap and implementation blocks

The planning and implementation process is bindingly described in
[docs/devops/development-workflow.md](../devops/development-workflow.md).

Phases correspond to the rough order of implementation. Before
implementation, each phase is broken down with the `/plan-block` prompt into
one or more concrete implementation blocks (see the
[template](template-implementation-block.md)) and saved to
`docs/plans/blocks/NNNN-name.md`. A block is implemented with the
`/implement-block` prompt.

## Current state

As of 2026-09-27 all phases below are **Done**, the last closed block is
[0060](blocks/0060-toolchain-go-and-golangci-lint-v2.md) and the current release is
**v1.2.1** (tags `v1.0.0`, `v1.1.0`, `v1.2.0`, `v1.2.1`; `v1.2.0` has no
GitHub Release because its CI run failed, fixed by block 0059). The
requirements are at v0.11. `v1.2.1` runs on the reference Raspberry Pi;
the upgrade and rollback procedure from the README was verified there by
the owner on 2026-09-27. New work starts as a new requirement and block
per the development workflow.

## Phase overview

| Phase | Name                                                                                          | Status                                             | Requirements                   |
| ----- | --------------------------------------------------------------------------------------------- | -------------------------------------------------- | ------------------------------ |
| 0     | Project bootstrap (repo, Go+Vue+PrimeVue skeleton, Makefile, embed, devops/agent scaffolding) | **Done**                                           | FR-01..04 (foundation)         |
| 1     | Requirements and architecture refinement                                                      | **Done** (requirements v0.2, ADR-0004 accepted)    | all (SRS review)               |
| 2     | Domain model + config store (in-memory + JSON persistence)                                    | **Done**                                           | FR-10..14, FR-30..33, ADR-0004 |
| 3     | Execution engine (safe execution of actions on the host)                                      | **Done**                                           | FR-11, FR-13, NFR-01, NFR-04   |
| 4     | Status/health-check engine (incl. optional polling)                                           | **Done**                                           | FR-12, FR-15, FR-16, FR-17     |
| 5     | REST API (card/action CRUD, execution, status reads)                                          | **Done**                                           | FR-40, FR-41                   |
| 6     | Dashboard UI (cards, status indicators, management forms)                                     | **Done**                                           | FR-20..23                      |
| 7     | Packaging and deployment (systemd unit, install script, arm64 release)                        | **Done** (2026-09-26; 0023, 0046)                  | FR-01..04, NFR-02              |
| 8     | Hardening (auth/access control, logging, error states, tests, documentation)                  | **Done** (2026-09-26; blocks 0024–0045)            | NFR-01, NFR-04..06, NFR-12     |
| 9     | UX redesign and shared design system                                                          | **Done**                                           | FR-24..29, NFR-08..10          |
| 10    | Real-time status delivery over SSE                                                            | **Done**                                           | FR-42, NFR-11                  |

## Planning notes

- Phase 1 is closed: `docs/requirements/requirements.md` was at v0.5 at
  that point (now v0.11; open questions resolved, see section 7 of the document) and
  [ADR-0004](../architecture/decisions/0004-action-card-domain-model.md) is
  accepted, including run history, output rules, polling and the
  concurrency limit.
- Phase 2 was broken down into implementation blocks and **approved** by the
  project owner (2026-09-25). Blocks 0001–0003 are done:
  - [0001 — Domain types and configuration validation](blocks/0001-config-domain-types.md)
  - [0002 — In-memory config store (card CRUD + run history)](blocks/0002-config-inmemory-store.md)
  - [0003 — JSON persistence of configuration, history and loading at startup](blocks/0003-config-json-persistence.md)

  The blocks are implemented in this order (0002 builds on the types from
  0001, 0003 on the store from 0002).

- Phase 3 was broken down into ADR-0005 and implementation blocks
  0004–0006, which are done:
  - [ADR-0005 — Safe action execution and concurrency limit](../architecture/decisions/0005-execution-engine.md)
  - [0004 — Process start and enforced timeout](blocks/0004-execution-process.md)
  - [0005 — Output capture and result evaluation](blocks/0005-execution-result.md)
  - [0006 — Global limit of concurrent actions](blocks/0006-execution-concurrency.md)

- ADR-0006 is accepted: the status history contains only actual state
  transitions with their duration; migration of older status histories is
  not part of the MVP.

- Phase 4 was broken down into blocks 0007–0009, which are done:
  - [0007 — Status projection and transition history](blocks/0007-status-projection.md)
  - [0008 — Status check service](blocks/0008-status-check-service.md)
  - [0009 — Standard and fast polling scheduler](blocks/0009-polling-scheduler.md)

- Phase 5 is broken down into blocks 0010–0012 in this order:
  - [0010 — REST API contract and HTTP transport](blocks/0010-rest-api-transport.md)
  - [0011 — Orchestration of the primary and status action](blocks/0011-action-orchestration.md)
  - [0012 — Application composition, reconcile and lifecycle](blocks/0012-application-composition-lifecycle.md)

  Blocks 0010–0012 are done.

- Phase 6 starts with block 0013:
  - [0013 — Card dashboard and action controls](blocks/0013-dashboard-overview.md)
  - [0014 — Action card management](blocks/0014-card-management.md)

- Phase 9 is set up as a separate UX/design area. [ADR-0007](../architecture/decisions/0007-ui-design-system-and-ux.md)
  is accepted and the default UI language is English. Before implementing
  the UX blocks, the detailed design of screens and interactions is approved
  in the [UX specification](../ux/ui-ux-specification.md).
  The first approved block is [0015 — UX-01: Design tokens and AppShell](blocks/0015-ux-foundation.md).
  It is followed by [0016 — UX-02: Overview and action feedback](blocks/0016-ux-overview.md).
  It is followed by [0017 — UX-03: Management CRUD workflow](blocks/0017-ux-management-crud.md).
  It is followed by [0018 — UX-04: Card detail, runs and status timeline](blocks/0018-ux-card-detail.md).
  The PrimeFlex layout foundation is captured in [0019](blocks/0019-primeflex-layout-foundation.md).
  It is followed by [0020 — PrimeFlex view migration](blocks/0020-primeflex-view-migration.md).
  The last completed block of the phase is [0021 — UX-05: Visual consistency,
  accessibility and responsive audit](blocks/0021-ux-consistency-accessibility.md).

- Phase 10 is set up by requirement FR-42/NFR-11. [ADR-0008](../architecture/decisions/0008-sse-status-event-stream.md)
  is accepted and the approved implementation block is [0022 — SSE: live
  status changes](blocks/0022-sse-status-events.md).

- Phase 7 is broken down in block [0023 — systemd deployment and release
  artifact](blocks/0023-deployment-systemd-release.md) for generic Linux with
  systemd; Ubuntu 24.x on a Raspberry Pi ARM64 serves as the reference
  validation environment.
  The block is **Done** (2026-09-26). In the devcontainer (x86_64,
  2026-09-26) the following passed: `systemd-analyze verify`, a real
  installation (which revealed and fixed a wrong binary path in
  `install.sh`), health, cross-site protection, graceful stop with history
  saved, restart with data preserved and repeated installation.
  On the reference Raspberry Pi (Ubuntu, ARM64) the project owner verified
  release `9bb21bf` on 2026-09-26: installation, `systemctl` restart,
  `Restart=on-failure`, startup after reboot and a WoL + ping card. Since
  2026-09-26 the installer has an automated staged test in `make test` and
  CI.
  Validation on the Raspberry Pi (2026-09-26) revealed that the `ping`
  status action fails in the hardened unit if the host forbids unprivileged
  ICMP (`net.ipv4.ping_group_range`). This is addressed by block
  [0046 — Unprivileged ping check during installation](blocks/0046-install-unprivileged-ping-check.md)
  (FR-05) — **Done** (2026-09-26); the installer prints instructions if the
  host forbids unprivileged ping, and the README describes verification and
  remediation.

- On 2026-09-26, Phase 8 was broken down, based on a project and code
  review, into blocks 0024–0034 and **approved** by the owner the same day.
  These are mostly bug fixes and bringing the state in line with the
  requirements/ADRs, not new functionality; nevertheless every block goes
  through the standard `/implement-block` and DoD. The blocks are split into
  three streams that can be implemented concurrently; within a stream the
  listed order applies:

  **Stream A — backend hardening (critical fixes first):**
  1. [0024 — Execution engine: process group termination and context propagation](blocks/0024-exec-process-group-termination.md)
     — **Done** (2026-09-26); timeout kills the whole process group, new
     run result `canceled`.
  2. [0025 — Configuration protection on a corrupted file and persistence consistency](blocks/0025-config-corruption-safety-and-persistence.md)
     — **Done** (2026-09-26); `.corrupt-*` quarantine, mutation rollback,
     sentinel errors, 422 for validation.
  3. [0026 — Protecting mutating endpoints against cross-site requests](blocks/0026-csrf-origin-protection.md)
     — **Done** (2026-09-26); `requireSameOrigin` middleware
     (`Content-Type`, `Sec-Fetch-Site`, `Origin`, `MARIONETTE_ALLOWED_HOSTS`).
  4. [0027 — Controlled shutdown: step order, SSE and action queue](blocks/0027-shutdown-lifecycle-hardening.md)
     — **Done** (2026-09-26); SSE closes immediately on shutdown, history
     is saved before the queue is drained, queue per FR-18
     (503 + `Retry-After`, deduplication), `MARIONETTE_SHUTDOWN_TIMEOUT`.
  5. [0028 — Input validation and consistency of API error responses](blocks/0028-input-validation-and-api-errors.md)
     — **Done** (2026-09-26); value limits, 422 with `fields`, 405/413
     JSON envelopes, health with version, SPA cache headers.
  6. [0034 — Backend layering, structured logging and minor cleanups](blocks/0034-backend-layering-and-logging.md)
     — **Done** (2026-09-26); packages `actions`, `events`, `execengine`,
     `slog`, minimal action environment, `UpdateSettings` removed.

  **Stream B — tooling and tests (unblocks the frontend):**
  1. [0031 — Repository hygiene, lint and CI](blocks/0031-repo-tooling-and-ci-hygiene.md)
     — **Done** (2026-09-26); introduces `make verify` as the single
     validation suite.
  2. [0030 — Frontend test infrastructure (Vitest)](blocks/0030-frontend-test-infrastructure.md)
     — **Done** (2026-09-26); `npm test` in `make test` and CI, first tests
     for `api.ts`, `StatusBadge`, `cardEditModel`.

  **Stream C — frontend (after 0030):**
  1. [0029 — Action state on the dashboard and shared status tracking](blocks/0029-dashboard-action-state-and-shared-status.md)
     — **Done** (2026-09-26); `useStatusEvents`/`useCardStatus`, buttons
     are released after the action finishes, detail refetches runs/history.
  2. [0032 — Frontend API layer and conformance with the UX specification](blocks/0032-frontend-api-layer-and-ux-conformance.md)
     — **Done** (2026-09-26); `ApiError` with `fields`, ConfirmDialog
     instead of `window.confirm`, accessible `ActionEditor`,
     `ConnectionStatus`.
  3. [0033 — Shared state vocabulary, components and theme preset](blocks/0033-frontend-vocabulary-and-shared-components.md)
     — **Done** (2026-09-26); `src/ui/vocabulary.ts`, §8.4 components,
     `definePreset`, no `!important`, views < 300 lines.

  **Addendum after the phase was closed:**
  1. [0035 — Replacing PrimeFlex with our own utility layer](blocks/0035-remove-primeflex-layout-utilities.md)
     — **Done** (2026-09-26); ADR-0010, `layout.css` with 14 classes,
     CSS bundle 368 kB → 30 kB, PrimeVue stays on v4 (MIT).
  2. [0036 — Code review fixes: backend](blocks/0036-review-fixes-backend.md)
     — **Done** (2026-09-26); the same-origin check works behind a TLS
     proxy (default ports 80/443 equivalent), allowlist with the same
     normalization, store rollback does not report a false `Dirty()`.
  3. [0037 — Code review fixes: frontend](blocks/0037-review-fixes-frontend.md)
     — **Done** (2026-09-26); waiting for a check only when the backend
     schedules it, the dashboard in polling mode reads `listCards`,
     snapshots are not overwritten by older ones, `args[N]` on the field,
     timeout covers the response body.
  4. [0038 — Code review fixes 2: backend](blocks/0038-review-fixes-backend-2.md)
     — **Done** (2026-09-26); `ErrWaitDelay` = success, two-level
     validation (`ValidateEssential` on load and in the engine, full on the
     API), unique temp file + `persistMu` for saving history,
     `ErrDirectorySync` without rollback, `Store.mutate`.
  5. [0039 — Code review fixes 2: frontend](blocks/0039-review-fixes-frontend-2.md)
     — **Done** (2026-09-26); 202 carries `checkedAt` as a baseline,
     `useActionRequest` shares action progress, `useCardStatus` with
     `supersedes`, manual check budget derived from the timeout, links as
     `<Button asChild>`.
  6. [0040 — Visible status read failure](blocks/0040-status-read-failure-visibility.md)
     — **Done** (2026-09-26); both dashboard and detail show Unknown
     and "Status could not be refreshed" (UX spec §4).
  7. [0041 — End-to-end tests (Playwright)](blocks/0041-e2e-playwright.md)
     — **Done** (2026-09-26); `make e2e` and a CI job against the real
     binary, 7 scenarios including NFR-12 from the browser and 320 px;
     revealed `null` instead of `[]` for an empty run history.
  8. [0042 — Toast for page-level notices](blocks/0042-toast-page-notices.md)
     — **Done** (2026-09-26); Card saved / Card deleted as a toast, other
     feedback inline, toast in palette colors (AA).

  **Access (the "auth/access control" item), approved 2026-09-26:**
  1. [0043 — Device pairing: backend](blocks/0043-device-pairing-backend.md)
     — **Done** (2026-09-26); device token in a cookie, one-time pairing
     code, [ADR-0011](../architecture/decisions/0011-device-pairing-access.md).
  2. [0044 — Device pairing: UI and E2E](blocks/0044-device-pairing-ui.md)
     — **Done** (2026-09-26); Pair this device and Devices screens, E2E
     with authentication enabled.
  3. [0045 — Pairing instructions after installation](blocks/0045-install-pairing-hint.md)
     — **Done** (2026-09-26); `install.sh` prints where to find the first
     pairing code, only if no device is paired yet.

  **Diagnostics from validation on the Raspberry Pi (2026-09-26):**
  1. [0047 — Output of the last status check in the card detail](blocks/0047-status-check-output-in-detail.md)
     — **Done** (2026-09-26); FR-21a, the detail shows the exit code,
     duration and expandable output of the last check.
  2. [0048 — Action command as a single line in the editor](blocks/0048-single-line-command-editor.md)
     — **Done** (2026-09-26,
     [ADR-0012](../architecture/decisions/0012-single-line-command-editor.md) accepted);
     FR-22a, Command line field with a split preview, API unchanged.
  3. [0049 — No duplicate action feedback and card jumping](blocks/0049-remove-duplicate-action-feedback.md)
     — **Done** (2026-09-26); no Queued/Running/Updated next to the badge,
     remaining messages in place of `Last checked`, stable card height.

  **After Phase 7 was closed (2026-09-27):**
  1. [0050 — A primary action run appears in the detail without a page reload](blocks/0050-refresh-runs-after-primary-action.md)
     — **Done** (2026-09-27); after starting the action the detail waits for
     a new run (querying at the card's fast polling interval, default 2 s,
     budget `timeoutSec + 30 s`), removes E2E flakiness.
  2. [0051 — Named release versions (git tags)](blocks/0051-tagged-release-versions.md)
     — **Done** (2026-09-27); release only from a `vX.Y.Z` tag and a clean
     tree, version in the archive name, `.sha256`; first version `v1.0.0`.
  3. [0052 — Renaming a paired device](blocks/0052-device-rename.md)
     — **Done** (2026-09-27); FR-57, `PATCH /api/devices/{id}`, **Rename**
     button on the Devices page.
  4. [0053 — Code review fixes 3](blocks/0053-review-fixes-3.md)
     — **Done** (2026-09-27); a multi-line argument is edited in a field
     that preserves line endings; waiting for a run without a baseline does
     not mistake an old run for a new one.
  5. [0054 — Card color stripe](blocks/0054-card-color-stripe.md)
     — **Done** (2026-09-27); FR-10a, optional card color from a palette of
     8 options, shown on the dashboard as the top edge.
  6. [0055 — Card color in the detail](blocks/0055-card-color-in-detail.md)
     — **Done** (2026-09-27); FR-10a, stripe on the icon tile in the detail
     header.
  7. [0056 — Application version in the UI](blocks/0056-version-in-ui.md)
     — **Done** (2026-09-27); FR-41a, version from `/api/health` in the
     footer.
  8. [0057 — Release from CI after pushing a tag](blocks/0057-release-in-ci.md)
     — **Done** (2026-09-27); FR-06, `release.yml` workflow, GitHub Release
     with the archive and `.sha256`; verified with `v1.2.0` (failed CI, no
     Release) and `v1.2.1` (Release, upgrade and rollback on the Pi).
  9. [0058 — SSE event after a primary action run is recorded](blocks/0058-run-recorded-event.md)
     — **Done** (2026-09-27); FR-42a, ADR-0008 addendum, `run.recorded`
     instead of repeated queries for runs.
  10. [0059 — Pairing screen at 320 px with a fallback font](blocks/0059-pairing-screen-width-fix.md)
     — **Done** (2026-09-27); fix for the E2E failure in the first release
     from CI (`v1.2.0`), FR-29.

  **After the repository was published (2026-09-27):**
  1. [0060 — Toolchain update: supported Go and golangci-lint v2](blocks/0060-toolchain-go-and-golangci-lint-v2.md)
     — **Done** (2026-09-27); Go 1.27 (1.23 was out of support),
     golangci-lint v2.14.0 with `golangci-lint-action@v9` (supersedes
     Dependabot PR #3).
  2. [0061 — SSE subscription before the connected comment, reliable timing tests](blocks/0061-sse-subscribe-order-and-timing-tests.md)
     — **Done** (2026-09-27); the SSE handler subscribes before
     `: connected` (ADR-0008 addendum), the shutdown test limit follows the
     budget.

  The open questions from the review are decided in
  [requirements.md, section 13](../requirements/requirements.md#13-decisions--project-review-2026-09-26)
  (FR-18 clarified, NFR-01 c, NFR-12 accepted, the `settings` endpoint is
  not introduced, PrimeFlex was replaced after Phase 8 by our own utility
  layer per
  [ADR-0010](../architecture/decisions/0010-own-layout-utilities-replace-primeflex.md)
  (block 0035, originally ADR-0009),
  MIT license). The "auth/access control" item from the name of Phase 8 has
  been covered by device pairing since 2026-09-26 (FR-50..FR-56, ADR-0011,
  blocks 0043–0045).

- Blocks within a phase should be small enough for a single implementation
  session with an AI agent (on the order of hours of work, not days) and
  must have clear done criteria
  (see [Definition of Done](../devops/definition-of-done.md)).
- The order of Phases 3 and 4 can be swapped/merged if it turns out that the
  status action and the primary action share practically the whole
  implementation (it is the same "Action" type, just a different call).
