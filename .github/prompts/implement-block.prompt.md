---
description: "Implementuje jeden schválený implementační blok vč. testů"
name: "Implementovat blok"
agent: "agent"
---

Implementuj jeden konkrétní implementační blok z `docs/plans/blocks/`.

Dodrž [jednotný vývojový workflow](../../docs/devops/development-workflow.md)
a [Definition of Done](../../docs/devops/definition-of-done.md).

Postup:

1. Pokud uživatel nezadal konkrétní blok, vyber první blok ve stavu
   `Schváleno` v pořadí roadmapy nebo se zeptej, pokud je výběr nejasný.
2. Blok ve stavu `Návrh`, `Zablokováno` nebo `Zamítnuto` neimplementuj.
3. Před změnou označ blok jako `Probíhá` a ověř jeho requirementy, přijatá ADR,
   závislosti, schválení a out-of-scope hranice.
4. Implementuj přesně schválený rozsah; potřebu mimo rozsah zastav a navrhni
   jako nový requirement, ADR nebo implementační blok.
5. Napiš jednotkové testy podle sekce „Testovací plán“ bloku a
   [testing-strategy.md](../../docs/devops/testing-strategy.md).
6. Spusť `make verify` (jediná definice validační sady: build UI, lint Go i
   web, `go test -race`, `go build`, `go vet`) a oprav chyby. Při iteraci
   lze použít dílčí `make test` / `make lint`.
7. Projdi [Definition of Done](../../docs/devops/definition-of-done.md),
   aktualizuj podle skutečnosti blok, requirements/ADR, veřejnou dokumentaci
   a `docs/plans/roadmap.md`.
8. Teprve po splnění DoD označ blok jako `Hotovo`; při překážce uveď důvod a
   použij stav `Zablokováno`.
9. V závěru uveď změny, ověření, případné odchylky a nesplněné položky DoD.
