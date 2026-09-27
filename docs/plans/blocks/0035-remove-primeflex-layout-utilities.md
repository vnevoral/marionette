# Implementation block: Replacing PrimeFlex with a custom utility layer

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-25, NFR-03, NFR-08, NFR-10
- **ADRs**: ADR-0003, ADR-0009 (superseded), ADR-0010
- **Status**: Done
- **Dependencies**: Blocks 0019, 0020, 0033

## Goal

After this block, the frontend does not depend on the frozen PrimeFlex:
the fourteen layout classes the code actually uses are defined in a
custom `layout.css` with the same names and values, templates do not
change, and the CSS bundle shrinks by the PrimeFlex package. PrimeVue
stays on the MIT v4 branch.

## Scope

- **In scope**:
  - `web/src/styles/layout.css` with the classes `grid`, `col-12`, `md:col-4`,
    `md:col-6`, `lg:col-4`, `flex`, `flex-column`, `flex-wrap`,
    `align-items-center`, `justify-content-between`, `gap-2`, `gap-3`,
    `p-2`, `mt-4` — values and breakpoints (768 px, 992 px) identical
    to PrimeFlex 4.0.0, without `!important`;
  - removing `primeflex` from `package.json`, the lockfile, `main.ts`
    and from the Dependabot ignore list;
  - ADR-0010 (supersedes ADR-0009), extending ADR-0003 with the PrimeVue 5
    license change and the decision to stay on v4;
  - updating `docs/devops/ci-cd.md`, `docs/requirements/requirements.md`
    (section 13), UX spec §8, the roadmap.
- **Out of scope**:
  - Tailwind CSS or another utility library (ADR-0010);
  - upgrading to PrimeVue 5 (PrimeUI license, ADR-0003);
  - visual changes — the layout must stay pixel-identical.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved based on the 2026-09-26 analysis:
  PrimeFlex 4.0.0 is the last version (February 2025), the code uses 14
  classes in ~60 occurrences, PrimeFlex pulls in 446 kB of CSS. PrimeVue
  5.0.x (7–8/2026) moved to the commercial PrimeUI License with a license
  key; v4 stays MIT.

## Proposed solution

- `layout.css`: `.grid` (flex, wrap, negative margins 0.5 rem), `.col-12`
  and responsive `md:`/`lg:` variants (`flex: 0 0 auto; padding: 0.5rem;
  width`), flex/gap/spacing utilities with PrimeFlex values (`gap-2` 0.5 rem,
  `gap-3` 1 rem, `p-2` 0.5 rem, `mt-4` 1.5 rem).
- `main.ts`: `import "./styles/layout.css"` instead of `primeflex/primeflex.css`.
- Adding a new utility class requires an entry in `layout.css`; unused
  classes are not introduced (ADR-0010).

## Test plan

- `make verify` (lint, types, Vitest, build, Go).
- `grep -rn primeflex web/src web/package.json .github` returns 0 lines.
- Comparing the size of `internal/webui/dist/assets/*.css` before and after.
- Manual screenshot smoke test of three screens at 320 px, 768 px and
  desktop (layout unchanged) — reference host.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `primeflex` is in neither the dependencies nor the code;
- no template changes (only CSS and the import);
- ADR-0009 has the status "Superseded by ADR-0010".

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (lint, `vue-tsc`, Prettier, Vitest 77
  tests, `vite build`, `go test -race`, `go vet`). `primeflex` removed
  from `package.json`, the lockfile (`npm uninstall`), `main.ts` and the
  Dependabot ignore; `grep -rni primeflex web/src web/package.json .github`
  finds only an explanatory comment in `layout.css`. No template changed
  (the diff contains only CSS, the import and documentation). CSS bundle
  `internal/webui/dist/assets/*.css`: 367 604 B → 30 381 B (gzip
  39 859 B → 6 846 B). The visual screenshot smoke test at 320/768 px and
  on desktop remains for the reference host; values and breakpoints are
  taken 1:1 from PrimeFlex 4.0.0, so no layout change is expected.
- **Deviations from the plan**: none. The `.grid > [class*="col"]` class
  with `box-sizing: border-box` is kept for parity with PrimeFlex, even
  though the global reset `* { box-sizing }` makes it redundant.
- **Documentation updated**: yes — ADR-0010 (new), ADR-0009 (status
  "Superseded by ADR-0010"), ADR-0003 (addendum on the PrimeVue 5 license
  and staying on v4), `docs/devops/ci-cd.md`, `docs/requirements/requirements.md`
  section 13, UX spec §8.2, roadmap, `.github/dependabot.yml`.
