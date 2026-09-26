# Testovací strategie

## Backend (Go)

- Jednotkové testy (`go test -race ./...`, spouští `make test`) pro veškerou doménovou logiku
  (vyhodnocení akcí, config store, status engine) — bez závislosti na
  skutečném spouštění procesů nebo souborovém systému, kde to jde (rozhraní +
  fake implementace).
- Execution engine se testuje přes abstrakci nad `os/exec` (interface s
  fake/mock implementací pro testy), reálné spouštění procesů se ověřuje jen
  v malém počtu integračních testů (např. `echo`, kontrola timeoutu).
- HTTP handlery (`internal/server`) se testují přes `net/http/httptest`.
- Cíl pokrytí: doménová logika a handlery ~80 %+; není cílem 100 % pokrytí
  triviálního kódu (gettery, `main.go`).

## Frontend (Vue)

- `npm run lint` (`--max-warnings 0`), `vue-tsc` (type-check) a
  `prettier --check` jsou povinnou součástí CI a `make verify`.
- Jednotkové a komponentové testy běží ve **Vitest** (`npm test`, součást
  `make test` a CI jobu `web`) v prostředí `happy-dom`; komponenty se
  montují přes Vue Test Utils s PrimeVue pluginem. Soubory `src/**/*.spec.ts`
  leží vedle testovaného kódu, sdílejí Vite konfiguraci (`vitest.config.ts`,
  alias `@`) a nedostávají se do bundlu. Testuje se čistá logika (API vrstva
  `api.ts` přes mock `fetch`, model formuláře `cardEditModel.ts`),
  composables (`useStatusEvents` přes injektovaný konektor,
  `useCardStatus` s falešnými časovači) a chování view i vlastních komponent
  (`HomeView`, `CardDetailView`, `CardEditView` s pamětovým routerem —
  `CardEditView` přes `RouterView`, aby platil `onBeforeRouteLeave`;
  `ActionEditor` včetně přístupných názvů, `ConnectionStatus`,
  `StatusBadge`, `ActionCard` pro každý stav, `StatusTimeline` s dobou
  trvání) a slovník/formátování (`ui/vocabulary`, `ui/format`) — netestuje
  se vzhled PrimeVue komponent samotných. REST
  funkce se mockují přes `vi.mock("@/api")` s částečným přepisem, SSE přes
  sdílený `src/test/fakeEventSource.ts` (`vi.stubGlobal("EventSource", …)`),
  potvrzovací dialog přes `src/test/fakeConfirm.ts` (provide místo
  `ConfirmationService`, test volá `accept`/`reject`).
  Pokrytí: `npm run test:coverage` (v8); cíl pro `api.ts` a čisté moduly
  ~80 %+, zbylé view se pokrývají v blocích 0032 a 0033. E2E/browser testy
  nejsou zavedené (samostatný blok po fázi 8).

## Manuální ověření

- Před release na ARM host: spustit `make build-arm64`, nasadit na testovací
  Linux se systemd; referenční ověření provést na Raspberry Pi ARM64 s Ubuntu
  24.x, včetně `systemd` start/stop/restart a základního scénáře (WOL + ping)
  end-to-end.

## Co se netestuje (vědomě)

- Vzhled/vizuální regrese UI (žádný visual regression tool v MVP).
- Zátěžové testy — mimo očekávaný rozsah použití (jednotky/desítky karet,
  málo souběžných uživatelů).
