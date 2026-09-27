# Implementační blok: Barevný proužek karty

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze), oblast UI
- **Vazba na požadavky**: FR-10a, FR-10, FR-20, FR-25, FR-28, FR-29
- **Vazba na ADR**: ADR-0004 (doménový model karty se rozšíří o volitelné
  pole, ukládání se nemění). Nové ADR není potřeba: nemění se architektura,
  jen nové nepovinné pole karty
- **Stav**: Hotovo
- **Závislosti**: Bloky 0019/0020 a 0033 (design tokeny, karta na
  dashboardu), 0049 (stabilní výška karty)

## Cíl bloku

Na dashboardu s více kartami se karty rozlišují jen ikonou a názvem. Po
dokončení bloku jde každé kartě v editaci přiřadit barvu z pevné palety
a karta na dashboardu ji ukáže jako tlustou barevnou horní hranu.
Karty tak jde seskupit podle smyslu (např. PC, síť, servery) a
najít je očima rychleji.

## Rozsah

- **Uvnitř**:
  - `internal/config`: pole `ActionCard.Color string` (`json:"color,omitempty"`)
    s názvem barvy z palety (ne hex hodnotou), aby šlo odstín upravit
    v UI bez migrace konfigurace. Validace v `Validate`: prázdné nebo
    jeden z názvů v `CardColors` → jinak chyba pole `color` (API `422`).
    Prázdná hodnota znamená bez barvy. Chybějící pole ve starší konfiguraci
    znamená také bez barvy;
  - API karet (`GET`/`POST`/`PUT`) pole přenáší beze změny kontraktu
    ostatních polí; tabulka polí karty v `docs/architecture/overview.md`;
  - paleta: 8 možností, viz návrh řešení. Hex hodnoty jako tokeny
    `--card-color-<name>` v `web/src/styles/tokens.css`. Seznam názvů je
    v UI (`web/src/ui/cardColors.ts`) i v Go a test hlídá, že se shodují;
  - `ActionCard.vue` (dashboard): karta s barvou má horní hranu 6 px
    v dané barvě, bez barvy vypadá jako dnes. Proužek je dekorativní
    a nemění výšku karty ani rozložení mřížky (FR-29, blok 0049): kreslí se
    jako `box-shadow: inset` a barva horního okraje, které žádné místo
    v layoutu neberou;
  - `CardIdentityFields.vue`: pole **Color** (`Select` se vzorkem barvy
    a textovým názvem, stejný vzor jako výběr ikony, FR-28), výchozí
    **None**. `cardEditModel` přenese barvu do formuláře a zpět,
    chyba serveru `color` se zobrazí u pole;
  - UX specifikace §5.2 (anatomie karty), §7.2 (Card identity) a §8.2
    (tokeny palety a pravidlo, že barva karty nenese stav);
  - `deploy/dev-fixture.json`: jedna karta s barvou;
  - testy: Go (`Validate` přijme prázdnou hodnotu a všechny názvy
    z palety, odmítne neznámou hodnotu a hex; JSON bez pole se načte;
    `omitempty` při uložení), Vitest (`ActionCard` s barvou a bez ní,
    `cardEditModel` round-trip barvy, `CardEditView` uloží vybranou barvu
    a zobrazí chybu `color`, shoda palety UI s Go přes sdílený seznam
    v testu), E2E (`card-lifecycle`: nastavení barvy v editoru, proužek
    je na dashboardu vidět i po obnovení stránky; karta s barvou má stejnou
    výšku jako stejná karta bez barvy).
- **Mimo rozsah**:
  - libovolná barva (color picker, hex vstup);
  - barva v detailu karty a v dalších obrazovkách (lze doplnit později);
  - filtrování nebo řazení dashboardu podle barvy;
  - volba umístění proužku per karta (umístění je jednotné pro všechny
    karty, aby mřížka působila konzistentně).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („souhlas“): horní hrana,
  paleta podle návrhu (bez červené a zelené). Vlastník požaduje jednotnou
  výšku karet s proužkem i bez něj, doplněno do návrhu a kritéria hotovosti.

## Návrh řešení

**Umístění**: tlustá horní hrana (6 px) napříč celou šířkou karty, se
zaoblením podle rohů karty. Proč ne bok: horní hrana navazuje na horní
pruh karty (ikona + status badge), na 320 px nebere šířku obsahu a
v mřížce o více sloupcích je barva vidět i při rychlém pohledu přes řádek.

