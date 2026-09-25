# Implementační blok: Orchestrace primární a status akce

- **Fáze**: 5 — REST API
- **Vazba na požadavky**: FR-11, FR-12, FR-13, FR-14, FR-15a, FR-17, FR-18, FR-40
- **Vazba na ADR**: ADR-0004, ADR-0005, ADR-0006
- **Stav**: Návrh
- **Závislosti**: Blok 0010; `config.Store`, `execengine.Runner`, `StatusCheckService`, `Scheduler`

## Cíl bloku

REST API přijme požadavek na primární nebo ruční status akci a vrátí klientovi
odpověď bez čekání na dokončení procesu. Samotný běh pokračuje na background
workeru; primární výsledek se uloží do historie, status výsledek aktualizuje
projekci. Čtení statusu zůstává oddělenou read-only operací.

## Rozsah

- **Uvnitř**:
  - `POST /api/cards/{id}/actions/primary`;
  - `POST /api/cards/{id}/actions/status/check`;
  - primární endpoint vrací `202 Accepted` po přijetí validního požadavku;
  - background orchestrace `Runner.Run` → `Result.ToRun("primary")` → `AppendRun`;
  - volání `Scheduler.NotifyPrimaryAction` ihned po přijetí primární akce;
  - status endpoint ověří status akci a zařadí `StatusCheckService.CheckNow` na
    background worker;
  - oba enqueue endpointy vracejí pouze potvrzení zařazení.
- **Mimo rozsah**:
  - nové process execution pravidlo;
  - změna status transition modelu;
  - veřejný async job resource, polling jobu, cancellation a prioritizace;
  - scheduler reconcile a lifecycle (0012).

## Schválení

- **Schválil**: čeká
- **Datum schválení**: čeká
- **Poznámky k rozhodnutí**: Primární request vrací přijetí, ne výsledek procesu.
  Selhání se po dokončení zaznamená do `Run` jako `fail` nebo `timeout`; status
  karty je autoritativní cesta pro zjištění výsledného stavu.

## Návrh řešení

Handler obdrží rozhraní pro background execution manager, status checker a
scheduler notifier. Primární endpoint ověří kartu a požadavek, okamžitě zavolá
`NotifyPrimaryAction` a předá `card.Primary` background workeru. Worker následně
zavolá `Runner.Run`, převede výsledek přes `Result.ToRun("primary")` a uloží jej
přes `AppendRun`. Background manager musí být součástí lifecycle aplikace, aby
shutdown počkal na přijaté primární běhy před `SaveFileWithHistory`.

Status se v tomto handleru nespouští; read-only status handler pouze vrací
poslední uloženou projekci. Doporučené tělo odpovědi primárního endpointu je
jen potvrzení přijetí s `cardId` a
`actionKind: "primary"`; výsledek se čte z historie/status endpointu.

## Testovací plán

- primární endpoint vrátí `202` bez čekání na dokončení procesu;
- background worker zpracuje outcome `ok`, `fail` a `timeout`;
- uložení primárního běhu v newest-first historii;
- notifikace scheduleru před dokončením primární akce;
  - status enqueue s `ok`/`fail`, chybějící status akcí a neexistující kartou;
  - runner/setup chyba a persistence chyba;
  - ověření, že background spuštění neobchází scheduler ani sdílený runner;
  - ověření, že `GET /status` pouze čte poslední snapshot a nic nespouští.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
primární endpoint nečeká na proces, každý přijatý běh je před shutdownem
drainovatelný a žádné spouštění neobchází sdílený concurrency-limited runner.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
