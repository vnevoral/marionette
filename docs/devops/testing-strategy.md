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

## Nasazení (instalační skript)

- `deploy/install_test.sh` (`make deploy-test`, součást `make test` a CI jobu
  `backend`) nainstaluje balík do dočasného `DESTDIR` bez roota a bez
  systemd a ověří, že cesty z unity (`ExecStart`, `EnvironmentFile`,
  `WorkingDirectory` = `ReadWritePaths`, `MARIONETTE_CONFIG`) v instalaci
  existují, práva odpovídají dokumentaci (binárka 755, data 750,
  konfigurace 640), opakovaný běh zachová konfiguraci i defaults a soubor,
  který není ELF, je odmítnut. Chování se skutečným systemd (start, restart,
  reboot, hardening) se ověřuje ručně na referenčním hostu (blok 0023).

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
  ~80 %+, zbylé view se pokrývají v blocích 0032 a 0033.

## End-to-end (Playwright)

- `make e2e` (blok 0041) sestaví binárku s embedded SPA, spustí ji
  s prázdnou konfigurací v dočasném adresáři na `127.0.0.1:18080`
  (`web/e2e/start-server.sh`) a projde `web/e2e/*.e2e.ts` v headless
  Chromiu (`web/playwright.config.ts`, jeden worker — specy sdílí server
  a každá začíná smazáním všech karet přes API).
- Pokrývá, co jednotkové testy nevidí: skutečný prohlížeč, embed, SSE
  (indikátor **Live**, výsledek kontroly), 202 → čekání → výsledek, 422 až
  k poli formuláře, potvrzovací dialogy, NFR-12 s hlavičkami, které
  posílá prohlížeč (stránka na jiném originu nespustí akci), a absenci
  horizontálního scrollu ve 320 px na přehledu, detailu a formulářích
  (UX spec §9, §10 scénáře 1, 3, 5, 6, 7 a 2 částečně).
- Lokátory hledají prvky podle rolí a přístupných jmen (jako uživatel),
  CSS třídy jen tam, kde role chybí (badge, `.field-error`).
- Není součástí `make verify` (vyžaduje prohlížeč, ~15 s); běží jako
  samostatný CI job `e2e` a spouští se ručně u bloků, které mění UI tok.
  Prohlížeč se instaluje `npx playwright install --with-deps chromium`
  (v devcontaineru `post-create.sh`).

## Manuální ověření

- Před release na ARM host: spustit `make build-arm64`, nasadit na testovací
  Linux se systemd; referenční ověření provést na Raspberry Pi ARM64 s Ubuntu
  24.x, včetně `systemd` start/stop/restart a základního scénáře (WOL + ping)
  end-to-end.

## Co se netestuje (vědomě)

- Vzhled/vizuální regrese UI (žádný visual regression tool v MVP).
- Zátěžové testy — mimo očekávaný rozsah použití (jednotky/desítky karet,
  málo souběžných uživatelů).
