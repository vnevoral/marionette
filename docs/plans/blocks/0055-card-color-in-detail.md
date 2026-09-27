# Implementační blok: Barva karty v detailu

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze), oblast UI
- **Vazba na požadavky**: FR-10a, FR-25, FR-29
- **Vazba na ADR**: nové ADR není potřeba
- **Stav**: Hotovo
- **Závislosti**: Blok 0054 (barva karty, `cardColors.ts`, tokeny)

## Cíl bloku

Barvu karty dnes ukazuje jen dashboard. Z dashboardu se ale jedním klikem
přechází do detailu, a bez barvy tam operátor ztrácí vodítko, na jaké kartě
je. Po dokončení bloku detail karty zobrazí stejnou barvu jako dashboard.

## Rozsah

- **Uvnitř**:
  - `CardDetailView`: dlaždice ikony v hlavičce (`detail-icon`, slot
    `identity` v `PageHeader`) dostane u karty s barvou proužek 4 px
    v barvě karty podél horní hrany dlaždice. Použije se stejná technika
    jako na dashboardu (`box-shadow: inset` a barva okraje), takže se
    rozměry hlavičky nezmění. Karta bez barvy nebo s barvou mimo paletu
    vypadá jako dnes (`cardColorValue`);
  - aby `CardDetailView.vue` zůstal pod 300 řádky (dnes 298), vyčlení se
    dlaždice ikony do malé komponenty `CardIdentityTile.vue` (ikona + barva).
    Dashboard ji nepoužívá, protože tam proužek patří celé kartě;
  - UX specifikace §6 (hlavička detailu);
  - testy: Vitest (`CardIdentityTile` s barvou, bez barvy a s neznámou
    barvou; `CardDetailView` předá barvu karty), E2E (detail karty s barvou
    má proužek; rozšíření scénáře z bloku 0054).
- **Mimo rozsah**:
  - barva na stránce úprav karty (tam je vidět ve výběru **Color**);
  - barevné pozadí nebo orámování celé hlavičky detailu (hlavička není
    karta a barevná plocha by soupeřila se stavovým badge).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („souhlas se vším“): proužek na dlaždici ikony podle návrhu.

## Návrh řešení

Proužek na dlaždici ikony, ne na celé hlavičce: dlaždice je jediný
„kartový“ prvek v hlavičce detailu, barva se tak váže ke stejnému místu,
kde je ikona, a nezmění rozložení na 320 px. Alternativa, levý okraj celé
hlavičky, je stejně jednoduchá, pokud ji vlastník preferuje.

## Testovací plán

- `cd web && npx vitest run`.
- `make e2e`.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- detail karty s barvou ukazuje stejnou barvu jako dashboard; detail karty
  bez barvy se nezměnil.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**: `make verify` prošel (202 testů Vitest, `go test -race`,
  lint, `install_test`, `release_version_test`); `make e2e` prošel
  (20 scénářů). Nové testy: Vitest `CardIdentityTile.spec.ts` (proužek
  pro barvu z palety; bez barvy, prázdná a neznámá barva bez proužku,
  výchozí ikona), `CardDetailView` (hlavička předá ikonu a barvu karty);
  E2E scénář barvy z bloku 0054 rozšířen o detail (`.identity-tile`
  s `data-color`).
- **Odchylky od návrhu**: žádné. Komponenta `CardIdentityTile.vue`
  převzala i styl dlaždice; `CardDetailView.vue` má 285 řádků.
- **Dokumentace aktualizována**: UX specifikace §6 (hlavička detailu).
