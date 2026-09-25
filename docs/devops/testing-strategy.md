# Testovací strategie

## Backend (Go)

- Jednotkové testy (`go test ./...`) pro veškerou doménovou logiku
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

- `npm run lint` a `vue-tsc` (type-check) jsou povinnou součástí CI.
- Komponentové/unit testy (Vitest + Vue Test Utils) se zavedou při
  implementaci fáze 6 (Dashboard UI) pro klíčovou logiku (zobrazení stavu
  karty, volání API) — netestuje se vzhled PrimeVue komponent samotných.

## Manuální ověření

- Před release na Raspberry Pi: spustit `make build-arm64`, nasadit na
  testovací Pi, ověřit `systemd` start/stop/restart a základní scénář (WOL +
  ping) end-to-end.

## Co se netestuje (vědomě)

- Vzhled/vizuální regrese UI (žádný visual regression tool v MVP).
- Zátěžové testy — mimo očekávaný rozsah použití (jednotky/desítky karet,
  málo souběžných uživatelů).
