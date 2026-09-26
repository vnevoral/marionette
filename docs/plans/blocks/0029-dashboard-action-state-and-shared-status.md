# Implementační blok: Stav akcí na dashboardu a sdílené sledování statusu

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-20, FR-21, FR-27, FR-42, NFR-03, NFR-11
- **Vazba na ADR**: ADR-0007, ADR-0008
- **Stav**: Hotovo
- **Závislosti**: Bloky 0013, 0016, 0018, 0022

## Cíl bloku

Po dokončení se tlačítka karty po dokončení akce znovu uvolní, detail karty
se po spuštění akce aktualizuje stejně jako dashboard a logika SSE/polling
žije v jednom sdíleném composable místo v jedné view.

## Rozsah

- **Uvnitř**:
  - `HomeView`: záznam `requests[card.id]` se po `success`/`error` vymaže
    (výsledek zůstane viditelný po krátkou dobu přes samostatný stav
    `lastResult`); `disabled`/`loading` odvozeny pouze z `queued|running`;
  - nový composable `useStatusEvents()` — singleton `EventSource` sdílený
    mezi view, stav `connected`, reconnect, REST fallback polling, korektní
    `close()` po posledním odběrateli;
  - nový composable `useCardStatus(cardId)` — per-karta snapshot, čekání na
    nový `checkedAt` s `AbortController`, zrušení při unmountu;
  - `CardDetailView`: po přijetí akce (202) používá `useCardStatus`, po
    změně statusu refetch runs + history; tlačítka blokována během běhu;
    „Action queued“ se po dokončení nahradí výsledkem;
  - `CardDetailView` a `CardEditView`: `watch(() => route.params.id, …,
    { immediate: true })`, normalizace `string[]` parametru, ignorování
    stale odpovědí po změně id;
  - typ `AcceptedAction` pro odpověď enqueue endpointů v `api.ts`.
- **Mimo rozsah**:
  - vizuální změny karet, slovník stavů (blok 0033);
  - nahrazení `window.confirm` (blok 0032);
  - testovací infrastruktura (blok 0030) — testy se ale v tomto bloku píší,
    proto 0030 musí být hotový dříve.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava chyb. Nález revize 2026-09-26 (H1, H2, M2, M9).
  Ověřeno v kódu: `HomeView.vue` po první akci nechá tlačítka trvale
  disabled a spinner běží až do remountu.

## Návrh řešení

- `web/src/composables/useStatusEvents.ts`: modulová proměnná s
  `EventSource` a čítačem odběratelů; `subscribe(handler)` vrací
  `unsubscribe`; při `error` přepne `mode = "polling"` a spustí interval
  5 s; při `open` refetch všech statusů a stop intervalu.
- `web/src/composables/useCardStatus.ts`: `ref<StatusSnapshot|null>`,
  `waitForNewer(previousCheckedAt, { signal, maxWaitMs })` — primárně čeká
  na SSE event, fallback na REST každé 2 s jen v polling módu.
- `HomeView.vue`: `requests` typ zúžen na `queued|running`; `lastResult`
  s `setTimeout` 4 s pro zobrazení „Updated“/„Result not available yet“.
- `CardDetailView.vue`: `runAction` → `enqueue*` → `waitForNewer` →
  `Promise.allSettled([loadRuns(), loadHistory()])`.

## Testovací plán

- Vitest + Vue Test Utils (infrastruktura z 0030):
  - `useStatusEvents`: mock `EventSource`; dva odběratelé sdílí jedno
    připojení; po odhlášení obou se zavře; `error` → polling; `open` →
    stop polling;
  - `useCardStatus.waitForNewer`: vrátí se po eventu s novějším
    `checkedAt`; `abort` ukončí čekání bez zápisu; stejné `checkedAt`
    nepovažuje za nový;
  - `HomeView.runAction`: po úspěchu jsou tlačítka enabled a bez spinneru
    (regresní test H1); chyba enqueue → enabled + chybová zpráva;
  - `CardDetailView`: po akci se zavolá refetch runs/history; změna
    `route.params.id` znovu načte detail.
