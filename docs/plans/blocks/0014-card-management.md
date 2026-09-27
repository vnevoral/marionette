# Implementation block: Action card management

- **Phase**: 6 — Dashboard UI
- **Requirements**: FR-10, FR-11, FR-12, FR-14, FR-15, FR-23, FR-40
- **ADRs**: ADR-0003, ADR-0004, ADR-0006
- **Status**: Done
- **Dependencies**: Blocks 0010–0013, the existing CRUD API in `internal/server`

## Goal

The administrator will be able to create, edit and delete an action card in
the UI, including the primary action, the optional status action and the
polling settings. The form will send structured JSON via the existing CRUD
API; server-side validation remains authoritative.

## Scope

- **In scope**:
  - a separate route `/manage` reachable from the dashboard;
  - a list of existing cards with edit and delete actions;
  - a form for a new card and for editing an existing card;
  - card identity fields: ID on creation, name, description and icon;
  - an editor for the primary and the optional status action: command,
    arguments, working directory, environment variables and timeout;
  - choice of result rule: exit code, match and not match including a
    regex pattern;
  - polling interval, fast-polling interval and fast-polling window;
  - client-side validation of required fields and numbers before
    submitting;
  - handling of statuses `201`, `200`, `204`, `400`, `404`, `409` and
    `500`;
  - a confirmation dialog before deletion and a return to the dashboard
    after a successful change;
  - PrimeVue form components, loading, empty and error states.
- **Out of scope**:
  - new backend endpoints or changes to domain validation;
  - authentication and authorization;
  - editing run history and status transition history;
  - card import/export, bulk operations and drag-and-drop ordering;
  - test-running a command directly from the form.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: Approved by the owner 2026-09-25; implementation in progress.

## Proposed solution

Extend `web/src/api.ts` with typed functions `createCard`, `updateCard` and
`deleteCard`. Add a management view and reusable components for the action
editor and dynamic rows of arguments/environment variables. The form will
work with a local copy of `ActionCard`, keep the ID from the URL/state when
editing, and when the status action is turned off send `status: undefined`
so that it matches the `omitempty` contract. After a successful write the
dashboard is reloaded; on error the form stays open and shows the server
message.

## Test plan

- The TypeScript build verifies the types of requests, response data and
  Vue templates.
- ESLint and Prettier verify frontend conventions.
- Manually verify creation, editing, turning the status action on/off and
  deletion.
- Manually verify validation of an empty command, timeout, regex and an
  invalid polling relation.
- Via a running backend, verify a conflicting ID, a server `400`,
  successful `201/200/204` and the dashboard refresh after a change.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
all mutations use the existing `/api/cards` endpoints, the form does not
lose unsaved changes on error, and both `npm run lint` and `npm run build`
pass.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint`, `npm run build` — successful; CRUD smoke test `201 → 200 → 204 → 404` — successful
- **Documentation updated**: yes; the roadmap links to this block
