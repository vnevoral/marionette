# Implementační blok: Přejmenování spárovaného zařízení

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze), oblast přístupu
- **Vazba na požadavky**: FR-57, FR-52, FR-55, NFR-12, NFR-13
- **Vazba na ADR**: ADR-0011. Nové ADR není potřeba: nový endpoint
  rozšiřuje správu zařízení ve stejném bezpečnostním modelu (stejná
  oprávnění jako odebrání, token ani platnost se nemění)
- **Stav**: Hotovo
- **Závislosti**: Bloky 0043, 0044 (registr zařízení, stránka Devices)

## Cíl bloku

Název zařízení se dnes zadá jen při párování (obrazovka párování nabídne
název odvozený z prohlížeče) a později nejde změnit; kdo přijal výchozí
návrh, má v seznamu třeba dvě „Chrome on Linux“. Po dokončení bloku jde
na stránce **Devices** přejmenovat kterékoli spárované zařízení.

## Rozsah

- **Uvnitř**:
  - `internal/access`: `Registry.Rename(id, name string) (Device, error)`
    — stejná validace jako `Pair` (`strings.TrimSpace`, neprázdné, nejvýš
    `MaxDeviceNameLength` znaků → `ErrInvalidName`), neznámé nebo
    vypršelé zařízení → `ErrDeviceNotFound`, okamžité uložení souboru
    zařízení (`saveLocked`) s návratem původního názvu při chybě zápisu,
    stejně jako `Remove`; token, `PairedAt` ani `LastSeenAt` se nemění;
  - `internal/server`: `PATCH /api/devices/{id}` s tělem `{"name": "…"}`
    → `200 {device}` (tvar jako v `GET /api/devices`, včetně
    `current`); neplatný název `422` s `fields.name`; neznámé zařízení
    `404`; chráněno párováním (FR-50) a cross-site ochranou
    (`requireSameOrigin`, NFR-12) jako ostatní mutující routy;
  - `web/src/api.ts`: `renameDevice(id, name)`;
  - `DevicesView`: u každého zařízení tlačítko **Rename** (ikona tužky
    s přístupným názvem „Rename <název>“), které nahradí název polem
    s tlačítky **Save** / **Cancel**; Enter uloží, Escape zruší; chyba
    `422` se zobrazí u pole; po uložení inline potvrzení
    „"<nový název>" saved“ (UX spec §4: inline, ne toast);
  - slovník (`ACCESS.rename`, …), UX specifikace §7.4, tabulka endpointů
    v `docs/architecture/overview.md`, README (sekce o zařízeních);
  - testy: Go (`Registry.Rename` — oříznutí, prázdný a příliš dlouhý
    název, neznámé zařízení, perzistence po znovunačtení, nezměněný
    token a časy; handler — 200, 422, 404, 401 bez tokenu, 403 z cizího
    originu), Vitest `DevicesView` (přejmenování, zrušení, chyba 422,
    klávesnice), E2E (přejmenování vlastního zařízení a jeho zobrazení
    po obnovení stránky).
- **Mimo rozsah**:
  - unikátnost názvů (FR-57: nemusí být unikátní);
  - omezení přejmenování jen na vlastní zařízení (FR-57: stejná
    oprávnění jako odebrání);
  - historie nebo audit změn názvu (změna se zapíše do logu služby na
    úrovni info, stejně jako párování a odebrání).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („souhlas s návrhy“).

## Návrh řešení

Viz rozsah. `PATCH` místo `PUT`, protože se mění jen jedno pole zdroje.
Zápis do logu: `renamed paired device` s `device` (id) a novým `name`;
starý název se do logu nezapisuje zbytečně dvakrát. Rozhraní na stránce
Devices používá PrimeVue `InputText` a `Button` (konvence projektu);
pole má popisek „Device name“ a stejný limit délky jako obrazovka
párování.

## Testovací plán

- `go test -race ./internal/access ./internal/server`.
- Vitest `DevicesView.spec.ts`.
- `make e2e` (rozšíření `pairing.e2e.ts`).
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- nový název je vidět na všech spárovaných zařízeních po obnovení seznamu
  a přežije restart služby.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27). `make verify` a
  `make e2e` (17 scénářů včetně nového přejmenování) prošly v závěrečném
  ověření spolu s bloky 0050 a 0051.
- **Ověření**: `go test -race ./internal/...`, `go vet ./...`,
  `gofumpt -l`, `golangci-lint run ./...` — bez chyb; ve `web/`
  `npx vitest run` (174 testů), `npx vue-tsc -b`,
  `npx eslint . --max-warnings 0`, `npx prettier --check .` — bez chyb.
  Nové testy: `internal/access` (`TestRenameKeepsTokenAndTimesAndPersists`,
  `TestRenameRejectsInvalidNamesAndUnknownDevices` včetně vypršelého
  zařízení a hranice 64 znaků, `TestRenameKeepsTheOldNameWhenSavingFails`),
  `internal/server` (`TestPairedDeviceRenamesDevices`: 200 s `current`,
  422 s `fields.name`, 404, 400 pro neznámé pole, 401 bez cookie, 403
  z cizího originu; `PATCH /api/devices/{id}` přidán do seznamu
  chráněných rout), Vitest `api.spec.ts` a `DevicesView.spec.ts`
  (přejmenování, synchronizace session u vlastního zařízení, zrušení
  tlačítkem i Escape, Enter, chyba 422 a prázdný název u pole), E2E
  `pairing.e2e.ts` (samostatný prohlížeč se přejmenuje, nový název je vidět
  po obnovení i na ostatních zařízeních, pak se odebere).
- **Odchylky od návrhu**: žádné věcné. Upřesnění: odpověď `PATCH` má tvar
  `{"device": {…}}` jako `POST /api/pairing`; řádek zařízení se
  přejmenováním vyčlenil do komponenty `web/src/components/DeviceRow.vue`,
  aby `DevicesView.vue` zůstal pod 300 řádky; prázdný název odmítne už
  klient (stejná hláška jako na obrazovce párování) bez požadavku na
  server; po přejmenování vlastního zařízení se aktualizuje i stav session
  v SPA. Existující E2E test odebrání vlastního zařízení hledá tlačítko
  podle názvu, protože řádek má nově dvě tlačítka.
- **Dokumentace aktualizována**: UX specifikace §7.4, tabulka endpointů
  v `docs/architecture/overview.md`, README (Pairing devices), slovník
  `ACCESS` ve `web/src/ui/vocabulary.ts`.
