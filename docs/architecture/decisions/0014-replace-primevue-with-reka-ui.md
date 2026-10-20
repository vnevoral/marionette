# ADR-0014: Replace PrimeVue with Reka UI and own styled components

- **Status**: Accepted
- **Date**: 2026-10-20 (accepted by the owner the same day)

## Context

[ADR-0003](0003-vue-primevue-frontend.md) chose PrimeVue v4 with the Aura
theme and `primeicons`, all MIT at the time. Since then PrimeTek has moved
its packages to the PrimeUI License one by one:

- PrimeVue 5 (2026), `@primeuix/themes` 3.x and `primeicons` 8.x
  (2026-07-15) are under the PrimeUI License. Its free Community License is
  per developer, needs a license key that must not be published, and
  directs open-source projects to the MIT versions.
- The project had used `primeicons` `^8` since its first commit without
  noticing; block 0072 moved it back to the MIT 7.0.0 release.
- Every PrimeTek package in the bundle is MIT again (PrimeVue 4.5.5,
  `@primeuix/*` 2.x and below, `primeicons` 7.0.0), but each of them is a
  line that the vendor no longer develops in the open. New features and,
  sooner or later, security fixes land only in the licensed lines; every
  Dependabot major update has to be checked for a licence change.

A Marionette contributor would need their own Community License to work on
the licensed lines, and builds without a key would show a license notice.
That does not fit an MIT project with a public repository and CI on pull
requests from forks. The owner asked on 2026-10-20 to plan a replacement.

What PrimeVue does for the project today is small:

- 12 components: Button, InputText, InputNumber, Textarea, Select,
  ToggleSwitch, Tag, Message, Card, ProgressSpinner, Toast (with
  ToastService) and ConfirmDialog (with ConfirmationService); 31 of the 76
  `.vue`/`.ts` files under `web/src` import PrimeVue, about 3,400 lines of
  `.vue` code in total.
- 35 `primeicons` classes.
- The look is already the project's own: `tokens.css` (ADR-0007),
  `layout.css` (ADR-0010) and a 168-line preset that re-tokenises Aura to
  the Marionette palette and fixes its contrast (UX spec §8.2).

The hard parts of a component library for this UI are the accessible
behaviour of Select (listbox, typeahead, keyboard), the confirmation
dialog (focus trap, return focus, `alertdialog`), the toast (live region)
and the switch, which FR-28 and NFR-09 require. The rest is markup and
CSS on top of the existing tokens.

## Decision

1. Interactive primitives with non-trivial accessibility come from
   **Reka UI** (MIT, headless, Vue 3; formerly Radix Vue): Select,
   AlertDialog for confirmations, Toast, Switch, NumberField and, where an
   icon-only control needs one, Tooltip. Reka UI renders no styles of its
   own.
2. The project owns a small styled component layer in `web/src/ui/`
   (`MButton`, `MInput`, `MTextarea`, `MSelect`, `MSwitch`,
   `MNumberInput`, `MTag`, `MMessage`, `MCard`, `MSpinner`, a confirmation
   composable and a toast composable), built on Reka UI where point 1
   applies and plain elements elsewhere, styled only with `tokens.css`.
   Views use these components, not Reka UI directly.
3. Icons come from **Lucide** (`@lucide/vue`, ISC) as individually
   imported SVG components, wrapped by one `MIcon` component that sets size
   and `aria-hidden`; `primeicons` is removed. The card's `icon` value
   changes from a PrimeIcons class (`pi pi-home`) to the icon's name
   (`home`, lowercase letters, digits and dashes, at most 64 characters);
   `web/src/ui/icons.ts` maps the supported names to Lucide icons. The
   owner decided (2026-10-20) that existing values are not migrated:
   - a configuration with old values still loads (the icon is not part of
     `ValidateEssential`, ADR-0004);
   - a card whose icon the UI does not know shows the Lucide **ghost**
     icon on the dashboard and in the detail;
   - the editor offers only the supported icons; a card with an unknown
     icon cannot be saved until a supported one is chosen, and the API
     rejects the old form with 422 (`icon`).
