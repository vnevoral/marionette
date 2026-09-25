# Implementační blok: Správa akčních karet

- **Fáze**: 6 — Dashboard UI
- **Vazba na požadavky**: FR-10, FR-11, FR-12, FR-14, FR-15, FR-23, FR-40
- **Vazba na ADR**: ADR-0003, ADR-0004, ADR-0006
- **Stav**: Hotovo
- **Závislosti**: Bloky 0010–0013, existující CRUD API v `internal/server`

## Cíl bloku

Administrátor bude moci v UI vytvořit, upravit a smazat akční kartu včetně
primární akce, volitelné status akce a polling nastavení. Formulář bude odesílat
strukturovaný JSON přes existující CRUD API; serverová validace zůstává
autoritativní.

## Rozsah

- **Uvnitř**:
  - samostatná route `/manage` dostupná z dashboardu;
  - seznam existujících karet s akcemi upravit a smazat;
  - formulář pro novou kartu a editaci existující karty;
  - pole identity karty: ID při vytvoření, název, popis a ikona;
  - editor primární a volitelné status akce: příkaz, argumenty, pracovní
    adresář, proměnné prostředí a timeout;
  - volba pravidla výsledku: exit code, match a not match včetně regex vzoru;
  - polling interval, fast-polling interval a fast-polling window;
  - klientská validace povinných polí a čísel před odesláním;
  - zpracování stavů `201`, `200`, `204`, `400`, `404`, `409` a `500`;
  - potvrzovací dialog před smazáním a návrat na dashboard po úspěšné změně;
  - PrimeVue formulářové komponenty, loading, empty a error stavy.
- **Mimo rozsah**:
  - nové backendové endpointy nebo změny doménové validace;
  - autentizace a autorizace;
  - editace historie běhů a status transition historie;
  - import/export karet, hromadné operace a drag-and-drop řazení;
  - testovací spuštění příkazu přímo z formuláře.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: Schváleno vlastníkem 2026-09-25; implementace probíhá.

## Návrh řešení

Rozšířit `web/src/api.ts` o typované funkce `createCard`, `updateCard` a
`deleteCard`. Přidat management view a znovupoužitelné komponenty pro editor
akce a dynamické řádky argumentů/proměnných prostředí. Formulář bude pracovat
s lokální kopií `ActionCard`, při editaci zachová ID z URL/stavu a při vypnutí
status akce odešle `status: undefined` tak, aby odpovídal `omitempty` kontraktu.
Po úspěšném zápisu se dashboard znovu načte; při chybě zůstane formulář
otevřený a zobrazí serverovou zprávu.

## Testovací plán

- TypeScript build ověří typy requestů, response dat a Vue šablon.
- ESLint a Prettier ověří frontendové konvence.
- Manuálně ověřit vytvoření, editaci, zapnutí/vypnutí status akce a smazání.
- Manuálně ověřit validaci prázdného příkazu, timeoutu, regexu a neplatného
  pollingového vztahu.
- Integračně přes běžící backend ověřit konfliktní ID, serverovou `400`,
  úspěšné `201/200/204` a obnovení dashboardu po změně.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
všechny mutace používají existující `/api/cards` endpointy, formulář nepřijde
o neuložené změny při chybě a `npm run lint` i `npm run build` projdou.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint`, `npm run build` — úspěšné; CRUD smoke test `201 → 200 → 204 → 404` — úspěšný
- **Dokumentace aktualizována**: ano; roadmapa odkazuje na tento blok
