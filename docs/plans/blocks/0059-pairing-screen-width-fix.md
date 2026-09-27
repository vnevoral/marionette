# Implementační blok: Obrazovka párování na 320 px s náhradním fontem

- **Fáze**: 8 — Zpevnění (oprava po uzavření fáze)
- **Vazba na požadavky**: FR-29, FR-06
- **Vazba na ADR**: nové ADR není potřeba
- **Stav**: Hotovo
- **Závislosti**: Bloky 0043/0044 (obrazovka párování), 0057 (release z CI)

## Cíl bloku

První release z CI (tag `v1.2.0`, 2026-09-27) se nezveřejnil. V jobu
`e2e` selhal scénář „at 320 px › the pairing screen and a shown code fit
the screen“: obrazovka `/pair` byla o 2 px širší než 320 px. Na runneru
GitHubu se jako náhradní font použije DejaVu Sans / DejaVu Sans Mono, které
jsou širší než fonty v devcontaineru, kde test procházel. Po opravě se
obrazovka párování vejde na 320 px nezávisle na náhradním fontu.

## Rozsah

- **Uvnitř**: `PairView.vue`: gridy `.pair-panel`, `.pair-form` a
  `.pair-field` mají `grid-template-columns: minmax(0, 1fr)`, takže
  vnitřní šířka pole s kódem (neproporcionální písmo 1.25rem s prostrkáním)
  sloupec nerozšíří.
- **Mimo rozsah**: přidání fontů do aplikace nebo CI; změna stávajícího
  E2E testu (test chybu správně odhalil).

## Schválení

- **Schválil**: projektový vlastník (oprava selhání release workflow, které
  vlastník předal k řešení 2026-09-27)
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Oprava chyby bez změny chování; blok
  zapsán zpětně spolu s implementací.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**:
  - Chyba reprodukována lokálně po instalaci fontů DejaVu
    (`fc-match sans-serif` → DejaVu Sans): `/pair` přetékal o 2 px.
    Širší než 320 px byly formulář, popisky a obě pole, všechny se
    šířkou 281 px. Stejné přetečení bylo i s `AppShell` před blokem 0056,
    chyba tedy existovala už dřív a projevila se až s fonty runneru.
  - Po opravě je přetečení 0 px na `/`, `/devices` (i se zobrazeným
    kódem), `/cards/new/edit` a `/pair`.
  - `make e2e` s fonty DejaVu prošel (20 scénářů); `make verify` prošel
    (206 testů Vitest).
  - Poznámka: jeden běh `make verify` na silně vytíženém stroji (load
    average kolem 20) selhal v časovém testu
    `TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory` (3,45 s
    proti limitu 3 s). Samostatně pětkrát a v dalším plném běhu prošel.
    Nesouvisí s touto opravou.
- **Odchylky od návrhu**: žádné.
- **Dokumentace aktualizována**: roadmapa.