**Jednotná výška**: karta má dnes horní okraj 1 px. U karty s barvou se
obarví a doplní vnitřním stínem `inset 0 5px 0` ve stejné barvě, dohromady
6 px. Okraj ani stín nemění rozměry boxu, takže karta s barvou i bez ní má
stejnou výšku i odsazení obsahu.
Boční proužek je technicky stejně jednoduchý. Pokud ho vlastník preferuje,
změní se jen CSS.

**Paleta (návrh)**:

| Název (UI) | Hodnota v JSON | Odstín |
|---|---|---|
| None | *(prázdné)* | bez proužku (bílá/průhledná) |
| Black | `black` | `#26332f` (barva textu) |
| Blue | `blue` | `#3b6fb6` |
| Teal | `teal` | `#2a9d8f` |
| Purple | `purple` | `#7b5ea7` |
| Pink | `pink` | `#c95b8f` |
| Orange | `orange` | `#e07b39` |
| Yellow | `yellow` | `#e0b53a` |

Červená a zelená v paletě záměrně nejsou. Na dashboardu už znamenají stav
**Problem** a **Healthy** (FR-25), takže červený proužek na zdravé kartě
by mátl. Oranžová a žlutá jsou sytější a teplejší než stavová `warning`
(`#ad7e3f`) a proužek je na jiném místě než badge s textem. Přesné odstíny
se doladí při implementaci podle kontrastu vůči `--color-canvas` a povrchu
karty.

**Kompatibilita a verze**: pole je nepovinné, takže konfigurace
z v1.0.0 se načte beze změny. Starší verze neznámé pole při načtení
ignoruje, ale při nejbližším uložení konfigurace ho zahodí. Po rollbacku
na v1.0.0 se tak barvy karet ztratí, karty samotné zůstanou. Jde
o zpětně kompatibilní novou funkčnost, tedy verzi MINOR (`v1.1.0`).

## Testovací plán

- `go test -race ./internal/config ./internal/server`.
- `cd web && npx vitest run`.
- `make e2e` (rozšíření `card-lifecycle.e2e.ts`).
- Ruční kontrola dashboardu na 320 px, tabletu a desktopu s kartami
  s barvou i bez ní (stejná výška karet, žádný posun layoutu).
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- barva vybraná v editoru se uloží do konfigurace, přežije restart služby
  a na dashboardu je vidět jako proužek; karta bez barvy vypadá jako
  před blokem;
- karty s barvou a bez ní mají stejnou výšku (ověřeno E2E porovnáním
  výšky dvou karet se stejným obsahem).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**: `make verify` prošel (183 testů Vitest, `go test -race`,
  golangci-lint, eslint, vue-tsc, prettier, `install_test`,
  `release_version_test`); `make e2e` prošel (18 scénářů včetně nového
  „colours a card with a stripe that keeps the card height“: barva
  vybraná v editoru, proužek po obnovení dashboardu, stejná výška karty
  s barvou a bez ní i stejné odsazení nadpisu). Vizuální kontrola
  dashboardu se všemi 8 možnostmi a rozbaleného výběru v editoru
  (snímky z Playwright, 1280 px).
  Nové testy: Go `TestActionCardValidateLimits` (prázdná, známá,
  neznámá, hex a velká písmena), `TestSaveFileRoundTripKeepsCardColor`
  (barva přežije uložení a načtení, karta bez barvy se zapíše bez klíče),
  `TestCardColorsMatchTheUIPalette` (seznam v Go = seznam v UI),
  `TestValidateEssentialIgnoresLimits` (neznámá barva ze souboru se
  načte), `TestRouterReturnsValidationFields` (`422` s `fields.color`);
  Vitest `ActionCard` (proužek jen pro barvu z palety),
  `cardEditModel` (fingerprint, mapování chyby `color`),
  `CardEditView` (uložená barva se zobrazí, nová se uloží, chyba `422`
  u pole).
- **Odchylky od návrhu**:
  - karta nese atribut `data-color` kvůli testům a ladění stylu;
  - neznámá barva načtená ze souboru (ručně upravená konfigurace) se
    na dashboardu nezobrazí a editor ukáže **None**; API ji při uložení
    odmítne až po změně, stejně jako jiné hodnoty mimo aktuální limity;
  - formulář drží „bez barvy“ jako prázdný řetězec, aby načtení jiné
    karty nepřevzalo barvu předchozí.
- **Dokumentace aktualizována**: UX specifikace §5.2, §7.2, §8.2; tabulka
  limitů polí v `docs/architecture/overview.md`; `deploy/dev-fixture.json`.
