# Roadmapa a implementační bloky

Proces plánování a implementace je závazně popsán v
[docs/devops/development-workflow.md](../devops/development-workflow.md).

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
| 4    | Status/health-check engine (vč. volitelného pollingu)                                        | **Hotovo**                                       | FR-12, FR-15, FR-16, FR-17     |
| 5    | REST API (CRUD karet/akcí, spuštění, čtení stavu)                                            | **Hotovo**                                       | FR-40, FR-41                   |
| 6    | Dashboard UI (karty, stavové indikátory, formuláře správy)                                   | **Hotovo**                                       | FR-20..23                      |
| 7    | Balíčkování a nasazení (systemd unit, install skript, arm64 release)                         | **Připraven návrh bloku 0023**                   | FR-01..04, NFR-02              |
| 8    | Zpevnění (auth/access control, logování, chybové stavy, testy, dokumentace)                  | Plánováno                                        | NFR-01, NFR-04..06             |
| 9    | UX redesign a sdílený design systém                                                          | **Hotovo**                                       | FR-24..29, NFR-08..10          |
| 10   | Realtime doručování statusů přes SSE                                                         | **Hotovo**                                       | FR-42, NFR-11                  |

## Poznámky k plánování

- Fáze 1 je uzavřena: `docs/requirements/requirements.md` je na v0.5 (otevřené
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

- Fáze 4 byla rozpracována do bloků 0007–0009, které jsou hotové:
  - [0007 — Status projekce a transition historie](blocks/0007-status-projection.md)
  - [0008 — Status check service](blocks/0008-status-check-service.md)
  - [0009 — Standardní a zrychlený polling scheduler](blocks/0009-polling-scheduler.md)

- Fáze 5 je rozpracována do bloků 0010–0012 v pořadí:
  - [0010 — REST API kontrakt a HTTP transport](blocks/0010-rest-api-transport.md)
  - [0011 — Orchestrace primární a status akce](blocks/0011-action-orchestration.md)
  - [0012 — Kompozice aplikace, reconcile a lifecycle](blocks/0012-application-composition-lifecycle.md)

  Bloky 0010–0012 jsou hotové.

- Fáze 6 začíná blokem 0013:
  - [0013 — Dashboard karet a ovládání akcí](blocks/0013-dashboard-overview.md)
  - [0014 — Správa akčních karet](blocks/0014-card-management.md)

- Fáze 9 je připravena jako samostatná UX/design oblast. [ADR-0007](../architecture/decisions/0007-ui-design-system-and-ux.md)
  je přijato a výchozím jazykem UI je angličtina. Před implementací UX bloků
  se schválí detailní návrh obrazovek a interakcí v
  [UX specifikaci](../ux/ui-ux-specification.md).
  První schválený blok je [0015 — UX-01: Design tokeny a AppShell](blocks/0015-ux-foundation.md).
  Následuje [0016 — UX-02: Overview a action feedback](blocks/0016-ux-overview.md).
  Následuje [0017 — UX-03: Management CRUD workflow](blocks/0017-ux-management-crud.md).
  Následuje [0018 — UX-04: Detail karty, běhy a status timeline](blocks/0018-ux-card-detail.md).
  PrimeFlex layout foundation je zachycen v [0019](blocks/0019-primeflex-layout-foundation.md).
  Následuje [0020 — PrimeFlex view migration](blocks/0020-primeflex-view-migration.md).
  Posledním dokončeným blokem fáze je [0021 — UX-05: Vizuální konzistence,
  accessibility a responsive audit](blocks/0021-ux-consistency-accessibility.md).

- Fáze 10 je připravena požadavkem FR-42/NFR-11. [ADR-0008](../architecture/decisions/0008-sse-status-event-stream.md)
  je přijato a schválený implementační blok je [0022 — SSE: živé změny
  statusů](blocks/0022-sse-status-events.md).

- Fáze 7 je připravena návrhem bloku [0023 — systemd nasazení a release
  artefakt](blocks/0023-deployment-systemd-release.md) pro obecný Linux se
  systemd; Ubuntu 24.x na Raspberry Pi ARM64 slouží jako referenční validační
  prostředí. Blok čeká na schválení; do té doby se deployment kód
  neimplementuje.

- Bloky uvnitř fáze by měly být dost malé na jednu implementační relaci s AI
  agentem (řádově hodiny práce, ne dny) a musí mít jasné kritérium hotovosti
  (viz [Definition of Done](../devops/definition-of-done.md)).
- Pořadí fází 3 a 4 lze prohodit/sloučit, pokud se ukáže, že status akce a
  primární akce sdílí prakticky celou implementaci (jde o stejný typ
  „Action“, jen jiné volání).
