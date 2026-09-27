# ADR-0012: Zadání akce jako jednoho příkazového řádku v UI

- **Stav**: Přijato
- **Datum**: 2026-09-26

## Kontext

Editor akce (primární i status) má dnes samostatné pole **Command** a
opakovatelné řádky **Arguments**, jeden řádek na argument
([UX specifikace §7.2](../../ux/ui-ux-specification.md)). Při první
konfiguraci karty WoL + ping na Raspberry Pi (2026-09-26) se ukázalo, že je
to pro uživatele velmi nepříjemné: příkaz, který zná z terminálu jako jeden
řádek (`/usr/bin/ping -c 1 -W 2 192.168.1.10`), musí ručně rozsekat do pěti
řádků. Když ho zadá do jednoho řádku, program dostane jediný argument
`"-c 1 -W 2 192.168.1.10"`, akce selže a z UI nejde poznat proč.

Model akce (`Action.Command` + `Action.Args`, ADR-0004) a bezpečnostní
pravidlo NFR-01 a (spouštění přes `exec.Command(name, args...)`, nikdy přes
shell ze stringu) jsou správné a mají zůstat.

## Rozhodnutí

1. Editor akce nahradí pole Command a řádky Arguments jedním polem
   **Command line**. Working directory, Environment, Timeout a Output rule
   zůstávají samostatná pole.
2. **Rozklad probíhá jen v UI.** API, doménový model, perzistence a
   execution engine se nemění: UI před uložením rozloží řádek na
   `command` (první slovo) a `args` (zbytek) a při načtení je zpět složí.
   Server nikdy nepřijímá ani neinterpretuje příkazový řádek jako text.
3. **Gramatika** je podmnožina POSIX shellu bez jakékoli expanze:
   - slova odděluje mezera nebo tabulátor; řádek nesmí obsahovat konec
     řádku;
   - `'…'`: vše uvnitř doslova, až po další `'`;
   - `"…"`: vše doslova kromě `\"` a `\\`;
   - mimo uvozovky `\` zruší zvláštní význam následujícího znaku;
   - sousední části bez mezery tvoří jedno slovo (`a'b c'd` → `ab cd`);
     `''` nebo `""` je prázdný argument;
   - nic se neexpanduje: proměnné, `~`, `*`, `?` ani výstup příkazů;
     `~`, `*` a `?` jsou obyčejné znaky.
4. **Shellové znaky mimo uvozovky jsou chyba.** `|`, `&`, `;`, `<`, `>`,
   `(`, `)`, `` ` `` a `$` bez uvozovek nebo `\` editor odmítne
   s vysvětlením, že Marionette příkazy spouští bez shellu, a s nápovědou
   (`/bin/sh -c '…'` pro roury, pole Environment pro proměnné). V
   uvozovkách jsou to obyčejné znaky. Chybou je i neukončená uvozovka a
   `\` na konci řádku.
5. **Zpětné složení** existující akce: slovo složené jen ze znaků
   `A–Z a–z 0–9 _ @ % + = : , . / -` se zapíše beze změny, prázdné jako
   `''`, ostatní v jednoduchých uvozovkách (`'` uvnitř jako `'\''`). Pro
   každou akci platí `parse(format(command, args)) = (command, args)`, takže
   uložení bez úprav akci nezmění.
6. Pod polem se živě zobrazuje výsledek rozkladu (příkaz a jednotlivé
   argumenty), aby uživatel před uložením viděl, co se spustí.

## Zvažované alternativy

- **Ponechat opakovatelné řádky** — zamítnuto; je to hlavní UX problém,
  kvůli kterému ADR vzniká.
- **Posílat příkazový řádek na server a rozkládat ho tam** — zamítnuto:
  rozšiřuje API a validaci na backendu, míchá textový a strukturovaný tvar
  akce a posouvá bezpečnostně citlivé parsování na stranu, kde běží příkazy.
  Rozklad v UI nechává serverový kontrakt strukturovaný.
- **Spouštět řádek přes `/bin/sh -c`** — zamítnuto; porušuje NFR-01 a a
  otevírá shell injection.
- **Rozdělit jen podle mezer, bez uvozovek** — zamítnuto; nejde zadat
  argument s mezerou (cesta, regulární výraz) a existující akce s takovým
  argumentem by nešly zobrazit.
- **Shellové znaky tiše brát jako text** — zamítnuto; `ping host | grep ttl`
  by vypadal správně, ale předal by `|` pingu jako argument a selhal by bez
  srozumitelné příčiny.

## Důsledky

- Zadání akce odpovídá tomu, co uživatel zná z terminálu; živý náhled
  rozkladu odhalí chyby dřív, než akce poprvé selže.
- Backend, API, perzistence a E2E kontrakt API zůstávají beze změny;
  existující konfigurace se nemigruje.
- NFR-01 a platí dál: server spouští strukturované `command` + `args`, nikdy
  text přes shell. Parser je čistá funkce ve frontendu a musí mít vlastní
  jednotkové testy včetně vlastnosti zpětného složení.
- Uživatel, který opravdu potřebuje rouru, ji musí zapsat explicitně přes
  `/bin/sh -c '…'` a odpovídá za obsah.
- Validační chyby serveru k `command` i `args[N]` se v UI zobrazí u jednoho
  pole Command line.
- UX specifikace §7.2 se mění: „Arguments … are repeatable rows“ platí
  nadále jen pro proměnné prostředí.

## Doplnění (2026-09-27, blok 0053)

Konec řádku je chybou jen **mimo uvozovky**; uvnitř `'…'` a `"…"` je
součástí argumentu (tak to parser dělal od začátku, bod 3 to jen neuváděl).
Akce uložená přes API nebo v souboru může mít víceřádkový argument (např.
dvouřádkový skript pro `sh -c`). Jednořádkové textové pole by konec řádku
při zobrazení tiše odstranilo a první úprava by skript změnila, proto se
takový příkazový řádek edituje ve víceřádkovém poli, které konec řádku
zachová. Pole zůstane víceřádkové až do opuštění editoru, aby se při
smazání konce řádku nevyměnilo a neztratilo fokus.
