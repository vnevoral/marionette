# Implementační blok: Příkaz akce jako jeden řádek v editoru

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-22a, FR-11, FR-22, FR-26, FR-27, FR-28,
  NFR-01 a
- **Vazba na ADR**: [ADR-0012](../../architecture/decisions/0012-single-line-command-editor.md)
  (musí být `Přijato`), ADR-0004, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0014, 0017, 0032 (editor karty a mapování chyb),
  0041 (E2E)

## Cíl bloku

Uživatel zadá primární i status akci jako jeden řádek, stejně jako
v terminálu (`/usr/bin/ping -c 1 -W 2 192.168.1.10`), a pod polem vidí,
jak se řádek rozloží na příkaz a argumenty. Uložený tvar akce
(`command` + `args`) a API se nemění.

## Rozsah

- **Uvnitř**:
  - čistý modul `web/src/views/commandLine.ts`:
    `parseCommandLine(line)` → `{ command, args }` nebo chyba s pozicí
    a zprávou; `formatCommandLine(command, args)` → řádek; gramatika,
    zakázané znaky a pravidla uvozování přesně podle ADR-0012;
  - `ActionEditor.vue`: pole **Command line** místo Command a
    Arguments, živý náhled rozkladu (příkaz + argumenty jako čitelné
    položky, přístupné čtečce), chyba rozkladu u pole; Working directory,
    Environment, Timeout a Output rule beze změny;
  - `cardEditModel.ts`: formulář drží řádek místo `ArgumentRow[]`;
    klientská validace (prázdný řádek, chyba rozkladu) blokuje uložení;
    serverové chyby `primary.command`, `primary.args`,
    `primary.args[N]` (a totéž pro `status`) se mapují na pole Command
    line; odstranění `ArgumentRow` a `argumentRows`, pokud nezůstane jiné
    použití;
  - slovník textů v `src/ui/vocabulary.ts` (FR-26): popisek, nápověda,
    chybové zprávy (neukončená uvozovka, shellový znak s nápovědou
    `/bin/sh -c '…'`, proměnná s odkazem na Environment);
  - UX specifikace §7.2 a scénáře v §10 podle ADR-0012;
  - E2E scénáře, které dnes vyplňují řádky argumentů.
- **Mimo rozsah**:
  - jakákoli změna API, backendu, validace na serveru nebo perzistence;
  - migrace uložené konfigurace (není potřeba);
  - expanze proměnných, `~` nebo globů; spouštění přes shell;
  - zadání Environment jako `KEY=value` na začátku řádku (proměnné zůstávají
    v samostatném poli).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („vše schvaluji“) spolu s přijetím ADR-0012. Shellové znaky mimo uvozovky jsou chyba, ne text.

## Návrh řešení

- Parser je ruční stavový automat nad znaky (stavy: mimo uvozovky,
  v `'…'`, v `"…"`, po `\`), bez regulárních výrazů na celý řádek a bez
  závislostí. Vrací první chybu s indexem znaku, aby šla zobrazit
  srozumitelná zpráva.
- `formatCommandLine` uvozuje podle ADR-0012 bod 5; pro načtení existující
  akce se volá jednou při inicializaci formuláře, během psaní se řádek
  nepřeformátovává (kurzor uživateli neskáče).
- Dirty-check formuláře porovnává rozložený tvar, ne text: přidání mezery
  navíc formulář nezašpiní.

## Testovací plán

- Jednotkové testy `commandLine.ts`: jednoduchý příkaz; více mezer
  a tabulátory; `'…'`, `"…"` s `\"` a `\\`; `\ ` mimo uvozovky; spojování
  částí (`a'b c'd`); prázdný argument `''`; neukončená uvozovka; `\` na
  konci; každý zakázaný znak mimo uvozovky i v nich; `~`, `*`, `?` jako
  text; prázdný a jen-mezerový řádek; konec řádku ve vstupu; vlastnost
  `parse(format(c, a)) = (c, a)` na sadě případů včetně argumentů s mezerou,
  `'`, `"`, `\`, `$`, `|` a prázdných.
- Testy `ActionEditor` a `cardEditModel`: náhled rozkladu, chyba u pole,
  mapování serverových chyb `args[N]` na Command line, načtení existující
  karty a uložení bez změn pošle stejné `command` + `args`.
- `make e2e`: vytvoření karty jedním řádkem, úprava existující karty
  s argumentem obsahujícím mezeru, odmítnutí `ping host | grep ttl`.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- akci `/usr/bin/ping -c 1 -W 2 192.168.1.10` lze zadat jedním řádkem
  a uloží se jako `command: /usr/bin/ping`, `args: [-c, 1, -W, 2,
  192.168.1.10]`;
- API kontrakt a backend jsou beze změny (`git diff` bez změn v `internal/`
  a `cmd/`).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (mj. 30 testů `commandLine.spec.ts`
  včetně vlastnosti `parse(format(c, a)) = (c, a)`); `make e2e` prošel
  včetně nového scénáře `edits a stored action as one command line`
  (načtení `echo 'hello from' quoted`, odmítnutí `echo hi | grep hi`,
  uložení `echo 'hello again' done` → `args: [hello again, done]`).
  `git diff` v `internal/` a `cmd/` obsahuje jen vygenerovaný
  `internal/webui/dist/index.html` z buildu UI; backend a API se nemění.
- **Implementace**: `web/src/views/commandLine.ts`
  (`parseCommandLine`, `formatCommandLine`, `quoteWord`) podle ADR-0012;
  `ActionEditor` má pole **Command line** s nápovědou, živým náhledem
  (seznam „Command and arguments“) a chybou rozkladu; `cardEditModel`
  drží řádek místo `ArgumentRow[]` (`commandLineOf`, `actionFrom`,
  `validate` přijímá stav editoru), serverové chyby `args` / `args[N]`
  patří k poli Command line; typ `ArgumentRow` odstraněn.
- **Odchylky od návrhu**:
  - popisky a nápověda pole zůstaly přímo v `ActionEditor.vue` jako
    ostatní popisky editoru; chybové zprávy rozkladu jsou v
    `commandLine.ts` vedle gramatiky, ne ve `vocabulary.ts`;
  - opravena vazba, kterou změna odhalila: zapnutá status akce, u které
    uživatel vyplnil jen příkazový řádek, by se neuložila (`form.status`
    zůstal nedefinovaný); payload, dirty-check i validace teď použijí
    `emptyAction()` jako výchozí hodnoty (test v `CardEditView.spec.ts`).
- **Dokumentace aktualizována**: ano — UX specifikace §7.2, ADR-0012
  (Přijato), roadmapa.
