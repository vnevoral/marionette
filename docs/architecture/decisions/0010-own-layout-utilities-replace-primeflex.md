# ADR-0010: Vlastní layout utility místo PrimeFlexu

- **Stav**: Přijato
- **Datum**: 2026-09-26

## Kontext

ADR-0009 ponechalo PrimeFlex 4.0.0 jako zamrzlou layout vrstvu pro MVP
s tím, že se rozhodnutí znovu otevře po fázi 8. Fáze 8 je uzavřená a
analýza 2026-09-26 ukázala:

- PrimeFlex 4.0.0 (únor 2025) je poslední verze; PrimeTek ho nerozvíjí a
  doporučuje Tailwind CSS s pluginem `tailwindcss-primeui`.
- Kód Marionette používá **14 utility tříd** v přibližně 60 výskytech
  (`grid`, `col-12`, `md:col-4`, `md:col-6`, `lg:col-4`, `flex`,
  `flex-column`, `flex-wrap`, `align-items-center`,
  `justify-content-between`, `gap-2`, `gap-3`, `p-2`, `mt-4`).
- PrimeFlex přidává 446 kB CSS bez tree-shakingu; celý CSS bundle má
  368 kB minifikovaných.
- PrimeVue 5.0.0 (červenec 2026) a 5.0.1 (srpen 2026) přešly z MIT na
  komerční **PrimeUI License** s offline licenčním klíčem
  (`@primeui/license-manager`); bezplatná Community licence platí jen pro
  organizace pod 1 mil. USD obratu, méně než 5 vývojářů a méně než 10
  zaměstnanců a vyžaduje roční obnovu. PrimeVue 4.5.5 zůstává MIT pod
  dist-tagem `v4-stable`. Budoucí kompatibilita PrimeFlexu s PrimeVue 5,
  kterou ADR-0009 uvádělo jako spouštěč, tedy není relevantní.

## Rozhodnutí

PrimeFlex se z projektu odstraňuje. Layout utility, které kód používá,
definuje vlastní soubor `web/src/styles/layout.css` se **stejnými názvy tříd
a stejnými hodnotami** (včetně breakpointů 768 px a 992 px), takže šablony
zůstávají beze změny a vizuální výsledek je shodný. Nové utility třídy se
přidávají jen do tohoto souboru a jen když je nějaká view skutečně potřebuje;
neexistuje generovaná sada „pro jistotu“.

PrimeVue zůstává na větvi **v4 (MIT)**; upgrade na PrimeVue 5 by znamenal
licenční klíč v buildu a pro nasazení mimo Community kritéria placenou
licenci na vývojáře, což pro malý self-hosted nástroj nedává smysl. Toto
rozhodnutí doplňuje ADR-0003.

## Zvažované alternativy

- **Tailwind CSS + `tailwindcss-primeui`** — aktivně udržované, oficiální
  směr PrimeTek. Zamítnuto: pro 14 tříd přináší nový build plugin, nový
  slovník tříd a přepis všech šablon bez uživatelského přínosu. Lze zvážit,
  pokud by frontend výrazně narostl.
- **Ponechat PrimeFlex zamrzlý (ADR-0009)** — funkční, ale trvale nese
  446 kB mrtvého CSS a závislost bez údržby. Zamítnuto.
- **Upgrade na PrimeVue 5** — mění licenci a vyžaduje licenční klíč.
  Zamítnuto pro MVP i další fáze, dokud se nezmění licenční model nebo
  potřeby projektu.

## Důsledky

- Závislost `primeflex` mizí z `package.json`, lockfile i Dependabot
  ignore seznamu; CSS bundle se zmenší řádově o stovky kB.
- `layout.css` je pod kontrolou projektu: změny hodnot nebo breakpointů
  jsou vědomé a viditelné v review.
- Dependabot dál ignoruje major verze `primevue` a `@primevue/themes`;
  přechod na PrimeVue 5 vyžaduje nové ADR s licenčním posouzením.
- ADR-0009 je tímto ADR nahrazeno; `docs/devops/ci-cd.md`, requirements
  sekce 13 a UX specifikace odkazují na toto ADR.
