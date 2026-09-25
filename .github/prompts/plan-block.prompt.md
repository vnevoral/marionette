---
description: "Rozpracuje fázi z roadmapy do konkrétního implementačního bloku"
name: "Naplánovat blok"
agent: "agent"
---

Rozpracuj zadanou fázi/téma z
[docs/plans/roadmap.md](../../docs/plans/roadmap.md) do jednoho nebo více
konkrétních implementačních bloků podle
[šablony](../../docs/plans/template-implementation-block.md).

Dodrž [jednotný vývojový workflow](../../docs/devops/development-workflow.md).
Tento krok je pouze plánovací: nesmí měnit zdrojový kód ani označit blok jako
`Schváleno` bez výslovného potvrzení vlastníka.

Postup:

1. Ověř, že fáze/téma má jasnou vazbu na FR/NFR a případně na přijaté ADR.
   Pokud artefakt chybí, zastav plánování a navrhni nejprve jeho doplnění.
2. Rozděl fázi na bloky dost malé na jednu implementační relaci, s jasným
   cílem, závislostmi, in/out of scope, návrhem a testovacím plánem.
3. Ulož každý blok jako `docs/plans/blocks/NNNN-nazev.md` (vzestupné číslo
   napříč všemi bloky, ne per fázi), se stavem `Návrh` a nevyplněným
   schválením.
4. Aktualizuj `docs/plans/roadmap.md` odkazy na nové bloky a jejich stav.
5. Na konci shrň závislosti, otevřené otázky a přesný krok vyžadující schválení.
6. Neimplementuj kód a blok neschvaluj v rámci tohoto promptu.
