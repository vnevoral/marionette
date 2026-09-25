# Definition of Done (implementační blok)

Blok je hotový, když jsou splněny všechny následující body:

- [ ] Kód odpovídá schválenému návrhu v bloku (`docs/plans/blocks/...`); pokud
      se návrh za implementace změnil, blok je aktualizován.
- [ ] `go build ./...`, `go vet ./...` a `make test` (resp. `go test ./...`)
      procházejí bez chyb.
- [ ] Pokud blok mění `web/`: `npm run lint` a `npm run build` procházejí bez
      chyb.
- [ ] Nová/změněná doménová logika má jednotkové testy pokrývající hlavní i
      chybové scénáře (viz [testing-strategy.md](testing-strategy.md)).
- [ ] Veřejné API/chování je zdokumentováno (komentář na exportovaném
      symbolu, případně aktualizace `docs/architecture/overview.md`).
- [ ] Požadavky (`requirements.md`) a ADR, na které se blok odkazuje, souhlasí
      se skutečnou implementací; případné rozpory jsou vyřešeny (aktualizace
      dokumentu, ne tiché odchýlení).
- [ ] `docs/plans/roadmap.md` má u příslušné fáze/bloku aktualizovaný stav.
- [ ] Žádná nová funkčnost navíc mimo schválený rozsah bloku (out of scope
      položky se řeší jako nový blok, ne „mimochodem“).
