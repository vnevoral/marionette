# Implementation block: UX-03 — Management CRUD workflow

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-10, FR-22, FR-24, FR-26, FR-27, FR-28, FR-29, NFR-08..10
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0015–0016, existing CRUD API

## Goal

Manage cards will be a real CRUD screen. For each existing card the user
will see explicit `Edit` and `Delete`, will create a new card via a clear
CTA and after deleting will stay in the management context without an
unexpected redirect.

## Scope

- **In scope**:
  - explicit actions on each list item: `Edit`, `Delete`;
  - selected item state and a clear marking of the open editor;
  - `New card` in the header/list and an empty state without duplicate
    actions;
  - delete confirmation with the card's name;
  - refreshing the list after create/update/delete;
  - preserving the form on a server error;
  - green-first tokens, keyboard focus and mobile layout.
- **Out of scope**:
  - a new backend/API contract;
  - card detail, runs and status timeline;
  - an unsaved-changes guard when navigating away from the page;
  - search and bulk operations.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: CRUD actions must be visible directly in the card list.

## Proposed solution

Extend `ManageView.vue` with a card row with the name, ID, status check
configuration and explicit PrimeVue Button actions. Delete will work with the
item's ID and name, not only with the currently open form. After a
successful delete the list is reloaded and the editor is reset to a new card
state. The form API and `ActionEditor` will remain unchanged.

## Test plan

- `npm run format`, `npm run lint`, `npm run build`.
- Browser smoke test: empty list, New card, Edit and Delete are visible.
- API smoke test create/update/delete verifies that the UI uses the existing
  endpoints.
- Keyboard smoke test goes through the list actions and the editor.

## Done criteria

On `/manage` the CRUD actions are obvious without knowing that one should
click the card name; deleting requires confirmation, an error preserves the
form data and after a successful operation the list stays consistent.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint`, `npm run build` — successful; the browser smoke test confirmed Edit/Delete on existing cards; CRUD API smoke test `201/200/204/404` — successful
- **Documentation updated**: yes; roadmap and UX specification
