# Implementační blok: Dashboard karet a ovládání akcí

- **Fáze**: 6 — Dashboard UI
- **Vazba na požadavky**: FR-20, FR-21, FR-23, FR-40
- **Vazba na ADR**: ADR-0003, ADR-0004, ADR-0006
- **Stav**: Hotovo
- **Závislosti**: Bloky 0010–0012, REST API v `internal/server`

## Cíl bloku

Dashboard zobrazí všechny akční karty v přehledné mřížce, načte jejich poslední
známý status a umožní asynchronně zařadit primární nebo ruční status akci.
Konfigurace karet, historie a detailní CRUD editor zůstávají v navazujících
blocích.

## Rozsah

- **Uvnitř**:
  - typovaný klient pro seznam karet, status a enqueue endpointy;
  - načtení karet při otevření dashboardu a ruční obnovení;
  - karta se stavem `unknown`, `ok`, `fail` nebo `running`;
  - tlačítko primární akce, ruční status akce a deaktivace během požadavku;
  - chybové a prázdné stavy;
  - responzivní layout postavený na PrimeVue.
- **Mimo rozsah**:
  - create/update/delete formulář karty;
  - detail běhů a historie status transitions;
  - autentizace, websockety a server-side job IDs.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: Async endpointy se zobrazí jako přijaté; UI nečeká na dokončení procesu.

## Návrh řešení

Rozšířit `web/src` o malý API modul s typy odpovídajícími existujícím JSON
typům backendu a dashboardovou view. Stav `running` bude lokální UI projekce
po dobu enqueue požadavku; po přijetí se karta obnoví z read-only status API.
PrimeVue Aura zůstává jediným vizuálním základem.

## Testovací plán

- TypeScript build ověří typy API dat a Vue šablon.
- ESLint ověří Vue/TypeScript konvence.
- Manuálně ověřit prázdný seznam, chybu API, úspěšné načtení a obě enqueue akce.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
dashboard používá skutečné `/api` endpointy, ne health-check mock, a `npm run
lint` i `npm run build` projdou.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint`, `npm run build` — úspěšné; lint hlásí 29 neblokujících warningů
- **Dokumentace aktualizována**: ano; roadmapa odkazuje na tento blok
