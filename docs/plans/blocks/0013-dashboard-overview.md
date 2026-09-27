# Implementation block: Card dashboard and action controls

- **Phase**: 6 — Dashboard UI
- **Requirements**: FR-20, FR-21, FR-23, FR-40
- **ADRs**: ADR-0003, ADR-0004, ADR-0006
- **Status**: Done
- **Dependencies**: Blocks 0010–0012, REST API in `internal/server`

## Goal

The dashboard shows all action cards in a clear grid, loads their last
known status and allows asynchronously enqueueing the primary or a manual
status action. Card configuration, history and the detailed CRUD editor
remain for follow-up blocks.

## Scope

- **In scope**:
  - a typed client for the card list, status and enqueue endpoints;
  - loading cards when the dashboard opens, and manual refresh;
  - a card with the status `unknown`, `ok`, `fail` or `running`;
  - a primary action button, a manual status action button and disabling
    them during the request;
  - error and empty states;
  - a responsive layout built on PrimeVue.
- **Out of scope**:
  - the card create/update/delete form;
  - run details and status transition history;
  - authentication, websockets and server-side job IDs.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: Async endpoints are shown as accepted; the UI does not wait for the process to complete.

## Proposed solution

Extend `web/src` with a small API module with types matching the existing
backend JSON types, and a dashboard view. The `running` state will be a
local UI projection for the duration of the enqueue request; after
acceptance the card is refreshed from the read-only status API. PrimeVue
Aura remains the only visual foundation.

## Test plan

- The TypeScript build verifies the types of API data and Vue templates.
- ESLint verifies Vue/TypeScript conventions.
- Manually verify an empty list, an API error, successful loading and both
  enqueue actions.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
the dashboard uses real `/api` endpoints, not a health-check mock, and both
`npm run lint` and `npm run build` pass.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint`, `npm run build` — successful; lint reports 29 non-blocking warnings
- **Documentation updated**: yes; the roadmap links to this block
