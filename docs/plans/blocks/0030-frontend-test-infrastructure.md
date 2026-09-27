# Implementation block: Frontend test infrastructure (Vitest)

- **Phase**: 8 — Hardening
- **Requirements**: NFR-05, NFR-06, FR-27
- **ADRs**: ADR-0003
- **Status**: Done
- **Dependencies**: `docs/devops/testing-strategy.md` (Vitest promised in phase 6), block 0013

## Goal

Once complete, `web/` has runnable unit tests (`npm test`), CI requires
them and there is a first set of tests for the API layer and the pure view
logic, so that blocks 0029 and 0032 can write regression tests.

## Scope

- **In scope**:
  - devDependencies `vitest`, `@vue/test-utils`, `happy-dom` (or
    `jsdom`), `@vitest/coverage-v8`;
  - `web/vitest.config.ts` sharing the `@` alias with Vite; `environment:
    happy-dom`; `include: src/**/*.spec.ts`;
  - scripts `test` (`vitest run`), `test:watch`, `test:coverage`;
  - `tsconfig.app.json` includes `src/**/*.spec.ts` and `vitest/globals`
    types;
  - first tests: `api.ts` (`request`: 204, 2xx JSON, 4xx `{error}`, 5xx
    without JSON, network error, loss of the `Accept` header on spread),
    `StatusBadge` (renders label + tone), `CardEditView` pure functions
    `validate`/`actionFrom` (exported to `src/views/cardEditModel.ts`);
  - the CI job `web` adds `npm test`; the Makefile `test` also runs
    `npm test` (or a new target `ui-test` called from `test`);
  - `testing-strategy.md` updates the Frontend section to the actual state.
- **Out of scope**:
  - E2E/browser tests (Playwright) — to be considered in a separate block
    after phase 8;
  - visual regression;
  - refactoring views for testability beyond extracting pure functions.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a bug fix. Finding of the 2026-09-26 review (H3). The testing
  strategy promises Vitest from phase 6; phases 6 and 9 are done without tests.

## Proposed solution

- `web/vitest.config.ts` via `mergeConfig(viteConfig, defineConfig({ test:
  { environment: "happy-dom", globals: true, coverage: { provider: "v8",
  include: ["src/**"] } } }))`.
- `web/src/api.ts`: `request` testable via a `globalThis.fetch` mock
  (`vi.stubGlobal`).
- `web/src/views/cardEditModel.ts`: pure functions currently inside
  `CardEditView.vue` (`validate`, `actionFrom`, `fingerprint`).
- CI: step `npm test -- --run` between lint and build; cache unchanged.

## Test plan

- The tests from the scope above pass locally and in CI.
- Verify that `npm run build` does not include `*.spec.ts` in the bundle
  (Vite includes only from `main.ts`; the tsconfig include of spec files
  must not break `vue-tsc -b`).
- `npm run lint` with `.spec.ts` files (globals `describe/it/expect`
  in the ESLint config).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `npm test` runs in CI and in `make test`;
- coverage of `api.ts` ≥ 80 %;
- `testing-strategy.md` matches reality.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (`make test` now calls `ui-test` =
  `npm test` and then `go test -race`). Vitest 5 + Vue Test Utils 2.5 +
  happy-dom 20 + coverage-v8; 24 tests in three files: `src/api.spec.ts`
  (2xx JSON, 204, 4xx envelope, 5xx without JSON, missing `error`, network
  error, preserving `Accept` with custom headers, `Content-Type` on mutating
  calls without a body, removing `currentStatus` from the payload, URL with
  an encoded ID, `connectStatusEvents` with a fake `EventSource` including
  invalid events), `src/components/StatusBadge.spec.ts` (label, icon, tone,
  default tone), `src/views/cardEditModel.spec.ts` (`actionFrom`,
  `validate`, `fingerprint`, `copyAction`, `environmentRows`). Coverage of
  `api.ts` ≥ 80 % (see `npm run test:coverage`), `cardEditModel.ts` 100 %.
  `npm run build` does not include spec files in the bundle (Vite starts
  from `main.ts`), `vue-tsc -b` type-checks them; `npm run lint` and
  `format:check` pass. **Finding:** the test revealed a real bug in
  `request` — `...options` after the merged headers dropped `Accept` on
  every call with a custom `Content-Type`; fixed.
- **Deviations from the plan**: (1) the tests import `describe/it/expect/vi`
  explicitly from `vitest` instead of `globals: true` + `vitest/globals`
  types — no global symbols in ESLint or tsconfig; (2) `EnvironmentRow`,
  `emptyAction`, `emptyCard`, `copyAction` and the new `environmentRows` are
  exported together with `validate`/`actionFrom`/`fingerprint`;
  `validate` returns `{ fieldErrors, firstError }` instead of mutating a
  reactive object; (3) installation required `npx npm@11` because of an
  `edgesOut` bug in npm 10.8 when resolving peer dependencies (lockfile v3
  remains compatible with `npm ci`); (4) added `web/.prettierignore` and
  `web/coverage/` in `.gitignore`, so that the coverage output does not break
  `format:check`.
- **Documentation updated**: yes — `docs/devops/testing-strategy.md`
  (the Frontend section matches reality), AGENTS.md (`make test`), CI
  (`npm test` between `format:check` and `build`), Makefile (`ui-test`),
  roadmap.
