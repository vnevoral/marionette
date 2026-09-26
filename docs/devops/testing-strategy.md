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
- Komponentové/unit testy (Vitest + Vue Test Utils) se zavedou při
  implementaci fáze 6 (Dashboard UI) pro klíčovou logiku (zobrazení stavu
  karty, volání API) — netestuje se vzhled PrimeVue komponent samotných.

## Manuální ověření

- Před release na ARM host: spustit `make build-arm64`, nasadit na testovací
  Linux se systemd; referenční ověření provést na Raspberry Pi ARM64 s Ubuntu
  24.x, včetně `systemd` start/stop/restart a základního scénáře (WOL + ping)
  end-to-end.

## Co se netestuje (vědomě)

- Vzhled/vizuální regrese UI (žádný visual regression tool v MVP).
- Zátěžové testy — mimo očekávaný rozsah použití (jednotky/desítky karet,
  málo souběžných uživatelů).
