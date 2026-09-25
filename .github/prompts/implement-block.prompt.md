---
description: "Implementuje jeden schválený implementační blok vč. testů"
name: "Implementovat blok"
agent: "agent"
---

Implementuj jeden konkrétní implementační blok z `docs/plans/blocks/`.

Postup:

1. Pokud uživatel nezadal konkrétní blok, zeptej se který (nebo vyber první
   ve stavu „Schváleno“ v pořadí roadmapy).
2. Blok, který je ve stavu „Návrh“, neimplementuj — nejdřív uprozorni
   uživatele, že čeká na schválení.
3. Implementuj přesně rozsah popsaný v bloku (viz jeho sekce „Rozsah“); pokud
   při implementaci zjistíš, že je potřeba něco navíc mimo rozsah, zastav se a
   navrhni to jako samostatný blok, neimplementuj to „mimochodem“.
4. Napiš jednotkové testy podle sekce „Testovací plán“ bloku a
   [testing-strategy.md](../../docs/devops/testing-strategy.md).
5. Spusť `go build ./...`, `go vet ./...`, `go test ./...` (a pro web
   `npm run lint` + `npm run build`), oprav chyby.
6. Aktualizuj stav bloku na „Hotovo“ a odpovídající řádek ve
   `docs/plans/roadmap.md`.
7. Projdi [Definition of Done](../../docs/devops/definition-of-done.md) a
   potvrď splnění všech bodů, než oznámíš dokončení uživateli.
