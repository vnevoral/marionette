# Implementační blok: Párování zařízení — UI a E2E

- **Fáze**: 8 — Zpevnění (položka „auth/access control“)
- **Vazba na požadavky**: FR-50..FR-55, FR-26, NFR-08, NFR-09
- **Vazba na ADR**: ADR-0011, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Blok 0043

## Cíl bloku

Po dokončení nespárovaný prohlížeč uvidí obrazovku **Pair this device**,
zadá kód a název zařízení a dál už se nikdy neověřuje. Spárované zařízení má
stránku **Devices** se seznamem, odebráním a generováním kódu pro další
zařízení. E2E sada běží se zapnutým ověřováním.

## Rozsah

- **Uvnitř**:
  - `api.ts`: `getSession`, `pairDevice`, `listDevices`, `removeDevice`,
    `createPairingCode`; `ApiError.isUnauthorized`;
  - router: kontrola session při startu aplikace a přesměrování na
    `/pair?next=…` po jakékoli `401`; po spárování návrat na `next`;
  - `PairView` (`/pair`): kód (8 znaků, velká písmena, pomlčka volitelná,
    předvyplnění z `?code=`), název zařízení (předvyplněný podle prohlížeče
    a systému), při prvním zařízení nápověda
    `journalctl -u marionette` s příkazem ke zkopírování, chyby inline;
  - `DevicesView` (`/devices`): seznam (název, spárováno, naposledy,
    „This device“), odebrání přes potvrzovací dialog, **Pair a new
    device** → kód, odpočet platnosti, odkaz ke zkopírování;
  - `AppShell`: položka navigace **Devices**; na `/pair` shell bez
    navigace a bez indikátoru Live;
  - `useStatusEvents`: při `401` z REST fallbacku přestane pollovat;
  - E2E: `setup` projekt spáruje prohlížeč bootstrap kódem z logu serveru
    a uloží `storageState`; specy: párování, chybný kód, odebrání zařízení
    → návrat na párování, druhé zařízení kódem z Devices, API bez tokenu
    → 401; stávající specy běží spárované;
  - UX spec (IA §2, nové obrazovky, slovník), testing strategy, README.
- **Mimo rozsah**: QR kód (nová závislost), přejmenování zařízení,
  backend (blok 0043).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: viz blok 0043.

## Návrh řešení

- Obrazovky používají stávající komponenty (`PageHeader`, `DetailPanel`,
  `RequestState`, `ConfirmDialog`, toast přes `useNotify` pro „Device
  paired“ a „Device removed“ — obojí mění stránku, UX spec §4).
- Kód se zobrazuje velkým neproporcionálním písmem ve skupinách 4+4
  (`ABCD-EFGH`), odkaz `/<base>/pair?code=ABCDEFGH`.

## Testovací plán

- Vitest: `PairView` (normalizace kódu, předvyplnění z query, chyby
  z 400, bootstrap nápověda), `DevicesView` (seznam, odebrání s potvrzením,
  odebrání vlastního → `/pair`, generování kódu a odpočet), router guard
  (401 → `/pair?next`), `api.ts` nové funkce.
- E2E viz rozsah; `make e2e`, `make verify`.
- Kontrast a 320 px pro obě nové obrazovky (E2E).

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (Vitest 97 → 119: `api.spec.ts` session
  paired/unpaired/bootstrap/open, párování, kód, seznam, odebrání, 401
  handler jen mimo session a párování; `PairView.spec.ts` nápověda s logem
  jen bez zařízení, formát a předvyplnění kódu, návrh jména, návrat na
  `next`, žádný návrat mimo aplikaci, jedna zpráva pro chybný kód, chyby
  jména; `DevicesView.spec.ts` seznam s „This device“ a platností,
  odebrání jiného zařízení inline, odebrání vlastního → párování + toast,
  kód s odpočtem a odkazem, detekce nově spárovaného zařízení, nový kód po
  vypršení, vypnuté ověřování; `deviceName.spec.ts`, `params.spec.ts`).
  `make e2e` se zapnutým ověřováním: 14 testů (setup spáruje prohlížeč
  kódem z logu serveru; nespárovaný prohlížeč vidí jen párování a API
  vrací 401; chybný kód; druhý prohlížeč kódem z Devices, detekce bez
  reloadu, odebrání a návrat odebraného na párování; párovací odkaz,
  odebrání vlastního zařízení = odhlášení; 320 px pro Devices, párování
  a zobrazený kód; všechny dřívější scénáře jako spárované zařízení);
  3 opakované běhy 14/14.
- **Odchylky od návrhu**: (1) Oprávnění k `/pair` řeší guard routeru přes
  `meta.pairing` místo samostatného layoutu; `AppShell` na této routě skryje
  navigaci i indikátor Live, takže se neotevře SSE stream bez tokenu.
  (2) Kód se zobrazuje v elementu `<output>` (jednou ho ohlásí čtečka),
  ne v `<p aria-label>`, který by čtečky ignorovaly. (3) `useStatusEvents`
  se neměnil: po 401 router odvede na párování, view se odpojí a stream i
  polling skončí s posledním odběratelem. (4) `tsconfig.node.json` dostal
  `target` ES2022 a `lib` s DOM, protože nově kontroluje i E2E soubory.
  (5) Přihlášený prohlížeč na `/pair` (např. otevřený párovací odkaz) je
  přesměrován na `next` nebo přehled.
- **Dokumentace aktualizována**: ano — UX spec §2.1, §2.2, nová §7.4,
  testing strategy (E2E se setup projektem), README (vývoj se zapnutým
  ověřováním), roadmapa.
