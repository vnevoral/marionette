# Implementation block: End-to-end tests (Playwright) against the real binary

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-20..29, FR-40..42, NFR-08, NFR-10, NFR-12
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0026, 0030, 0039, 0040

## Goal

When done, there is an automated suite that starts the real binary with the
embedded SPA and drives it from a real browser: creating, running, checking
and deleting a card, validation errors, dialogs, cross-site protection and
the layout at 320 px. Unit tests do not cover these layers (embed, SSE,
browser headers, real CSS).

## Scope

- **In scope**:
  - `@playwright/test` (devDependency of `web/`), `web/playwright.config.ts`,
    `web/e2e/start-server.sh` (binary, empty configuration in a temporary
    directory, `127.0.0.1:18080`), `web/e2e/helpers.ts`, the specs
    `card-lifecycle.e2e.ts` and `security-and-layout.e2e.ts`;
  - `make e2e`, `npm run test:e2e`, CI job `e2e` with a report artifact,
    Chromium in the devcontainer's `post-create.sh`;
  - ESLint, `vue-tsc -b` (`tsconfig.node.json`) and Prettier cover `e2e/`
    and the configuration; outputs in `.gitignore`;
  - documentation: testing strategy, CI/CD, DoD, README, roadmap.
- **Out of scope**: visual regression snapshots, other browsers (Firefox,
  WebKit), tests on the ARM64 host, inclusion in `make verify`.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: The "optional E2E block" item from the review; the
  instruction "let's move on to the next steps" with verification on the Pi
  deferred. Chromium only, outside `make verify` because of the browser
  dependency; the DoD requires `make e2e` for blocks that change a UI flow
  or the HTTP contract.

## Proposed solution

- One worker; specs prepare their state through the API (`createCard`,
  `deleteAllCards` as a non-browser client without `Origin`).
- Locators by roles and accessible names; `actionEditor(page, title)` and
  `currentStatusBadge(page)` for sections without their own name.
- Cross-site: `request` with `Origin`/`Sec-Fetch-Site`/`text/plain` and a
  page on `localhost` (a different origin than `127.0.0.1`) that sends a
  `no-cors` POST; the test verifies that no run was created.
- 320 px: `scrollWidth - clientWidth ≤ 0` on `/`, the detail, the edit
  form and the new-card form.

## Test plan

- `make e2e`: 7 tests; `--repeat-each=3` without flaky results.
- `make verify` (lint and types cover `e2e/`).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the CI job `e2e` is defined and `make e2e` passes locally.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make e2e` — 7 passed (~13 s); `npx playwright test
--repeat-each=3` — 21 passed; `make verify` passed. Covered UX spec §10
  scenarios: 1 (empty overview → creation), 2 partially (320 px without
  horizontal scrolling; tablet/desktop and appearance remain a manual
  check), 3 (Queued/Running → result, no false success), 5 (422 keeps the
  form values), 6 (unsaved changes), 7 (deletion with confirmation). NFR-12
  verified with real browser headers.
- **Finding during implementation**: `GET /api/cards/{id}/runs` returned
  `null` instead of `[]` for a card without runs (`Store.GetRuns` copied
  via `append` into a nil slice). The frontend masked it with `?? []`;
  another client would have crashed. Fixed in `store.go`, added
  `TestRouterServesEmptyHistoriesAsArrays` (runs and status history →
  `[]`).
- **Deviations from the plan**: none.
- **Environment note**: the running devcontainer has Node 20, while
  `devcontainer.json`, `.nvmrc` and `engines` prescribe 22 — the container
  was built before the change and needs "Rebuild Container"; Playwright and
  the other tools work on Node 20.
- **Documentation updated**: yes — `docs/devops/testing-strategy.md`
  (E2E section), `docs/devops/ci-cd.md` (job `e2e`), DoD (`make e2e` for UI
  and HTTP changes), README, roadmap.
