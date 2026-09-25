# Implementační blok: Kompozice aplikace, reconcile a lifecycle

- **Fáze**: 5 — REST API
- **Vazba na požadavky**: FR-15, FR-15a, FR-17, FR-35, FR-40
- **Vazba na ADR**: ADR-0004, ADR-0006
- **Stav**: Návrh
- **Závislosti**: Bloky 0010–0011; `cmd/marionette/main.go`, scheduler, persistence

## Cíl bloku

Aplikace sestaví všechny služby z jedné konfigurace, spustí scheduler před HTTP
serverem a při CRUD změnách živě přepočítá polling workery. Při graceful
shutdownu nejprve zastaví příjem nových požadavků a počká na přijaté
background primární akce, aby se jejich historie uložila před ukončením.

## Rozsah

- **Uvnitř**:
  - konstrukce executor/runner/status service/scheduler/router v `main`;
  - background execution manager pro přijaté primární i ruční status akce;
  - start scheduleru v application contextu;
  - `Scheduler.Reconcile` pro přidání, odebrání a retiming workerů;
  - volání reconcile po úspěšném create/update/delete karty;
  - shutdown pořadí: zastavit přijímání HTTP, počkat na background primární
    akce, zastavit scheduler, ukončit HTTP server a volat `SaveFileWithHistory`;
  - dokumentace residual risku neomezeného čekání `Scheduler.Stop`.
- **Mimo rozsah**:
  - context-aware cancellation queued runneru;
  - hard shutdown deadline;
  - settings endpoint a dynamická změna runner concurrency;
  - deployment/systemd.

## Schválení

- **Schválil**: čeká
- **Datum schválení**: čeká
- **Poznámky k rozhodnutí**: Současné `Scheduler.Stop` zůstává; může čekat déle než pětisekundový HTTP shutdown timeout.

## Návrh řešení

`main` načte store, nastaví `OnChange`, vytvoří sdílený executor a runner,
background execution manager, status check service, scheduler a server router
s dependency injection. Background manager eviduje všechny přijaté primární i
ruční status akce a při shutdownu je nechá doběhnout nebo korektně uzavřít podle
zvoleného execution kontraktu.
Scheduler dostane reconcile operaci, která je race-safe vůči `Start`/`Stop` a
při změně karty restartuje dotčený worker. CRUD handler ji zavolá až po úspěšné
mutaci store.

Při SIGINT/SIGTERM se nejprve zastaví přijímání nových HTTP požadavků, počká se
na přijaté background primární akce, zastaví se scheduler, ukončí HTTP server a
nakonec se uloží konfigurace včetně historie. Hard bounded shutdown zůstává
mimo tento blok a je případný budoucí blok.

## Testovací plán

- start/stop všech složených služeb v `main`;
- přijetí primární i ruční status akce, dokončení background běhu a persistence
  odpovídající projekce/historie;
- reconcile po create/update/delete a změně polling konfigurace;
- současný reconcile, polling a shutdown pod `go test -race`;
- ověření pořadí shutdownu a uložení primární i status historie;
- konstrukční chyby a čisté ukončení bez částečně zapojeného serveru.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
API a scheduler používají stejné instance store, runneru a status služeb a
graceful shutdown nepřijde o historii po dokončení workerů.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
