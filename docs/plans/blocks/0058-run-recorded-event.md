# Implementační blok: SSE událost po zapsání běhu primární akce

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-42a, FR-42, FR-17, FR-15, FR-27
- **Vazba na ADR**: ADR-0008, doplněk „2026-09-27: událost `run.recorded`“
  (přijato 2026-09-27)
- **Stav**: Hotovo
- **Závislosti**: Bloky 0034 (`internal/events`), 0050 (čekání na běh
  v detailu)

## Cíl bloku

Spuštění akce je asynchronní: `POST …/actions/primary` akci jen zařadí do
fronty a vrátí `202`. Kdy proces doběhne a běh se zapíše do historie, se
klient dnes nedozví. Detail proto po bloku 0050 opakovaně načítá `/runs`
v intervalu zrychleného pollingu, dokud nový běh neuvidí. Po dokončení
bloku server po zapsání běhu pošle událost po existujícím SSE streamu a
detail běh zobrazí hned, bez opakovaných dotazů.

Průběh po bloku:

1. `POST` → `202` (akce ve frontě) — beze změny;
2. worker frontu vyzvedne, proces běží (nejvýš do timeoutu) — beze změny;
3. worker zapíše běh (`Store.AppendRun`) → **nově** událost
   `run.recorded` s kartou a zapsaným během;
4. detail, který na běh čeká, ho přidá do **Recent runs** a čekání ukončí.

## Rozsah

- **Uvnitř**:
  - `internal/events`: broker rozesílá dva druhy událostí (změna statusu
    a zapsaný běh). Stávající chování `status.changed` se nemění. Broker
    dál nikdy neblokuje producenta. Pomalý odběratel ztratí nejstarší
    události, což u běhů znamená, že klient o běhu nemusí dostat zprávu.
    Pokrývá to záložní mechanismus níže;
  - `internal/config`: háček `Store.OnRunAppended` (obdoba
    `OnStatusChange`), volaný po úspěšném `AppendRun` mimo zámek storu;
    napojení na broker v `cmd/marionette` (kompoziční kořen, ADR-0008
    doplněk bloku 0034);
  - `internal/server` (`/api/events`): událost
    `event: run.recorded`, `data: {"cardId": "…", "run": {…}}`, kde `run`
    má stejný tvar jako položka `GET /api/cards/{id}/runs`;
  - týká se jen **primární akce**. Status kontroly už mají
    `status.changed` (jen při změně stavu) a detail načítá poslední kontrolu
    podle ní;
  - `web/src/api.ts` + `useStatusEvents`: posluchač `run.recorded`
    (kontrola tvaru jako u `status.changed`);
  - `useCardActivity.waitForNewRun`: kromě dotazů reaguje na událost pro
    svou kartu. Běh novější než výchozí bod ho hned ukončí s `found` a
    zařadí se na začátek seznamu. Periodické dotazy zůstávají jen jako
    záloha:
    - když SSE není připojené (`EventSource` nedostupný nebo odpojený),
      dotazy běží jako dnes;
    - když je SSE připojené, první dotaz přijde až po uplynutí času akce
      (`timeoutSec`) a pak v dnešním intervalu až do rozpočtu. Pokryje
      ztracenou událost i běh zapsaný během výpadku spojení;
    - po znovupřipojení SSE se běhy jednou načtou přes REST (ADR-0008:
      reconnect synchronizace přes REST);
  - **dashboard** (`HomeView`, `ActionCard`): `run.recorded` pro kartu na
    dashboardu, ať akci spustil kdokoli, se promítne do řádku poznámky karty
    (`ActionNote`, blok 0049), protože výsledek běhu badge neukazuje:
    - neúspěch zůstane viditelný, dokud se na kartě nespustí další akce
      nebo kontrola, nepřijde novější běh nebo se stránka neobnoví: **Action
      failed · exit 2**, **Action timed out**, **Action canceled**. Chybu
      běhu nesmí operátor přehlédnout, proto nezmizí po 4 s jako
      ostatní výsledky;
    - úspěch u karty **bez status akce** krátce ukáže **Action finished**
      (nahradí **Accepted**); u karty se status akcí je úspěch jen
      oznámen čtečce, protože stav ukáže badge po kontrole;
    - na výšku karty to nemá vliv (řádek je rezervovaný, blok 0049);
  - UX specifikace §4 a §5.2 (poznámka karty), §6 (Recent runs), ADR-0008 doplněk, tabulka událostí
    v `docs/architecture/overview.md`;
  - testy: Go (háček se volá jen po úspěšném zápisu a s kopií běhu;
    broker doručí oba druhy; SSE handler zapíše `run.recorded` ve správném
    formátu; queue → store → broker v integračním testu s falešným
    runnerem), Vitest (událost ukončí čekání bez dalšího `getRuns`;
    událost pro jinou kartu nebo starší běh se ignoruje; bez SSE dotazy
    jako dnes; ztracená událost → záložní dotaz najde běh), E2E
    (`card-lifecycle`: běh se v detailu objeví dřív než v zálohovém
    intervalu; na dashboardu karta s padající akcí ukáže **Action failed ·
    exit N** a karta bez status akce **Action finished**; stávající
    scénáře beze změny). Vitest `HomeView`: neúspěch trvá do další akce,
    úspěch bez status akce je krátký, se status akcí tichý.
- **Mimo rozsah**:
  - událost při zařazení nebo startu akce (`queued`, `started`): stav
    požadavku dnes ukazuje UI podle `202` a status badge; přidat, až bude
    potřeba skutečný stav „běží“ z workeru;
  - historie událostí a `Last-Event-ID` replay (ADR-0008: MVP bez event
    logu).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem včetně výsledku běhu na dashboardu, který byl původně mimo rozsah („0058 zahrň rovnou vše“). Doplněk ADR-0008 přijat.

