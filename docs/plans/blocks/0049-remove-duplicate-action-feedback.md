# Implementační blok: Bez duplicitní zpětné vazby akce a skákání karty

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-21, FR-27, FR-29, NFR-08, NFR-09
- **Vazba na ADR**: ADR-0007 (UX a design systém); nové ADR není potřeba
- **Stav**: Hotovo
- **Závislosti**: Bloky 0029, 0037, 0039 (průběh akce a jeho výsledek),
  0040 (poznámka „Status could not be refreshed“), 0042 (kanály zpětné
  vazby), 0041 (E2E)

## Cíl bloku

Po spuštění akce karta na dashboardu ukazuje v badge **Queued** /
**Running** a pod popisem řádek zpětné vazby se stejným textem; po
dokončení tam asi 4 s visí **Updated**, přestože badge už ukazuje nový
stav. Řádek vzniká a zaniká, takže karta mění výšku a v mřížce poskakuje
celá řada. Vlastník projektu (2026-09-26) duplicitu i skákání odmítl.

Po dokončení bloku karta i detail zobrazují jen zpětnou vazbu, která nese
informaci navíc proti badge, a to na místě, které nemění výšku karty.

## Rozsah

- **Uvnitř**:
  - **Ruší se** (text duplikuje badge): řádek s **Queued** / **Running**
    během akce a výsledek **Updated** / **Status updated** na dashboardu
    i v detailu.
  - **Zůstává** (informace, kterou badge nenese):
    - **Result not available yet** — kontrola v čekací době nedoběhla,
      badge ukazuje předchozí stav;
    - chyba zařazení akce (plná fronta 503, síť, 4xx) — pravidlo §4:
      chyba nikdy nezmizí beze stopy;
    - **Accepted** / **Action accepted**, když žádná kontrola
      nenásleduje (karta bez status akce nebo bez zrychleného pollingu) —
      jinak by uživatel neměl potvrzení, že se akce spustila.
  - **Dashboard karta** (`ActionCard.vue`): zbývající zprávy se zobrazí
    **místo** řádku `Last checked …` / `Not checked yet` na stejné
    pozici a se stejnou výškou (tón a ikona podle typu, text se ořízne na
    jeden řádek s plným textem v `title`/tooltipu); po skončení zprávy
    (asi 4 s, beze změny délky) se vrátí `Last checked …` s novým časem.
    Karta bez status akce, která dnes řádek `Last checked` nemá, má tento
    řádek rezervovaný trvale (prázdný, mimo dobu zprávy), aby se její výška
    neměnila.
  - **Detail karty** (`CardDetailView.vue`): zbývající zprávy akce se
    přesunou z řádku nad souhrnem do panelu **Actions** pod tlačítka,
    do slotu s rezervovanou výškou jednoho řádku. Řádek nad souhrnem
    zůstává jen pro mazání karty (**Deleting…** a chyba mazání).
  - **Přístupnost** (NFR-09): změna stavu požadavku se čtečkám dál
    oznamuje, a to vizuálně skrytou `aria-live="polite"` oblastí (text
    **Queued**, **Running**, **Updated** …), která nezabírá místo. Badge
    zůstává nositelem stavu pro vidící uživatele (text + ikona + barva,
    FR-25).
  - UX specifikace §4 (feedback rules, feedback channels), §5.2 (anatomie
    karty) a §6 (detail, Actions) podle tohoto bloku.
  - Testy komponent a úprava E2E (`card-lifecycle.e2e.ts` dnes čeká na
    viditelný text **Updated**).
- **Mimo rozsah**:
  - změna logiky čekání na kontrolu, časových limitů nebo `useActionRequest`
    (mění se jen prezentace);
  - toasty (**Card saved**, **Card deleted**) a zpětná vazba formulářů;
  - vzhled badge a slovník stavů.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („vše schvaluji“). Rozdělení zpráv, zobrazení místo `Last checked` a použití i v detailu karty potvrzeno 2026-09-26.

## Návrh řešení

- `ActionCard.vue`: `feedback` vrací zprávu jen pro `result`, který není
  `updated`; `pending` do viditelného textu nevstupuje. Řádek `card-meta`
  se vykreslí vždy (`Last checked …`, zpráva, nebo prázdný pro kartu bez
  status akce) s pevnou výškou jednoho řádku.
- Poznámka **Status could not be refreshed** (blok 0040) zůstává
  samostatná; pokud by i ona způsobovala skok, řeší se mimo tento blok.
- Sdílená vizuálně skrytá live oblast: malá komponenta nebo třída
  `visually-hidden` v globálních stylech (pokud už neexistuje).
- `RequestState.vue` zůstává pro ostatní použití (editace, zařízení).

## Testovací plán

- Vitest `ActionCard`: během `pending` není viditelný text **Queued** /
  **Running** mimo badge, ale je v live oblasti; `result.updated` nic
  viditelného nepřidá; **Result not available yet**, chyba a **Accepted**
  se zobrazí místo `Last checked` a po vypršení se `Last checked` vrátí;
  karta bez status akce má řádek vždy.
- Vitest `CardDetailView`: zprávy akce v panelu Actions, mazání nad
  souhrnem.
- `make e2e`: `card-lifecycle` ověří výsledek podle badge místo textu
  **Updated**; měření, že výška karty během a po akci zůstane stejná
  (`boundingBox` před / během / po).
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- spuštění akce na dashboardu nezmění výšku žádné karty v mřížce;
- žádný viditelný text zpětné vazby neopakuje to, co právě ukazuje badge.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (164 testů Vitest, `install_test`,
  golangci-lint, eslint, vue-tsc, prettier, `go test -race`); `make e2e`
  prošel (16 scénářů), `card-lifecycle` měří výšku karty před akcí, po
  **Check status** i po **Run action**.
- **Implementace**: nová komponenta `ActionNote.vue` (klidový řádek, který
  výsledek akce nahradí na místě, a vizuálně skrytá `aria-live` oblast pro
  fáze a výsledky); `RequestResult.quiet` pro **Updated** / **Status
  updated** (`outcomeResult`); `ActionCard` zobrazuje `ActionNote` místo
  `Last checked` s pevnou výškou jednoho řádku; `CardDetailView` přesunul
  výsledky akcí do panelu Actions (`useTransientResult`), řádek nad
  souhrnem zůstal jen pro mazání. Odstraněny nepoužívané texty
  `FEEDBACK.queued` a `FEEDBACK.actionQueued`.
- **Odchylky od návrhu**: klidový text panelu Actions („Actions are queued
  asynchronously…“) se přesunul do slovníku (`FEEDBACK.actionsAsync`),
  protože ho `ActionNote` zobrazuje jako fallback.
- **Nález mimo rozsah**: scénář `creates, runs, checks and deletes a card`
  jednou selhal, protože detail po **Action accepted** načte běhy hned,
  ještě než primární akce doběhne (sekce Recent runs zůstala prázdná).
  Opakovaný běh (3×) prošel; chování je původní z bloku 0039 a tento blok
  ho nemění. Návrh: nový blok, který po přijetí akce bez následné kontroly
  načte běhy znovu po jejím dokončení (např. po `timeoutSec` nebo přes
  událost běhu).
- **Dokumentace aktualizována**: ano — UX specifikace §4, §5.2, §6,
  roadmapa.
