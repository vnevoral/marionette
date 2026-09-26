# ADR-0009: PrimeFlex jako zamrzlá layout vrstva pro MVP

- **Stav**: Přijato
- **Datum**: 2026-09-26

## Kontext

Bloky 0019 a 0020 zavedly PrimeFlex 4 jako závaznou utility vrstvu pro
layout (grid, flex, spacing) vedle PrimeVue komponent a vlastních design
tokenů. Revize projektu 2026-09-26 upozornila, že upstream PrimeFlex je
zamrzlý na verzi 4.0.0 (únor 2025), PrimeTek ho dále nerozvíjí a jako
náhradu doporučuje Tailwind CSS s pluginem `tailwindcss-primeui`. PrimeFlex
navíc tahá kompletní CSS bez tree-shakingu (~370 kB CSS v bundlu před gzip
společně s Aura theme). Projekt potřebuje rozhodnout, zda do fáze 8 a dál
stavět na zamrzlé knihovně, nebo migrovat.

## Rozhodnutí

PrimeFlex 4.0.0 zůstává layout vrstvou Marionette po dobu MVP jako
**zamrzlá, ale dostačující** závislost. Nezavádí se žádná další utility
knihovna a nemigruje se na Tailwind. Podmínky:

- verze je pevně připnutá (`4.0.0`, bez `^`) a Dependabot ji neaktualizuje;
- nové obrazovky používají pouze podmnožinu tříd již použitou v projektu
  (grid, flex, gap, spacing, display, text alignment) — rozsah je
  zdokumentován v UX specifikaci;
- veškeré vizuální přizpůsobení PrimeVue komponent jde přes `definePreset`
  a design tokeny (blok 0033), ne přes `!important` nebo PrimeFlex přepisy;
- rozhodnutí se znovu otevře, pokud (a) PrimeVue 5 přestane být s PrimeFlex
  kompatibilní a projekt bude chtít na PrimeVue 5 přejít (viz ADR-0003),
  nebo (b) velikost CSS bundlu začne měřitelně zpomalovat načtení na
  Raspberry Pi (NFR-03).

## Zvažované alternativy

- **Migrace na Tailwind CSS + `tailwindcss-primeui`** — aktivně udržované,
  tree-shaking, oficiální směr PrimeTek. Zamítnuto pro MVP: vyžaduje
  přepis všech tří view a komponent, zavádí PostCSS build krok a nový
  slovník tříd bez uživatelského přínosu; hodí se jako samostatná fáze po
  stabilizaci (fáze 8).
- **Odstranění utility vrstvy, pouze scoped CSS + tokeny** — nejmenší
  závislosti, ale opakuje layout kód napříč view a jde proti blokům
  0019/0020, které duplicitní scoped CSS právě odstraňovaly. Zamítnuto.
- **Ponechat PrimeFlex bez omezení** — riziko tichého rozšiřování závislosti
  na zamrzlé knihovně. Zamítnuto ve prospěch podmíněného zmrazení výše.

## Důsledky

- Žádná práce navíc ve fázi 8; bloky 0029, 0032 a 0033 staví na stávajícím
  layoutu.
- Bundle CSS zůstává větší, než by byl s tree-shakingem; pro LAN nasazení
  a jednotky karet je to přijatelné (NFR-03 měřeno na referenčním hostu).
- Upgrade na PrimeVue 5 bude vyžadovat revizi tohoto ADR společně s
  ADR-0003.
- `docs/devops/ci-cd.md` (verzování závislostí) a UX specifikace uvádějí
  PrimeFlex jako připnutou závislost s omezenou sadou tříd.
