# Implementační blok: UX-03 — Management CRUD workflow

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-10, FR-22, FR-24, FR-26, FR-27, FR-28, FR-29, NFR-08..10
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0015–0016, existující CRUD API

## Cíl bloku

Manage cards bude skutečná CRUD obrazovka. Uživatel u každé existující karty
uvidí explicitní `Edit` a `Delete`, novou kartu založí přes jasnou CTA a po
smazání zůstane v management kontextu bez neočekávaného redirectu.

## Rozsah

- **Uvnitř**:
  - explicitní akce u každé položky seznamu: `Edit`, `Delete`;
  - stav vybrané položky a jasné označení otevřeného editoru;
  - `New card` v headeru/seznamu a prázdný stav bez duplicitních akcí;
  - potvrzení mazání s názvem karty;
  - obnovení seznamu po create/update/delete;
  - zachování formuláře při serverové chybě;
  - green-first tokeny, keyboard focus a mobilní layout.
- **Mimo rozsah**:
  - nový backend/API kontrakt;
  - detail karty, běhy a status timeline;
  - unsaved-changes guard při navigaci mimo stránku;
  - vyhledávání a hromadné operace.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: CRUD akce musí být viditelné přímo v seznamu karet.

## Návrh řešení

Rozšířit `ManageView.vue` o card row s názvem, ID, konfigurací status checku a
explicitními PrimeVue Button akcemi. Delete bude pracovat s ID a názvem položky,
nejen s právě otevřeným formulářem. Po úspěšném delete se seznam načte znovu a
editor se resetuje na nový card state. Formulářové API a `ActionEditor` zůstanou
beze změny.

## Testovací plán

- `npm run format`, `npm run lint`, `npm run build`.
- Browser smoke test: prázdný seznam, New card, Edit a Delete jsou viditelné.
- API smoke test create/update/delete ověří, že UI používá existující endpointy.
- Keyboard smoke test projde list actions a editor.

## Kritérium hotovosti

Na `/manage` jsou CRUD akce zřejmé bez znalosti, že se má klikat na název karty;
mazání vyžaduje potvrzení, chyba zachová data formuláře a po úspěšné operaci
zůstává seznam konzistentní.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint`, `npm run build` — úspěšné; browser smoke test potvrdil Edit/Delete u existujících karet; CRUD API smoke test `201/200/204/404` — úspěšný
- **Dokumentace aktualizována**: ano; roadmapa a UX specifikace
