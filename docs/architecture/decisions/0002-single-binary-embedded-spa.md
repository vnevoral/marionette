# ADR-0002: Jeden Go binární soubor s vestavěným SPA

- **Stav**: Přijato
- **Datum**: 2026-09-25

## Kontext

Cílová platforma je Raspberry Pi s Ubuntu (linux/arm64) a obecně
nízkovýkonné/omezené prostředí, kde nechceme vyžadovat instalaci Node.js,
webserveru (nginx) ani jiného runtime jen kvůli provozu UI (viz FR-01, FR-02,
NFR-02).

## Rozhodnutí

Backend je napsán v Go a build produkuje jeden statický binární soubor.
Frontend (Vue 3 SPA) se sestaví přes Vite do `web/dist` a pomocí `go:embed`
(`internal/webui`) se vloží do binárky. HTTP server (`internal/server`) servíruje
API i statická aktiva SPA ze stejného procesu a portu, s fallbackem na
`index.html` pro klientský routing (Vue Router).

## Zvažované alternativy

- Samostatný Node.js server pro UI + Go API — zamítnuto, vyžaduje runtime a
  proces navíc na cíli (NFR-02).
- Servírování UI přes nginx/Apache vedle Go API — zamítnuto ze stejného
  důvodu, zbytečná provozní komplexita na Raspberry Pi.
- SSR/Nuxt — zamítnuto, zbytečná komplexita pro interní dashboard bez
  požadavku na SEO.

## Důsledky

- Release artefakt je jeden soubor per platforma (`bin/marionette`,
  `bin/marionette-linux-arm64`), instalace = zkopírovat + spustit (+ systemd
  unit, viz roadmapa fáze 7).
- Build pipeline musí vždy nejdřív sestavit UI (`make ui-build`) před
  `go build`, jinak embed selže nebo obsahuje starý obsah — řešeno pořadím
  cílů v `Makefile`.
- Vývojový režim (`ui-dev` + `backend-dev`) běží odděleně přes Vite proxy na
  `/api`, produkční build je vždy sjednocený do jednoho procesu.
