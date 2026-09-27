# Implementation block: Device pairing — UI and E2E

- **Phase**: 8 — Hardening ("auth/access control" item)
- **Requirements**: FR-50..FR-55, FR-26, NFR-08, NFR-09
- **ADRs**: ADR-0011, ADR-0007
- **Status**: Done
- **Dependencies**: Block 0043

## Goal

When done, an unpaired browser sees the **Pair this device** screen,
enters the code and a device name, and never authenticates again. A
paired device has a **Devices** page with a list, removal and generating
a code for another device. The E2E suite runs with authentication on.

## Scope

- **In scope**:
  - `api.ts`: `getSession`, `pairDevice`, `listDevices`, `removeDevice`,
    `createPairingCode`; `ApiError.isUnauthorized`;
  - router: session check at application start and redirect to
    `/pair?next=…` after any `401`; after pairing, return to `next`;
  - `PairView` (`/pair`): code (8 characters, uppercase, hyphen optional,
    prefilled from `?code=`), device name (prefilled based on the browser
    and system), for the first device a hint
    `journalctl -u marionette` with a command to copy, inline errors;
  - `DevicesView` (`/devices`): list (name, paired, last seen,
    "This device"), removal via a confirmation dialog, **Pair a new
    device** → code, validity countdown, link to copy;
  - `AppShell`: **Devices** navigation item; on `/pair` the shell without
    navigation and without the Live indicator;
  - `useStatusEvents`: stops polling on `401` from the REST fallback;
  - E2E: a `setup` project pairs the browser with the bootstrap code from
    the server log and saves `storageState`; specs: pairing, wrong code,
    device removal → back to pairing, a second device with a code from
    Devices, API without a token → 401; existing specs run paired;
  - UX spec (IA §2, new screens, glossary), testing strategy, README.
- **Out of scope**: QR code (a new dependency), renaming devices,
  backend (block 0043).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: see block 0043.

## Proposed solution

- The screens use existing components (`PageHeader`, `DetailPanel`,
  `RequestState`, `ConfirmDialog`, toast via `useNotify` for "Device
  paired" and "Device removed" — both change the page, UX spec §4).
- The code is shown in a large monospace font in groups of 4+4
  (`ABCD-EFGH`), link `/<base>/pair?code=ABCDEFGH`.

## Test plan

- Vitest: `PairView` (code normalization, prefill from the query, errors
  from 400, bootstrap hint), `DevicesView` (list, removal with
  confirmation, removing the own device → `/pair`, code generation and
  countdown), router guard (401 → `/pair?next`), new `api.ts` functions.
- E2E see scope; `make e2e`, `make verify`.
- Contrast and 320 px for both new screens (E2E).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (Vitest 97 → 119: `api.spec.ts`
  session paired/unpaired/bootstrap/open, pairing, code, list, removal,
  401 handler only outside session and pairing; `PairView.spec.ts` hint
  with the log only with no devices, code format and prefill, name
  suggestion, return to `next`, no return outside the application, one
  message for a wrong code, name errors; `DevicesView.spec.ts` list with
  "This device" and validity, inline removal of another device, removing
  the own device → pairing + toast, code with countdown and link,
  detection of a newly paired device, a new code after expiry,
  authentication turned off; `deviceName.spec.ts`, `params.spec.ts`).
  `make e2e` with authentication on: 14 tests (setup pairs the browser
  with the code from the server log; an unpaired browser sees only
  pairing and the API returns 401; wrong code; a second browser with a
  code from Devices, detection without reload, removal and the removed
  one back to pairing; pairing link, removing the own device = sign-out;
  320 px for Devices, pairing and the shown code; all earlier scenarios
  as a paired device); 3 repeated runs 14/14.
- **Deviations from the plan**: (1) Access to `/pair` is handled by the
  router guard via `meta.pairing` instead of a separate layout;
  `AppShell` on this route hides both the navigation and the Live
  indicator, so no SSE stream is opened without a token. (2) The code is
  shown in an `<output>` element (the screen reader announces it once),
  not in `<p aria-label>`, which screen readers would ignore.
  (3) `useStatusEvents` did not change: after a 401 the router redirects
  to pairing, the view unmounts and the stream and polling end with the
  last subscriber. (4) `tsconfig.node.json` got `target` ES2022 and a
  `lib` with DOM, because it now also checks the E2E files. (5) A signed-in
  browser on `/pair` (e.g. an opened pairing link) is redirected to
  `next` or the overview.
- **Documentation updated**: yes — UX spec §2.1, §2.2, new §7.4, testing
  strategy (E2E with a setup project), README (development with
  authentication on), roadmap.
