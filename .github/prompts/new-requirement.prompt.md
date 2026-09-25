---
description: "Zapíše nebo upraví jeden požadavek v docs/requirements/requirements.md"
name: "Nový požadavek"
agent: "agent"
---

Zaznamenej nový nebo upravený požadavek do
[docs/requirements/requirements.md](../../docs/requirements/requirements.md).

Postup:

1. Zjisti od uživatele (pokud to není zřejmé ze zadání), zda jde o funkční
   (FR) nebo nefunkční (NFR) požadavek a do které sekce patří.
2. Přiděl další volné ID v rámci dané sekce (např. další `FR-1x`), nepřečíslovávej existující ID.
3. Zapiš požadavek stručně, jednoznačně a testovatelně (jedna věta/odstavec),
   ve stejném stylu jako okolní položky.
4. Pokud požadavek mění nebo ruší existující položku, existující položku
   neproříznout beze stopy — označ ji jako nahrazenou/aktualizovanou s
   odkazem na nové ID, ať zůstává historie.
5. Pokud požadavek implikuje architektonické rozhodnutí (nová závislost,
   způsob perzistence, protokol...), uprozorni uživatele, že by mělo vzniknout
   i ADR (`/new-adr`), ale sám ho nevytvářej bez potvrzení.
6. Zkontroluj, jestli požadavek nekoliduje s existujícím ADR — pokud ano,
   upozorni na rozpor místo tichého zapsání.
