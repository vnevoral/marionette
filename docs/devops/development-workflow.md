# Vývojový workflow

Tento dokument je jediný normativní popis procesu plánování a implementace v
projektu Marionette. `AGENTS.md`, roadmapa, implementační šablony, DevOps
kontroly a prompty v `.github/prompts/` tento proces používají a nesmí zavádět
paralelní metodiku.

## Životní cyklus

Každá nová funkce nebo změna prochází těmito kroky:

1. **Požadavek** — popiš problém, uživatelský dopad a hranice změny.
2. **Requirement** — zapiš nebo uprav testovatelný FR/NFR v
   `docs/requirements/requirements.md`.
3. **ADR** — pokud změna ovlivňuje architekturu, perzistenci, veřejný protokol,
   bezpečnost nebo technologii, připrav ADR. Implementace na něm může stavět až
   po stavu `Přijato`.
4. **Roadmapa** — zařaď práci do fáze a urč pořadí závislostí.
5. **Implementační blok** — rozlož práci na malý, samostatně ověřitelný blok
   podle `docs/plans/template-implementation-block.md`.
6. **Schválení** — blok musí být ve stavu `Schváleno`; samotný stav `Návrh`
   neopravňuje k implementaci.
7. **Implementace** — změň pouze schválený rozsah, přidej testy a proveď
   ověření podle bloku a Definition of Done.
8. **Uzavření** — aktualizuj stav bloku a roadmapy, dokumentaci, ADR/requirements
   při odchylce a uveď provedené kontroly.

Plánovací kroky (`/new-requirement`, `/new-adr`, `/plan-block`) nemění zdrojový
kód ani nespouštějí implementaci. Implementační krok (`/implement-block`) nesmí
začít bez schváleného bloku.

## Stavy a přechody

- `Návrh` — artefakt je připraven k připomínkám, není implementovatelný.
- `Schváleno` — rozsah, vazby a testovací plán byly potvrzeny.
- `Probíhá` — implementace schváleného bloku začala.
- `Hotovo` — Definition of Done je splněna a ověření je zaznamenané.
- `Zablokováno` — práce nemůže pokračovat kvůli konkrétní překážce.
- `Zamítnuto` — návrh se nebude realizovat; důvod zůstává v dokumentaci.

Povolený běžný přechod je `Návrh` → `Schváleno` → `Probíhá` → `Hotovo`.
Přechod do `Zablokováno` nebo `Zamítnuto` musí uvést důvod. Změna
schváleného rozsahu se nejdříve promítne do requirementu, ADR nebo nového
implementačního bloku; nepřidává se neohlášeně během implementace.

## Brány kvality

Před implementací ověř:

- requirement pokrývá požadované chování;
- související ADR je `Přijato`, pokud je potřeba;
- blok má jasný cíl, rozsah včetně out-of-scope, návrh řešení, testovací plán,
  závislosti a kritérium hotovosti;
- blok je uvedený v roadmapě a má stav `Schváleno`.

Před uzavřením ověř:

- implementace odpovídá schválenému rozsahu;
- testy pokrývají hlavní i chybové scénáře;
- `make verify` prošel (jediná definice validační sady pro Go i web, viz
  `Makefile`);
- veřejné chování a dokumentace odpovídají skutečnosti;
- roadmapa a stav bloku jsou aktualizované.

Podrobný uzavírací seznam je v
[Definition of Done](definition-of-done.md). Pokud kontrola selže, blok není
`Hotovo`.

## Odchylky a urgentní opravy

Když implementace odhalí neřešitelný rozpor nebo potřebu mimo rozsah, zastav
změnu, popiš dopad a založ nový requirement, ADR nebo blok. U urgentní opravy
je dovoleno provést minimální zásah před úplným plánováním pouze kvůli obnově
funkčnosti nebo bezpečnosti; bezprostředně poté se doplní chybějící artefakty,
testy a odkaz na důvod výjimky.

Historické ADR, requirements a dokončené bloky se nemění kvůli nové metodice
bez zachování historie. Opravují se pouze faktické rozpory, odkazy a stavové
údaje potřebné pro konzistenci.