## Návrh řešení

Název `run.recorded`, ne `action.finished`: událost vzniká, až je běh
zapsaný v historii, takže ji klient může rovnou zobrazit a nemusí nic
dočítat. Běh, který se nezapíše (akce přerušená při vypnutí služby, chyba
runneru), událost nevytvoří, a proto zůstává časový rozpočet čekání
z bloku 0050.

Zálohové dotazy při připojeném SSE začínají až po `timeoutSec`. Dřív
nemá smysl se ptát, protože událost by v té době přišla, pokud běh skončil.
Na slabém hardwaru tak ubydou dotazy během běhu akce, což je hlavní přínos
bloku vedle rychlejšího zobrazení.

## Testovací plán

- `go test -race ./internal/...`.
- `cd web && npx vitest run`.
- `make e2e` a `npx playwright test e2e/card-lifecycle.e2e.ts
  --repeat-each 5` (stabilita jako u bloku 0050).
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- při připojeném SSE se nový běh objeví v detailu do 1 s od zápisu
  na serveru bez opakovaných dotazů na `/runs` během běhu akce;
- bez SSE nebo při ztracené události se běh objeví nejpozději zálohovým
  dotazem, jako po bloku 0050.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**: `make verify` prošel (202 testů Vitest, `go test -race`,
  lint, `install_test`, `release_version_test`); `make e2e` prošel
  (20 scénářů); `npx playwright test e2e/card-lifecycle.e2e.ts
  --repeat-each 5` prošel (41/41).
  Nové testy: Go `TestStoreRunAppendedCallbackRunsOutsideTheLock`
  (háček po úspěšném zápisu, čtení storu z háčku bez deadlocku, žádný
  háček pro neznámou kartu), `TestBrokerPublishesRunsInTheSameSequence`
  (společná řada `id`, kopie běhu), `TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents`
  rozšířen o `run.recorded` (formát `id`/`event`/`data`),
  `TestFinishedPrimaryActionIsPublishedAsRecordedRun` (fronta → store →
  broker); Vitest `api.spec.ts` (parsování `run.recorded`, zahození
  poškozených), `useStatusEvents.spec.ts` (rozeslání `onRun`),
  `CardDetailView.spec.ts` (událost ukončí čekání bez dotazů během běhu;
  událost jiné karty se ignoruje a záloha čte po timeoutu akce; výpadek
  streamu posune první dotaz; znovupřipojení načte běhy; běh z jiného
  zařízení se přidá), `HomeView.spec.ts` (neúspěch trvá do další akce;
  timeout a canceled; úspěch krátce, tichý jen s následnou kontrolou;
  běh zapsaný před odpovědí `202` nepřepíše **Accepted**; neznámá karta);
  E2E „reports a failed run on the dashboard and shows the run in the
  detail at once“ (Action failed · exit 3 i po 4,5 s; běh v detailu do
  3 s, tedy dřív než první záložní dotaz v 5 s).
- **Odchylky od návrhu**:
  - detail po události běh nevkládá do seznamu sám, ale jednou načte
    `/runs`: seznam tak přesně odpovídá serveru včetně ořezu na N běhů,
    a přitom během běhu akce odpadly opakované dotazy;
  - bez výchozího bodu (seznam se načítal nebo selhal) čekání ukončí
    jakýkoli `run.recorded` karty, protože REST čtení nový běh rozeznat
    neumí;
  - dashboard: úspěch je „tichý“ jen tam, kde po akci opravdu následuje
    kontrola (`expectsFollowUpCheck`), ne u každé karty se status akcí.
    Karta se status akcí bez automatických kontrol by jinak o výsledku
    nic neukázala (nález z E2E);
  - dashboard: běh zapsaný ještě před odpovědí `202` (rychlé příkazy) má
    přednost před výsledkem požadavku (**Accepted**/**Updated**);
  - upřesnění FR-42a a ADR-0008: nezapíše se akce zrušená při vypnutí
    služby, než dostala místo ke spuštění, a běh karty smazané během běhu.
    Běh přerušený za chodu se zapíše jako `canceled`;
  - `StatusListener.onStatus` je nově volitelný (posluchač může odebírat
    jen běhy nebo jen stav připojení).
- **Dokumentace aktualizována**: ADR-0008 (doplněk 2026-09-27),
  requirements FR-42a, UX specifikace §4 a §6, `docs/architecture/overview.md`
  (tabulka událostí SSE, kompozice, broker).
- **Opravy po code review (2026-09-27)**:
  - dashboard bral jakýkoli běh zapsaný během rozpracovaného požadavku za
    výsledek tohoto požadavku a zahodil vlastní výsledek požadavku. Běh
    spuštěný jinde tak během **Check status** skryl **Result not available
    yet** a při chybě zařazení skryl i chybovou hlášku. Běh se teď
    přiřazuje jen k rozpracované **primární** akci. Chyba zařazení se
    ukáže vždy. **Result not available yet** ustoupí jen neúspěšnému běhu
    (`keepsRunNote`). Nové testy v `HomeView.spec.ts` (kontrola stavu
    s cizím během, chyba zařazení se zapsaným během, neúspěšný běh proti
    vypršení čekání); první dva na původní logice selžou;
  - detail: když událost načetla nový běh ještě před odpovědí `202`,
    čekání zbytečně dotazovalo až do rozpočtu. Teď na začátku zkontroluje
    už načtený seznam a skončí hned (test v `CardDetailView.spec.ts`);
  - ověření: `make verify` (206 testů Vitest) a `make e2e` (20 scénářů)
    prošly.

