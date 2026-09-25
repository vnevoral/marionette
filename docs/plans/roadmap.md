# Roadmapa a implementační bloky

Fáze odpovídají hrubému pořadí implementace. Každá fáze se před implementací
rozpracuje promptem `/plan-block` do jednoho nebo více konkrétních
implementačních bloků (viz [šablona](template-implementation-block.md)) a
uloží do `docs/plans/blocks/NNNN-nazev.md`. Blok se implementuje promptem
`/implement-block`.

## Přehled fází

| Fáze | Název                                                                                        | Stav                                             | Vazba na požadavky             |
| ---- | -------------------------------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------ |
| 0    | Bootstrap projektu (repo, Go+Vue+PrimeVue skelet, Makefile, embed, devops/agent scaffolding) | **Hotovo**                                       | FR-01..04 (základ)             |
| 1    | Zpřesnění požadavků a architektury                                                           | **Hotovo** (requirements v0.2, ADR-0004 přijato) | vše (SRS review)               |
| 2    | Doménový model + config store (in-memory + JSON perzistence)                                 | **Hotovo**                                       | FR-10..14, FR-30..33, ADR-0004 |
| 3    | Execution engine (bezpečné spouštění akcí na hostu)                                          | **Hotovo**                                       | FR-11, FR-13, NFR-01, NFR-04   |
| 4    | Status/health-check engine (vč. volitelného pollingu)                                        | **Probíhá** (0007 hotovo; 0008–0009 čekají)      | FR-12, FR-15, FR-16, FR-17     |
| 5    | REST API (CRUD karet/akcí, spuštění, čtení stavu)                                            | Plánováno                                        | FR-40, FR-41                   |
| 6    | Dashboard UI (karty, stavové indikátory, formuláře správy)                                   | Plánováno                                        | FR-20..23                      |
| 7    | Balíčkování a nasazení (systemd unit, install skript, arm64 release)                         | Plánováno                                        | FR-02..04, NFR-02              |
| 8    | Zpevnění (auth/access control, logování, chybové stavy, testy, dokumentace)                  | Plánováno                                        | NFR-01, NFR-04..06             |

## Poznámky k plánování

- Fáze 1 je uzavřena: `docs/requirements/requirements.md` je na v0.2 (otevřené
  otázky vyřešeny, viz sekce 7 dokumentu) a
  [ADR-0004](../architecture/decisions/0004-action-card-domain-model.md) je
  přijato vč. historie běhů, pravidla na výstup, pollingu a limitu
  souběžnosti.
- Fáze 2 byla rozpracována do implementačních bloků a **schválena** vlastníkem
  projektu (2026-09-25). Bloky 0001–0003 jsou hotové:
  - [0001 — Doménové typy a validace konfigurace](blocks/0001-config-domain-types.md)
  - [0002 — In-memory config store (CRUD karet + historie běhů)](blocks/0002-config-inmemory-store.md)
  - [0003 — JSON perzistence konfigurace, historie a načtení při startu](blocks/0003-config-json-persistence.md)

  Bloky se implementují v tomto pořadí (0002 staví na typech z 0001, 0003 na
  store z 0002).

- Fáze 3 byla rozpracována do ADR-0005 a implementačních bloků 0004–0006,
  které jsou hotové:
  - [ADR-0005 — Bezpečné spouštění akcí a limit souběžnosti](../architecture/decisions/0005-execution-engine.md)
  - [0004 — Spuštění procesu a vynucený timeout](blocks/0004-execution-process.md)
  - [0005 — Zachycení výstupu a vyhodnocení výsledku](blocks/0005-execution-result.md)
  - [0006 — Globální limit souběžných akcí](blocks/0006-execution-concurrency.md)

- ADR-0006 je přijato: status historie obsahuje pouze skutečné přechody stavů
  s dobou jejich trvání; migrace starších status historií není součástí MVP.

- Fáze 4 je rozpracována do bloků 0007–0009 a bude se implementovat v pořadí:
  0007 (status projekce a přechody) → 0008 (jedna status kontrola) → 0009
  (standardní a zrychlený polling scheduler).

- Bloky uvnitř fáze by měly být dost malé na jednu implementační relaci s AI
  agentem (řádově hodiny práce, ne dny) a musí mít jasné kritérium hotovosti
  (viz [Definition of Done](../devops/definition-of-done.md)).
- Pořadí fází 3 a 4 lze prohodit/sloučit, pokud se ukáže, že status akce a
  primární akce sdílí prakticky celou implementaci (jde o stejný typ
  „Action“, jen jiné volání).
