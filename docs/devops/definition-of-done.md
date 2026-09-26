# Definition of Done (implementační blok)

Tento seznam je výstupní brána společného workflow popsaného v
[development-workflow.md](development-workflow.md). Blok může být označen jako
`Hotovo` až po splnění všech bodů.

Blok je hotový, když jsou splněny všechny následující body:

- [ ] Blok byl před implementací ve stavu `Schváleno` a během práce je ve
      stavu `Probíhá`.
- [ ] Kód odpovídá schválenému návrhu v bloku (`docs/plans/blocks/...`); pokud
      se návrh za implementace změnil, blok je aktualizován.
- [ ] `make verify` prochází bez chyb (build UI, `golangci-lint`, `eslint`
      s `--max-warnings 0`, `vue-tsc`, `prettier --check`, `go test -race`,
      `go build`, `go vet`). Je to jediná definice validační sady; dílčí
      příkazy (`make test`, `make lint`) slouží jen pro rychlou iteraci.
      Blok, který mění UI tok nebo HTTP kontrakt, navíc spustí `make e2e`.
- [ ] Nová/změněná doménová logika má jednotkové testy pokrývající hlavní i
      chybové scénáře (viz [testing-strategy.md](testing-strategy.md)).
- [ ] Veřejné API/chování je zdokumentováno (komentář na exportovaném
      symbolu, případně aktualizace `docs/architecture/overview.md`).
- [ ] Požadavky (`requirements.md`) a ADR, na které se blok odkazuje, souhlasí
      se skutečnou implementací; případné rozpory jsou vyřešeny (aktualizace
      dokumentu, ne tiché odchýlení).
- [ ] `docs/plans/roadmap.md` má u příslušné fáze/bloku aktualizovaný stav.
- [ ] Blok je po ověření označen jako `Hotovo` a jsou zaznamenané provedené
      validační příkazy nebo důvod výjimky.
- [ ] Žádná nová funkčnost navíc mimo schválený rozsah bloku (out of scope
      položky se řeší jako nový blok, ne „mimochodem“).
