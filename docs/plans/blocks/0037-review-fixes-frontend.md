# Implementační blok: Opravy z code review — frontend (čekání na kontrolu, dashboard polling, API)

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-21, FR-22, FR-26, FR-42, NFR-03, NFR-11
- **Vazba na ADR**: ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0029, 0032, 0033

## Cíl bloku

Po dokončení frontend čeká na novou kontrolu jen tehdy, když ji backend
opravdu naplánuje; dashboard v polling režimu nezahazuje výsledky ostatních
karet a načítá stavy jedním požadavkem; chyby u `args[N]` se zobrazí u pole;
timeout požadavku kryje i čtení těla; sdílené konstanty jsou na jednom místě.

## Rozsah

- **Uvnitř** (nálezy code review 2026-09-26):
  1. `useCardStatus.ts`: `expectsFollowUpCheck(card, action)` — status
     akce vždy (kontrola se zařadí přímo), primární akce jen když má karta
     status akci a `pollingIntervalSeconds`, `fastPollingIntervalSeconds`
     i `fastPollingWindowSeconds` > 0 (zrcadlí
     `Scheduler.NotifyPrimaryAction`); `waitBudgetMs(card)` vrací okno
     v ms, výchozí `defaultMaxWaitMs`. `HomeView` i `CardDetailView` bez
     kontroly hlásí **Accepted** ihned a tlačítka uvolní.
  2. `cardEditModel.ts`: cesty `primary.args[N]`/`status.args[N]` se mapují
     na chybu bloku Arguments.
  3. `api.ts`: timeout `REQUEST_TIMEOUT_MS` platí až do přečtení těla
     odpovědi.
  4. `HomeView.vue`: místo globální `statusRefreshGeneration` se výsledek
     refreshe aplikuje per karta jen tehdy, když nese novější `checkedAt`
     (`isNewerCheck`), nebo když karta ještě žádný snapshot nemá.
  5. `HomeView.vue`: `onRefresh` (polling režim) a refresh po načtení
     používají jeden `listCards()` a jeho `currentStatus` místo N ×
     `getStatus`; selhání `listCards` označí monitorované karty jako
     `statusErrors`, karty zůstanou zobrazené.
  6. Odstranit duplicitní konstanty: `resultVisibleMs` → `messageVisibleMs`
     z `useTransientMessage`, `defaultFastPollingWindowSeconds` → bod 1.
  7. UX spec §4 a §6: čekání na kontrolu jen u karet s automatickými
     kontrolami; polling režim čte seznam karet.
- **Mimo rozsah**:
  - změna backendu (např. zařazení status kontroly po primární akci u karet
    bez pollingu);
  - Toast / jiná prezentace výsledků;
  - E2E testy.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava stavu („naplanuj a oprav
  všechny nálezy“). Pravidlo pro čekání je záměrně frontendové zrcadlo
  backendu; rozšíření 202 odpovědi o příznak „kontrola naplánována“ by bylo
  čistší, ale je to změna API kontraktu mimo rozsah opravy.

## Návrh řešení

- `useCardStatus.ts`: `export function expectsFollowUpCheck(card: ActionCard,
  action: ActionKind): boolean`, `export function waitBudgetMs(card:
  ActionCard): number`.
- `HomeView.vue`: `refreshStatuses()` → `listCards()`; `applySnapshot(id,
  snapshot)` zapíše jen novější check; `onSnapshot` z `waitForNewerStatus`
  jde stejnou cestou; generace zmizí.
- `api.ts`: `clearTimeout` po `response.json()`; abort během čtení těla se
  mapuje na „Request timed out“.
- `cardEditModel.ts`: `rest[0]` začínající `args` → klíč `args`.

## Testovací plán

- `useCardStatus.spec.ts`: `expectsFollowUpCheck` pro status akci, primární
  s pollingem, primární bez pollingu, karta bez status akce; `waitBudgetMs`.
- `HomeView.spec.ts`: primární akce na kartě bez pollingu → **Accepted**
  ihned, tlačítka uvolněná, `getStatus` se nevolá; polling režim volá
  `listCards`, ne `getStatus`; starší snapshot z refreshe nepřepíše novější
  z SSE.
- `CardDetailView.spec.ts`: primární akce bez pollingu → **Accepted**.
- `cardEditModel.spec.ts`: `primary.args[2]` → `primaryArgs`.
- `api.spec.ts`: tělo, které se nedočte do `REQUEST_TIMEOUT_MS`, končí
  „Request timed out“.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `grep -n "4000\|= 120" web/src/views/*.vue` nenajde lokální konstanty;
- `HomeView.vue` neobsahuje `statusRefreshGeneration`.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel; Vitest 77 → 85 testů. Nové testy:
  `useCardStatus.spec.ts` (`supersedes`, `expectsFollowUpCheck` pro status
  akci / primární s pollingem / bez pollingu / bez status akce,
  `waitBudgetMs`), `HomeView.spec.ts` (Accepted ihned u karty bez pollingu
  a bez volání `getStatus`; polling režim volá jen `listCards` a aplikuje
  jeho `currentStatus`; pomalý refresh nepřepíše novější snapshot ze
  streamu), `CardDetailView.spec.ts` (Accepted bez čekání, tlačítka
  uvolněná), `cardEditModel.spec.ts` (`primary.args[2]` → `primaryArgs`,
  druhá zpráva se nepřepíše), `api.spec.ts` (tělo, které se nedočte do
  `REQUEST_TIMEOUT_MS`, končí „Request timed out“). Kritérium: `grep -n
  "4000\|= 120" web/src/views/*.vue` a `grep statusRefreshGeneration`
  nenajdou nic; `HomeView.vue` má 229 řádků.
- **Odchylky od návrhu**: (1) místo `isNewerCheck` (pouhá nerovnost
  `checkedAt`) rozhoduje o přepsání nová funkce `supersedes(next,
  current)` — porovnává časy `checkedAt`, takže starší snapshot z pomalého
  REST čtení nikdy nepřepíše novější ze streamu; nezkontrolovaný snapshot
  nepřepíše zkontrolovaný. Stejnou cestou jdou i SSE události. (2)
  `statusErrors` v `HomeView` byl zapisovaný, ale nikde nezobrazený stav
  (šablona ho `ActionCard` nepředává); byl odstraněn místo přepojení na
  `listCards`. Dashboard tedy chybu čtení statusu nijak nevizualizuje —
  karta dál ukazuje poslední známý stav a `Last checked …`. Pokud má UX
  spec §4 („marks its data as unknown“) platit i pro dashboard, je to
  samostatná UX úprava (`ActionCard` prop), ne oprava z review. (3) Po
  úvodním `listCards` se už neposílá druhé kolo `getStatus` — seznam už
  `currentStatus` nese.
- **Dokumentace aktualizována**: ano — UX spec §4 (čekání jen když kontrola
  následuje, polling režim čte seznam karet, snapshot se nepřepíše starším)
  a §6 (Action accepted bez čekání), roadmapa.