- `npm run lint`, `npm run build`, `npm test`.
- Manuální smoke test: run → check → run na stejné kartě bez reloadu;
  detail karty ukáže nový běh bez F5.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- v `web/src/views` není žádný přímý `new EventSource`;
- regresní test H1 a H2 prochází v CI;
- UX spec §4 a §6 aktualizované, pokud se chování zpřesnilo.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (lint, `vue-tsc`, Prettier, Vitest 45
  testů v 7 souborech, `vite build`, `go test -race`, `go vet`). Nové testy:
  `composables/useStatusEvents.spec.ts` (dva odběratelé sdílí jedno
  připojení, zavření po posledním odhlášení, polling do `open`, `open` →
  jednorázový refresh a stop pollingu, `error` → polling, fallback bez
  `EventSource`), `composables/useCardStatus.spec.ts` (`isNewerCheck`,
  `waitForNewerStatus`: event s novějším `checkedAt`, REST poll, chyba REST
  bez ukončení čekání, `timeout`, `abort` bez zápisu; `useCardStatus`:
  filtr podle karty, uvolnění se scope, refresh jen když `enabled`,
  ignorování stale odpovědi po změně id), `views/HomeView.spec.ts`
  (regrese H1: po dokončení akce jsou tlačítka enabled a bez spinneru,
  výsledek zmizí po 4 s, chyba enqueue → enabled + zpráva, karta bez
  status akce → „Accepted“, REST polling jen dokud stream neběží),
  `views/CardDetailView.spec.ts` (regrese H2: tlačítka blokována do
  nového `checkedAt`, poté refetch runs + history a „Status updated“;
  chyba enqueue; změna `route.params.id` znovu načte detail a stale
  odpověď se ignoruje; karta bez status akce; nenalezená karta). Regresní
  testy H1 ověřeny jako padající na původním `HomeView.vue`. V `web/src/views`
  není žádný přímý `new EventSource` (jediný je v `api.ts`). Manuální smoke
  test v prohlížeči (run → check → run bez reloadu) zůstává k ověření na
  referenčním hostu.
- **Odchylky od návrhu**: (1) `waitForNewer` dotazuje REST každé 2 s
  **vždy**, ne jen v polling módu — backend publikuje `status.changed` jen
  při změně stavu (`Store.UpdateStatus`), takže kontrola se stejným
  výsledkem změní `checkedAt` bez události; SSE událost čekání jen zkracuje.
  (2) `waitForNewer` vrací `"updated" | "timeout" | "aborted"` místo boolean,
  aby view po abortu (unmount, změna id) nic nezapisovalo. (3) Přibyla
  testovatelná továrna `createStatusEvents(connect, intervalMs)`; singleton
  `useStatusEvents()` se na ní staví a odběr registrovaný uvnitř scope
  komponenty se uvolní přes `onScopeDispose`. Odběratel dostává i `onRefresh`
  (polling tick / reconnect), aby view rozhodlo, co znovu načíst. (4) Režim
  je odvozen z `connected` (`live`/`polling`), polling běží i před prvním
  `open`, stejně jako dřívější `HomeView`. (5) `HomeView` čeká s vlastní
  mapou snapshotů přes `waitForNewerStatus` (funkce sdílená
  s `useCardStatus`), protože jeden composable na kartu by v seznamu nedával
  smysl. (6) `CardDetailView` pro kartu bez status akce hlásí „Action
  accepted“ a runs/history načte hned po 202 — bez status akce není signál
  dokončení. (7) Helper `router/params.ts::singleParam` normalizuje
  `string | string[]`; `CardEditView` sleduje `[route.name, id]`, protože
  stejná komponenta obsluhuje `new` i `edit`. (8) `AcceptedAction` je návratový
  typ `enqueuePrimary`/`enqueueStatus`. (9) Sdílený falešný `EventSource`
  pro testy je v `web/src/test/fakeEventSource.ts`.
- **Dokumentace aktualizována**: ano — UX spec §4 (výsledek po dokončení,
  uvolnění tlačítek, SSE + REST fallback) a §6 (Actions v detailu),
  `docs/devops/testing-strategy.md` (composables a view testy),
  `docs/architecture/overview.md` (řádek SPA), roadmapa.
