# Testing strategy

## Backend (Go)

- Unit tests (`go test -race ./...`, run by `make test`) for all domain logic
  (action evaluation, config store, status engine) — without depending on
  actually spawning processes or on the file system where possible
  (interface + fake implementation).
- The execution engine is tested through an abstraction over `os/exec` (an
  interface with a fake/mock implementation for tests); real process
  execution is verified only in a small number of integration tests (e.g.
  `echo`, timeout check).
- HTTP handlers (`internal/server`) are tested via `net/http/httptest`.
- Coverage goal: domain logic and handlers ~80 %+; 100 % coverage of trivial
  code (getters, `main.go`) is not a goal.

## Deployment (install script)

- `deploy/install_test.sh` (`make deploy-test`, part of `make test` and of
  the CI job `backend`) installs the package into a temporary `DESTDIR`
  without root and without systemd and verifies that the paths from the unit
  (`ExecStart`, `EnvironmentFile`, `WorkingDirectory` = `ReadWritePaths`,
  `MARIONETTE_CONFIG`) exist in the installation, the permissions match the
  documentation (binary 755, data 750, configuration 640), a repeated run
  preserves the configuration and the defaults, and a file that is not ELF
  is rejected. Behavior with real systemd (start, restart, reboot,
  hardening) is verified manually on the reference host (block 0023).

## Frontend (Vue)

- `npm run lint` (`--max-warnings 0`), `vue-tsc` (type-check) and
  `prettier --check` are a mandatory part of CI and `make verify`.
- Unit and component tests run in **Vitest** (`npm test`, part of
  `make test` and of the CI job `web`) in the `happy-dom` environment;
  components are mounted via Vue Test Utils with the PrimeVue plugin. The
  `src/**/*.spec.ts` files sit next to the code under test, share the Vite
  configuration (`vitest.config.ts`, alias `@`) and do not end up in the
  bundle. Tested are the pure logic (the API layer `api.ts` via a `fetch`
  mock, the form model `cardEditModel.ts`), composables (`useStatusEvents`
  via an injected connector, `useCardStatus` with fake timers) and the
  behavior of views and custom components (`HomeView`, `CardDetailView`,
  `CardEditView` with a memory router — `CardEditView` via `RouterView` so
  that `onBeforeRouteLeave` applies; `ActionEditor` including accessible
  names, `ConnectionStatus`, `StatusBadge`, `ActionCard` for each state,
  `StatusTimeline` with durations) and the vocabulary/formatting
  (`ui/vocabulary`, `ui/format`) — the appearance of the PrimeVue components
  themselves is not tested. REST
  functions are mocked via `vi.mock("@/api")` with a partial override, SSE
  via the shared `src/test/fakeEventSource.ts`
  (`vi.stubGlobal("EventSource", …)`), the confirmation dialog via
  `src/test/fakeConfirm.ts` (provide instead of `ConfirmationService`, the
  test calls `accept`/`reject`).
  Coverage: `npm run test:coverage` (v8); the goal for `api.ts` and pure
  modules is ~80 %+, the remaining views are covered in blocks 0032 and 0033.

## End-to-end (Playwright)

- `make e2e` (block 0041) builds the binary with the embedded SPA, starts it
  with an empty configuration in a temporary directory on `127.0.0.1:18080`
  (`web/e2e/start-server.sh`) and runs `web/e2e/*.e2e.ts` in headless
  Chromium (`web/playwright.config.ts`, one worker — the specs share the
  server and each starts by deleting all cards via the API).
- It covers what unit tests cannot see: a real browser, embed, SSE
  (the **Live** indicator, check result), 202 → waiting → result, 422 all
  the way to the form field, confirmation dialogs, NFR-12 with the headers
  the browser sends (a page on a different origin does not run an action),
  and the absence of horizontal scrolling at 320 px on the overview, detail
  and forms (UX spec §9, §10 scenarios 1, 3, 5, 6, 7 and 2 partially).
- The server runs with authentication enabled (block 0044): the `setup`
  project (`e2e/pair.setup.ts`) pairs the browser using the code from the
  server log (`E2E_SERVER_LOG`) through the pairing screen and saves the
  cookie to `e2e/.auth/device.json`; the other specs run as a paired device.
  Unpaired browsers are created with an empty `storageState` —
  `browser.newContext()` otherwise inherits the project's saved state.
- Locators find elements by roles and accessible names (like a user),
  CSS classes only where a role is missing (badge, `.field-error`).
- It is not part of `make verify` (requires a browser, ~15 s); it runs as a
  separate CI job `e2e` and is run manually for blocks that change the UI
  flow. The browser is installed with
  `npx playwright install --with-deps chromium` (in the devcontainer by
  `post-create.sh`).

## Manual verification

- Before a release to an ARM host: run `make build-arm64`, deploy to a test
  Linux with systemd; perform reference verification on a Raspberry Pi ARM64
  with Ubuntu 24.x, including `systemd` start/stop/restart and the basic
  scenario (WOL + ping) end-to-end.

## What is not tested (deliberately)

- UI appearance/visual regression (no visual regression tool in the MVP).
- Load tests — outside the expected scope of use (a handful to dozens of
  cards, few concurrent users).
