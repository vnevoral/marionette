# Implementation block: Renaming a paired device

- **Phase**: 8 — Hardening (addendum after the phase was closed), access
  area
- **Requirements**: FR-57, FR-52, FR-55, NFR-12, NFR-13
- **ADRs**: ADR-0011. No new ADR is needed: the new endpoint extends
  device management within the same security model (the same permissions
  as removal; neither the token nor the expiry changes)
- **Status**: Done
- **Dependencies**: Blocks 0043, 0044 (device registry, Devices page)

## Goal

Today a device name is entered only during pairing (the pairing screen
offers a name derived from the browser) and cannot be changed later; anyone
who accepted the default suggestion may have, say, two "Chrome on Linux"
entries in the list. After this block, any paired device can be renamed on
the **Devices** page.

## Scope

- **In scope**:
  - `internal/access`: `Registry.Rename(id, name string) (Device, error)`
    — the same validation as `Pair` (`strings.TrimSpace`, non-empty, at
    most `MaxDeviceNameLength` characters → `ErrInvalidName`), unknown or
    expired device → `ErrDeviceNotFound`, immediate save of the device
    file (`saveLocked`) with the original name restored on a write error,
    just like `Remove`; the token, `PairedAt` and `LastSeenAt` do not
    change;
  - `internal/server`: `PATCH /api/devices/{id}` with body `{"name": "…"}`
    → `200 {device}` (same shape as in `GET /api/devices`, including
    `current`); invalid name `422` with `fields.name`; unknown device
    `404`; protected by pairing (FR-50) and cross-site protection
    (`requireSameOrigin`, NFR-12) like the other mutating routes;
  - `web/src/api.ts`: `renameDevice(id, name)`;
  - `DevicesView`: a **Rename** button for each device (pencil icon with
    the accessible name "Rename <name>") that replaces the name with a
    field with **Save** / **Cancel** buttons; Enter saves, Escape cancels;
    a `422` error is shown at the field; after saving, an inline
    confirmation "\"<new name>\" saved" (UX spec §4: inline, not a toast);
  - vocabulary (`ACCESS.rename`, …), UX specification §7.4, the endpoint
    table in `docs/architecture/overview.md`, README (section on devices);
  - tests: Go (`Registry.Rename` — trimming, empty and too long name,
    unknown device, persistence after reload, unchanged token and times;
    handler — 200, 422, 404, 401 without a token, 403 from a foreign
    origin), Vitest `DevicesView` (rename, cancel, 422 error, keyboard),
    E2E (renaming one's own device and its display after a page reload).
- **Out of scope**:
  - uniqueness of names (FR-57: they need not be unique);
  - restricting renaming to one's own device only (FR-57: the same
    permissions as removal);
  - history or audit of name changes (the change is written to the service
    log at info level, just like pairing and removal).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner ("agree with the proposals").

## Proposed solution

See scope. `PATCH` instead of `PUT`, because only one field of the resource
changes. Log entry: `renamed paired device` with `device` (id) and the new
`name`; the old name is not needlessly written to the log twice. The UI on
the Devices page uses PrimeVue `InputText` and `Button` (project
convention); the field has the label "Device name" and the same length
limit as the pairing screen.

## Test plan

- `go test -race ./internal/access ./internal/server`.
- Vitest `DevicesView.spec.ts`.
- `make e2e` (extension of `pairing.e2e.ts`).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the new name is visible on all paired devices after the list is
  refreshed and survives a service restart.

## Closure

- **Status after implementation**: Done (2026-09-27). `make verify` and
  `make e2e` (17 scenarios including the new rename) passed in the final
  verification together with blocks 0050 and 0051.
- **Verification**: `go test -race ./internal/...`, `go vet ./...`,
  `gofumpt -l`, `golangci-lint run ./...` — no errors; in `web/`
  `npx vitest run` (174 tests), `npx vue-tsc -b`,
  `npx eslint . --max-warnings 0`, `npx prettier --check .` — no errors.
  New tests: `internal/access` (`TestRenameKeepsTokenAndTimesAndPersists`,
  `TestRenameRejectsInvalidNamesAndUnknownDevices` including an expired
  device and the 64-character boundary,
  `TestRenameKeepsTheOldNameWhenSavingFails`), `internal/server`
  (`TestPairedDeviceRenamesDevices`: 200 with `current`, 422 with
  `fields.name`, 404, 400 for an unknown field, 401 without a cookie, 403
  from a foreign origin; `PATCH /api/devices/{id}` added to the list of
  protected routes), Vitest `api.spec.ts` and `DevicesView.spec.ts`
  (rename, session sync for one's own device, cancel by button and by
  Escape, Enter, 422 error and empty name at the field), E2E
  `pairing.e2e.ts` (a separate browser renames itself, the new name is
  visible after a reload and on the other devices, then it is removed).
- **Deviations from the plan**: none of substance. Clarification: the
  `PATCH` response has the shape `{"device": {…}}` like
  `POST /api/pairing`; the device row with renaming was extracted into the
  component `web/src/components/DeviceRow.vue` so that `DevicesView.vue`
  stays under 300 lines; an empty name is already rejected by the client
  (the same message as on the pairing screen) without a request to the
  server; after renaming one's own device, the session state in the SPA is
  updated too. The existing E2E test for removing one's own device looks
  up the button by name, because the row now has two buttons.
- **Documentation updated**: UX specification §7.4, the endpoint table in
  `docs/architecture/overview.md`, README (Pairing devices), the `ACCESS`
  vocabulary in `web/src/ui/vocabulary.ts`.
