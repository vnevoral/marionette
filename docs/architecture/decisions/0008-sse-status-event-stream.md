# ADR-0008: SSE stream pro živé změny statusů

- **Stav**: Přijato
- **Datum**: 2026-09-25

## Kontext

Status scheduler aktualizuje projekci stavu karet na serveru, ale browser se o
změně dozvídá až při ručním obnovení, navigaci nebo periodickém REST pollingu.
Marionette potřebuje jednosměrný transport pro okamžité doručení změn statusů
bez zavádění obousměrného protokolu nebo externího message brokeru.

Požadavek je veden ve FR-42 a NFR-11. Status projekce již rozlišuje poslední
kontrolu od skutečného přechodu stavu (ADR-0006), takže opakované kontroly
stejného stavu nemají vytvářet zbytečné klientské události.

## Rozhodnutí

Marionette použije Server-Sent Events pro živé doručování změn statusů ze
serveru do připojených browser klientů.

- Veřejný endpoint bude poskytovat `text/event-stream` a událost
  `status.changed`.
- Událost ponese `id`, identifikátor karty a JSON `StatusSnapshot`.
- Událost se publikuje pouze po změně interpretovaného stavu karty; opakovaná
  kontrola bez změny stavu se publikuje pouze jako interní aktualizace projekce.
- Server bude posílat heartbeat a při ukončení HTTP request contextu odběratele
  bezpečně odhlásí bez blokování scheduleru, REST API nebo ostatních klientů.
- Klient použije nativní `EventSource`, automatický reconnect a REST read-only
  API jako počáteční načtení i fallback při nedostupném SSE.
- SSE nenahrazuje interní scheduler, status service ani REST kontrakt a
  nevyžaduje externí broker.

## Zvažované alternativy

- **Polling pouze z browseru** — zamítnuto jako hlavní řešení; zvyšuje počet
  dotazů, přináší zpoždění a klient neví, kdy se změna skutečně stala. Zůstává
  jako fallback.
- **WebSocket** — zamítnuto; aplikace potřebuje pouze server → browser tok a
  nemá požadavek na obousměrnou komunikaci.
- **Externí message broker** — zamítnuto; odporuje cíli jednoho binárního
  procesu bez runtime/databázových závislostí.

## Důsledky

- Dashboard může zobrazit změnu statusu prakticky okamžitě bez ručního refresh.
- Přibude lifecycle a paměťová správa připojených SSE klientů; broadcaster
  musí mít omezené buffery a nesmí čekat na pomalého odběratele.
- REST API zůstává nutné pro první načtení, reconnect synchronizaci a fallback.
- SSE endpoint musí být součástí HTTP testů a browser smoke testu včetně
  odpojení, reconnectu a paralelních klientů.
- V MVP se nepersistuje event log a klient nemůže žádat historické události;
  při reconnectu načte aktuální projekci přes REST.

## Doplnění 2026-09-26 (blok 0027)

Broadcaster (`StatusEventBroker`) má `Close()`, které při řízeném ukončení
aplikace odpojí všechny odběratele a ukončí každý běžící `/api/events`
handler okamžitě; volá se z `http.Server.RegisterOnShutdown`, takže
`Shutdown` na dlouhožijící SSE spojení nečeká. Události publikované po
`Close()` se zahazují. Klient se po restartu služby připojí znovu díky
nativnímu reconnectu `EventSource`.