4. The migration is incremental. Both libraries coexist while views move
   over; each block keeps `make verify` and `make e2e` green. The last block
   removes `primevue`, `@primeuix/*`, `primeicons` and the preset.
5. A new major version of Reka UI or Lucide needs a changelog and licence
   check, as for any key UI dependency (`docs/devops/ci-cd.md`).
6. The migration is also a visual refresh (owner, 2026-10-20): the layout
   of the screens (UX spec §2, §5–§7) stays, while tokens, typography,
   buttons, badges, form controls and overlays get a more current look
   within the visual direction of UX spec §8.1. The UX specification is
   updated with the new values.

FR-23 changes from "Vue 3 + PrimeVue" to "Vue 3 with accessible headless
primitives (Reka UI) and the project's own styled components" once this
ADR is accepted. This ADR supersedes ADR-0003 in the choice of component
library and icons (Vue 3, Vite and `vue-router` stay) and updates ADR-0007
(the project layer becomes the source of the basic components).

## Considered alternatives

- **Stay on PrimeVue 4 (MIT) and watch its support** — the cheapest today
  and legally fine, but it leaves the project on a line the vendor has
  moved away from; every update needs a licence check, and the eventual
  switch would have to happen under pressure. Kept as the state until the
  migration is done.
- **PrimeVue 5 under the Community License** — rejected: per-developer
  licence and key, not usable for contributors and fork CI without
  workarounds, terms can change (see the context).
- **A styled MIT library (Naive UI, Element Plus, Vuetify, Quasar)** —
  faster to adopt, but each brings its own design language and theme
  system, so the Marionette tokens would again be mapped onto a foreign
  theme as with Aura; they are larger than the 12 components needed and
  would keep the bundle above Vite's 500 kB warning. Rejected.
- **shadcn-vue** — copies Reka UI based components into the repository,
  close to this decision, but its styles are Tailwind classes, which would
  replace `tokens.css` and `layout.css` (ADR-0010). Rejected; its
  components can serve as a reference for markup.
- **Own components without a library** — rejected: reimplementing
  listbox, dialog focus management and live regions is where accessibility
  bugs come from, and FR-28/NFR-09 depend on them.
- **Headless UI (Tailwind Labs)** — MIT, but its Vue package offers fewer
  primitives (no toast, number field or tooltip), so more of them would be
  written by hand.

## Consequences

- No PrimeTek package in the bundle and no licence key, for anyone who
  builds the project.
- The screens keep their layout and behaviour; their look is refreshed
  from `tokens.css` (point 6). The Aura preset and its contrast fixes
  disappear; contrast is checked again for the new palette (NFR-09).
- More code owned by the project: about a dozen small components with
  their CSS and Vitest tests. Behaviour that Reka UI provides (keyboard,
  focus, ARIA) is covered by its tests; the project tests its own props,
  states and wiring.
- The bundle should shrink (PrimeVue's styled runtime and Aura tokens go,
  icons are tree-shaken); this addresses the bundle size backlog item,
  measured at each block.
- Unit tests that install `primevue/config` change as their components
  move; the E2E suite (block 0041) checks each step from the browser,
  including 320 px and the accessibility scenarios.
- The API no longer accepts `pi pi-<name>` icons, an incompatible API
  change: the release with block 1 is **v2.0.0** (`docs/devops/ci-cd.md`,
  release versioning). Configurations keep loading; each card gets its new
  icon the next time it is edited.
- AGENTS.md and the Vue instructions change from "PrimeVue components
  instead of custom UI elements" to "components from `web/src/ui/`".
- Proposed block order, each a separate pull request:
  1. foundation: Reka UI and `@lucide/vue` added, `MIcon`, `MButton`,
     `MSpinner`, `MTag`, `MMessage`, `MCard`, icons replaced;
  2. form controls: `MInput`, `MTextarea`, `MNumberInput`, `MSelect`,
     `MSwitch` and the card editor;
  3. feedback: toast and confirmation dialog, the services removed;
  4. removal of PrimeVue, `@primeuix/*`, `primeicons` and the preset;
     documentation (FR-23, ADR-0003/0007 notes, UX spec).
