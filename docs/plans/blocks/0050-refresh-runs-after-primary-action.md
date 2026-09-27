# Implementační blok: Běh primární akce se objeví v detailu bez obnovení stránky

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-21, FR-17, FR-27
- **Vazba na ADR**: ADR-0008 (SSE se nemění); nové ADR není potřeba
- **Stav**: Hotovo
- **Závislosti**: Bloky 0029, 0039 (průběh akce), 0049 (zpětná vazba
  v panelu Actions)

## Cíl bloku

Po **Run action** v detailu karty se běh v sekci **Recent runs** často
neobjeví. Detail načte běhy jednou: hned po **Action accepted** (karta bez
následné kontroly), nebo jakmile je vidět novější status kontrola. Primární
akce v tu chvíli ale často ještě běží nebo čeká ve frontě, takže seznam
zůstane beze změny, dokud uživatel stránku neobnoví. Nález pochází
z nestabilního E2E scénáře (blok 0049, uzavření).

Po dokončení bloku detail počká na nový běh a zobrazí ho, jakmile ho server
zapíše.

## Rozsah

- **Uvnitř**:
  - `CardDetailView` / `useCardActivity`: po přijetí primární akce
    (`202`) se běhy načítají znovu v intervalu **zrychleného pollingu
    karty** (`fastPollingIntervalSeconds`), a pokud ho karta nemá
    nastavený, každé **2 s**, dokud se neobjeví běh
    novější než nejnovější běh známý před kliknutím, nebo dokud nevyprší
    rozpočet `primary.timeoutSec + 30 s` (rezerva na frontu, stejná jako
    u ruční kontroly, UX spec §4). Porovnává se `startedAt` nového běhu se
    `startedAt` dosud nejnovějšího běhu ze serveru, ne s hodinami
    prohlížeče (odolné vůči posunu času);
  - platí pro kartu bez následné kontroly i pro kartu, kde kontrola
    proběhne dřív, než primární akce skončí; status historie se dál načítá
    podle dnešních pravidel;
  - čekání na běh neblokuje tlačítka (ta se uvolní podle dnešní logiky
    stavu požadavku) a skončí při odchodu ze stránky, změně karty nebo
    novém spuštění akce;
  - chyba čtení běhů během čekání se zobrazí jako dnes
    (**Run history is unavailable**) a čekání pokračuje;
  - UX specifikace §6 (Actions, Recent runs);
  - testy: Vitest `CardDetailView` (běh přibude až po několika dotazech;
    rozpočet vyprší; odchod ze stránky čekání ukončí) a E2E scénář
    `creates, runs, checks and deletes a card` bez nestability.
- **Mimo rozsah**:
  - nová SSE událost pro dokončení běhu (změna kontraktu ADR-0008; zvážit
    až při potřebě živé historie i na dashboardu);
  - historie běhů na dashboardu (dashboard ji nezobrazuje);
  - změny backendu nebo API.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Vlastník určil, že interval dotazů na běhy
  není pevných 2 s, ale zrychlený polling interval karty
  (`fastPollingIntervalSeconds`); 2 s jen jako výchozí hodnota, když
  karta parametr nemá.

## Návrh řešení

- `useCardActivity` dostane `waitForNewRun({ after, signal, maxWaitMs })`:
  cyklus `getRuns` s intervalem `intervalMs`, výsledek `found | timeout |
  aborted`; při `found` nastaví `runs`. `after` je `startedAt` prvního
  (nejnovějšího) běhu v `runs` před odesláním požadavku (`undefined` =
  zatím žádný běh).
- `CardDetailView.runAction` pro `primary` po `202` spustí čekání na běh
  souběžně s čekáním na status (`requestAction`) a sdílí s ním
  `AbortController`, aby ho `loadDetail` i `onBeforeUnmount` ukončily.
- Interval: `card.fastPollingIntervalSeconds * 1000`, jinak výchozí 2 s
  (`runPollIntervalMs(card)`); rezerva na frontu (30 s) se převezme
  z `useCardStatus`, aby byla definovaná jednou.

## Testovací plán

- Vitest s falešnými časovači: první dva `getRuns` vrátí starý seznam,
  třetí nový běh → běh je vidět a dotazy skončí; bez nového běhu dotazy
  skončí po rozpočtu; odchod ze stránky a nové spuštění akce předchozí
  čekání zruší; chyba jednoho dotazu ukáže chybu a čekání pokračuje.
- `make e2e` opakovaně (`--repeat-each 5` pro `card-lifecycle`).
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- po **Run action** v detailu se nový běh objeví v **Recent runs** nejvýš
  jeden interval (zrychlený polling karty, výchozí 2 s) od jeho zápisu na
  serveru, bez obnovení stránky.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**: `make verify` prošel (174 testů Vitest, `install_test`,
  `release_version_test`, golangci-lint, eslint, vue-tsc, prettier,
  `go test -race`); `make e2e` prošel (17 scénářů);
  `npx playwright test e2e/card-lifecycle.e2e.ts --repeat-each 5` prošel
  (30/30) — nestabilita z bloku 0049 se neopakovala.
- **Implementace**: `useCardActivity.waitForNewRun(card, previousStartedAt)`
  (dotazy v intervalu `runPollIntervalMs(card)`, termín
  `runWaitBudgetMs(card)` jako samostatný časovač, zrušení při novém
  čekání, `reset()` a odchodu ze stránky), `requestAction` s callbackem
  `onAccepted`, `CardDetailView` po výsledku akce načítá jen status
  historii (běhy obstará čekání, takže pomalé starší čtení nepřepíše nový
  seznam). Nové testy s falešnými časovači v `CardDetailView.spec.ts`
  (interval 2 s, interval zrychleného pollingu 10 s, rozpočet, chyba
  čtení, odchod ze stránky).
- **Odchylky od návrhu**: čekání vlastní `useCardActivity` (ne
  `CardDetailView`), aby view zůstalo pod 300 řádků; po dokončení status
  kontroly se znovu načítá jen historie, ne běhy.
- **Dokumentace aktualizována**: ano — UX specifikace §6, roadmapa.
