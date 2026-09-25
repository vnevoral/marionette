# Dokumentace projektu Marionette

Tento adresář je zdroj pravdy pro požadavky, architektonická rozhodnutí a plán
implementace. Kód by se s ním neměl rozcházet — pokud ano, nejdřív se
aktualizuje dokumentace (nový requirement / ADR), pak kód.

- [requirements/requirements.md](requirements/requirements.md) — funkční a
  nefunkční požadavky (SRS), zdroj pravdy pro FR/NFR
- [requirements/glossary.md](requirements/glossary.md) — pojmy domény
- [architecture/overview.md](architecture/overview.md) — přehled komponent a
  jejich vazeb
- [architecture/decisions/](architecture/decisions) — log architektonických
  rozhodnutí (ADR)
- [plans/roadmap.md](plans/roadmap.md) — fáze projektu a implementační bloky
- [devops/testing-strategy.md](devops/testing-strategy.md) — jak se testuje
- [devops/ci-cd.md](devops/ci-cd.md) — CI pipeline a release proces
- [devops/development-workflow.md](devops/development-workflow.md) — závazný
  společný proces plánování a implementace
- [devops/definition-of-done.md](devops/definition-of-done.md) — výstupní
  kontrola, kdy je blok hotový

## Jak se v repozitáři pracuje s AI agenty

Viz [AGENTS.md](../AGENTS.md) v kořeni repozitáře — hlavní vstupní bod pro
agenty. Závazný proces je v
[devops/development-workflow.md](devops/development-workflow.md) a
`.github/prompts/` obsahuje jeho opakovatelné kroky.
