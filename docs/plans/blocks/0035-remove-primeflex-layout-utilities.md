# Implementační blok: Náhrada PrimeFlex vlastní utility vrstvou

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-25, NFR-03, NFR-08, NFR-10
- **Vazba na ADR**: ADR-0003, ADR-0009 (nahrazeno), ADR-0010
- **Stav**: Hotovo
- **Závislosti**: Bloky 0019, 0020, 0033

## Cíl bloku

Po dokončení frontend nezávisí na zamrzlém PrimeFlexu: čtrnáct layout tříd,
které kód skutečně používá, definuje vlastní `layout.css` se stejnými názvy
a hodnotami, šablony se nemění a CSS bundle se zmenší o balík PrimeFlex.
PrimeVue zůstává na MIT větvi v4.

## Rozsah

- **Uvnitř**:
  - `web/src/styles/layout.css` s třídami `grid`, `col-12`, `md:col-4`,
    `md:col-6`, `lg:col-4`, `flex`, `flex-column`, `flex-wrap`,
    `align-items-center`, `justify-content-between`, `gap-2`, `gap-3`,
    `p-2`, `mt-4` — hodnoty a breakpointy (768 px, 992 px) shodné
    s PrimeFlex 4.0.0, bez `!important`;
  - odstranění `primeflex` z `package.json`, lockfile, `main.ts`
    a z ignore seznamu Dependabotu;
  - ADR-0010 (nahrazuje ADR-0009), doplnění ADR-0003 o licenční změnu
    PrimeVue 5 a rozhodnutí zůstat na v4;
  - aktualizace `docs/devops/ci-cd.md`, `docs/requirements/requirements.md`
    (sekce 13), UX spec §8, roadmapy.
- **Mimo rozsah**:
  - Tailwind CSS nebo jiná utility knihovna (ADR-0010);
  - upgrade na PrimeVue 5 (licence PrimeUI, ADR-0003);
  - vizuální změny — layout musí zůstat pixelově stejný.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno na základě analýzy 2026-09-26:
  PrimeFlex 4.0.0 je poslední verze (únor 2025), kód používá 14 tříd v ~60
  výskytech, PrimeFlex tahá 446 kB CSS. PrimeVue 5.0.x (7–8/2026) přešlo na
  komerční PrimeUI License s licenčním klíčem; v4 zůstává MIT.

## Návrh řešení

- `layout.css`: `.grid` (flex, wrap, záporné okraje 0.5 rem), `.col-12`
  a responzivní `md:`/`lg:` varianty (`flex: 0 0 auto; padding: 0.5rem;
  width`), flex/gap/spacing utility s hodnotami PrimeFlex (`gap-2` 0.5 rem,
  `gap-3` 1 rem, `p-2` 0.5 rem, `mt-4` 1.5 rem).
- `main.ts`: `import "./styles/layout.css"` místo `primeflex/primeflex.css`.
- Přidání nové utility třídy vyžaduje zápis do `layout.css`; nepoužívané
  třídy se nezavádějí (ADR-0010).

## Testovací plán

- `make verify` (lint, typy, Vitest, build, Go).
- `grep -rn primeflex web/src web/package.json .github` vrací 0 řádků.
- Porovnání velikosti `internal/webui/dist/assets/*.css` před a po.
- Manuální screenshot smoke test tří obrazovek v 320 px, 768 px a desktopu
  (layout beze změny) — referenční host.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `primeflex` není v závislostech ani v kódu;
- žádná šablona se nemění (jen CSS a import);
- ADR-0009 má stav „Nahrazeno ADR-0010“.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (lint, `vue-tsc`, Prettier, Vitest 77
  testů, `vite build`, `go test -race`, `go vet`). `primeflex` odstraněn
  z `package.json`, lockfile (`npm uninstall`), `main.ts` i Dependabot
  ignore; `grep -rni primeflex web/src web/package.json .github` nachází jen
  vysvětlující komentář v `layout.css`. Žádná šablona se nezměnila (diff
  obsahuje jen CSS, import a dokumentaci). CSS bundle
  `internal/webui/dist/assets/*.css`: 367 604 B → 30 381 B (gzip
  39 859 B → 6 846 B). Vizuální screenshot smoke test v 320/768 px a na
  desktopu zůstává na referenční host; hodnoty i breakpointy jsou převzaté
  1:1 z PrimeFlex 4.0.0, takže se změna layoutu neočekává.
- **Odchylky od návrhu**: žádné. Třída `.grid > [class*="col"]`
  s `box-sizing: border-box` je zachována kvůli shodě s PrimeFlex, i když
  globální reset `* { box-sizing }` ji činí redundantní.
- **Dokumentace aktualizována**: ano — ADR-0010 (nové), ADR-0009 (stav
  „Nahrazeno ADR-0010“), ADR-0003 (dodatek o licenci PrimeVue 5 a setrvání
  na v4), `docs/devops/ci-cd.md`, `docs/requirements/requirements.md`
  sekce 13, UX spec §8.2, roadmapa, `.github/dependabot.yml`.
