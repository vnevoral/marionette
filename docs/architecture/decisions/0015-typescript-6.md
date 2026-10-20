# ADR-0015: TypeScript 6 now, TypeScript 7 once the tooling supports it

- **Status**: Accepted
- **Date**: 2026-10-20

## Context

The web UI was on TypeScript 5.9. Two newer major versions exist:
TypeScript 6.0 (6.0.3, April 2026), the last release of the JavaScript
compiler, and TypeScript 7.0 (7.0.2, July 2026), the native compiler. Both
are older than the two-week rule (`docs/devops/ci-cd.md`), and
`ci-cd.md` asks for a compatibility check and an ADR before a new major
of TypeScript.

The type checking runs through `vue-tsc` and the linting through
`typescript-eslint`; both use the TypeScript compiler API:

- `typescript-eslint` 8.72 declares `typescript >=4.8.4 <6.1.0`;
- `vue-tsc` 3.3 is built on the JavaScript API of the compiler, which
  TypeScript 7.0 does not offer.

TypeScript 6.0 changes several defaults: `types` is empty unless set,
`noUncheckedSideEffectImports` is on, and `baseUrl` is deprecated (an
error unless silenced).

## Decision

The web UI moves to **TypeScript 6.0** and adapts its configuration to
the new defaults instead of silencing them:

- `tsconfig.app.json` drops `baseUrl` (`paths` resolve relative to the
  config file) and sets `types: ["vite/client"]`, which also declares the
  CSS side-effect imports;
- an exported test helper gets an explicit return type, which composite
  declaration output now requires.

TypeScript 7 is adopted only when `typescript-eslint` and `vue-tsc`
support it; Dependabot keeps ignoring TypeScript majors.

## Considered alternatives

- **TypeScript 7 now** — rejected: `typescript-eslint` refuses it and
  `vue-tsc` cannot run on it, so `make lint` and the build would break.
- **Stay on 5.9** — rejected: 6.0 works with the current tooling, its
  deprecations are the ones TypeScript 7 removes, and fixing them now
  makes the later move to 7 smaller.
- **`ignoreDeprecations: "6.0"`** — rejected: it only postpones the
  `baseUrl` change to the TypeScript 7 upgrade.

## Consequences

- No change in the built bundle or the behaviour; `make verify` and
  `make e2e` pass.
- The configuration has no deprecated options left, so the TypeScript 7
  upgrade is a tooling question only.
