# Implementační blok: Opravy z code review 2 — frontend (baseline z 202, sdílený průběh akce, snapshot guard)

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-21, FR-22, FR-26, FR-42, NFR-08
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0029, 0033, 0037

## Cíl bloku

Po dokončení čekání na novou kontrolu vychází z okamžiku přijetí požadavku
serverem, průběh akce (queued → enqueue → running → wait → výsledek) existuje
v kódu jednou, snapshot na detailu nepřepíše novější starším, rozpočet čekání
u ruční kontroly odpovídá timeoutu status akce a odkazy „New card“ / „Edit
card“ jsou PrimeVue tlačítka.

## Rozsah

- **Uvnitř** (nálezy code review 2026-09-26, druhé kolo):
  1. API: odpověď 202 z `POST …/actions/primary|status/check` nese
     `checkedAt` posledního známého checku v okamžiku přijetí (server jej
     čte před zařazením do fronty); `AcceptedAction.checkedAt?: string`.
     Frontend použije tuto hodnotu jako baseline pro `waitForNewerStatus`
     místo hodnoty zachycené před požadavkem.
  2. Nový composable `useActionRequest.ts`: `requestAction(card, action,
     { signal, onPhase, onSnapshot, onError }) → ActionOutcome`
     (`accepted | updated | timeout | aborted | failed(message)`) a
     `outcomeResult(outcome, labels)`; `HomeView` i `CardDetailView` jej
     používají, vlastní `runAction` jen mapuje stav a slovník.
  3. `useCardStatus`: `apply`/`set` respektují `supersedes`; `set(undefined)`
     stále maže; `waitForNewer` z `CardStatus` odstraněn (průběh řeší
     `requestAction`).
  4. `waitBudgetMs(card, action)`: status akce → `(status.timeoutSec +
     30) × 1000` (rezerva na frontu), primární → fast okno nebo
     `defaultMaxWaitMs`.
  5. `.primary-action-link` z `tokens.css` odstraněn; „New card“ a „Edit
     card“ jsou `<Button asChild>` s `RouterLink` uvnitř a třídou
     `primary-action-button` (AGENTS.md: PrimeVue komponenty místo vlastních
     UI prvků).
  6. UX spec §4: baseline z přijetí požadavku; rozpočet čekání u ruční
     kontroly.
- **Mimo rozsah**: Toast, E2E, další refaktor views.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno („oprav všechny nálezy“). Rozšíření
  202 o `checkedAt` je jediná změna API kontraktu; je zpětně kompatibilní
  (pole je volitelné, klient bez něj se chová jako dosud).

## Návrh řešení

- `server.go`: `acceptedAction.CheckedAt *time.Time
  `json:"checkedAt,omitempty"`` z `store.GetStatus(cardID)` před enqueue.
- `useActionRequest.ts` (viz rozsah); `HomeView.runAction` ~15 řádků,
  `CardDetailView.runAction` ~20 řádků.
- `useCardStatus.ts`: `apply(id, next)` → `if (supersedes(next,
  snapshot.value ?? undefined))`; `set(next)` → `undefined` maže, jinak
  jako `apply`.

## Testovací plán

- `useActionRequest.spec.ts`: accepted bez kontroly; updated přes stream;
  timeout; failed (enqueue reject); aborted; baseline z 202 — SSE se
  stejným `checkedAt` jako v 202 nevyřeší čekání, novější ano.
- `useCardStatus.spec.ts`: `set`/stream nepřepíše novější snapshot starším.
- `HomeView.spec.ts`, `CardDetailView.spec.ts`: stávající scénáře +
  baseline z 202 (H1/H2 mock vrací `checkedAt`).
- `server_test.go`: 202 nese `checkedAt` po `UpdateStatus`, bez něj pole
  chybí.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `grep -rn 'primary-action-link' web/src` je prázdný;
- `grep -c 'waitForNewerStatus' web/src/views/*.vue` = 0.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel; Vitest 85 → 93 testů. Nové:
  `useActionRequest.spec.ts` (accepted bez čekání; baseline z 202 — událost
  se stejným ani starším `checkedAt` čekání nevyřeší, novější ano; ruční
  kontrola vyprší po `timeoutSec + 30 s`; failed/aborted; `outcomeResult`),
  `useCardStatus.spec.ts` (`set`/`apply`/stream nepřepíší novější snapshot,
  `fail` jen pro sledovanou kartu, `waitBudgetMs` per akce, `isNewerCheck`
  odmítne starší check), `HomeView.spec.ts` (check známý serveru při
  přijetí není výsledkem), `CardDetailView.spec.ts` (načtená karta se
  starším checkem nepřepíše novější událost). `server_test.go`: 202 bez
  `checkedAt` u nezkontrolované karty, s `checkedAt` po `UpdateStatus`.
  Kritéria: `grep primary-action-link web/src` = 0, `waitForNewerStatus`
  ve views = 0; `HomeView.vue` 206 řádků, `CardDetailView.vue` 279.
- **Odchylky od návrhu**: (1) `isNewerCheck` porovnává pořadí časů (`>`)
  místo pouhé nerovnosti — bez toho by událost se starším `checkedAt` než
  baseline z 202 ukončila čekání; test to odhalil. (2) `CardStatus` místo
  `waitForNewer` vystavuje `apply(id, snapshot)` a `fail(id)`, aby průběh
  akce v `requestAction` zapisoval jen pro kartu, pro niž byl spuštěn, i po
  změně route. (3) Rozpočet ruční kontroly nemá spodní hranici
  `defaultMaxWaitMs`: u `timeoutSec` 5 s je čekání 35 s, což odpovídá
  „výsledek nedorazil“ přesněji než 120 s.
- **Dokumentace aktualizována**: ano — UX spec §4 (baseline z 202,
  rozpočty čekání), komentář `acceptedAction` v `server.go` a
  `AcceptedAction` v `api.ts`, roadmapa.
