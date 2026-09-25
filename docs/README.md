# Dokumentace projektu Marionette

Tento adresář je zdroj pravdy pro požadavky, architektonická rozhodnutí a plán
implementace. Kód by se s ním neměl rozcházet — pokud ano, nejdřív se
aktualizuje dokumentace (nový requirement / ADR), pak kód.

- [requirements/requirements.md](requirements/requirements.md) — funkční a
  nefunkční požadavky (SRS), stav: **návrh v0.1, ke zpřesnění**
- [requirements/glossary.md](requirements/glossary.md) — pojmy domény
- [architecture/overview.md](architecture/overview.md) — přehled komponent a
  jejich vazeb
- [architecture/decisions/](architecture/decisions) — log architektonických
  rozhodnutí (ADR)
- [plans/roadmap.md](plans/roadmap.md) — fáze projektu a implementační bloky
- [devops/testing-strategy.md](devops/testing-strategy.md) — jak se testuje
- [devops/ci-cd.md](devops/ci-cd.md) — CI pipeline a release proces
- [devops/definition-of-done.md](devops/definition-of-done.md) — kdy je blok
  hotový

## Jak se v repozitáři pracuje s AI agenty

Viz [AGENTS.md](../AGENTS.md) v kořeni repozitáře — hlavní vstupní bod pro
agenty, a `.github/prompts/` pro opakovatelné úkoly (nový požadavek, nové ADR,
plánování bloku, implementace bloku).
