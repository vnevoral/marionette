---
description: "Rozpracuje fázi z roadmapy do konkrétního implementačního bloku"
name: "Naplánovat blok"
agent: "agent"
---

Rozpracuj zadanou fázi/téma z
[docs/plans/roadmap.md](../../docs/plans/roadmap.md) do jednoho nebo více
konkrétních implementačních bloků podle
[šablony](../../docs/plans/template-implementation-block.md).

Postup:

1. Over, že fáze/téma má jasnou vazbu na FR/NFR v requirements.md a případně
   na ADR — pokud chybí, uprozorni uživatele a navrhni nejdřív doplnit
   požadavek/ADR místo rovnou plánování.
2. Rozděl fázi na bloky dost malé na jednu implementační relaci (řádově
   hodiny, ne dny), s jasným rozsahem (in/out of scope) a testovacím plánem.
3. Ulož každý blok jako `docs/plans/blocks/NNNN-nazev.md` (vzestupné číslo
   napříč všemi bloky, ne per fázi).
4. Aktualizuj `docs/plans/roadmap.md` odkazem na nově vzniklé bloky a jejich
   stav (výchozí: „Návrh“).
5. Neimplementuj kód v rámci tohoto promptu — pouze plán.
