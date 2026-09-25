# ADR-0006: Historie statusu jako přechody stavů

- **Stav**: Návrh
- **Datum**: 2026-09-25
- **Nahrazuje část**: ADR-0004 týkající se ukládání každého status běhu do historie

## Kontext

Status akce se může spouštět každých 10–60 sekund, ale při stabilním zařízení
se stav nemění. Ukládání každé kontroly proto rychle zaplní limit historie a
neodpovídá tomu, co je pro operátora užitečné: kdy se stav změnil a jak dlouho
v něm zařízení zůstalo.

## Rozhodnutí

- Historie primární akce zůstává historií jednotlivých běhů (`Run`), protože
  každý pokus o provedení je významná událost.
- Status akce má dvě oddělené projekce:
  1. poslední kontrolu/stav pro dashboard (`last status check`), která se může
     aktualizovat při každém pollingu;
  2. historii přechodů stavů (`StatusChange`), do které se nový záznam přidá
     pouze při skutečné změně stavu.
- `StatusChange` obsahuje stav, čas začátku a čas konce nebo trvání. Aktuální
  otevřený stav nemá při zápisu ještě konečný čas; jeho aktuální délka se
  dopočítá z `now - StartedAt`. Po dalším přechodu se předchozí záznam uzavře.
- Opakovaná kontrola, která vrátí stejný stav, nemění historii přechodů.
- Historie přechodů má vlastní limit posledních `N` změn. Tento limit se
  nesnižuje počtem pollingových kontrol.
- Status `unknown` se použije při startu bez platné předchozí kontroly nebo
  při ztrátě informace; přechod do `ok`/`fail` se zaznamená stejně jako opačný
  přechod.
- Při graceful shutdown se persistuje poslední status projekce i historie
  přechodů. Jednotlivé opakované status kontroly se do historie nepersistují.

## Důsledky

- Historie statusu zůstane čitelná i při častém pollingu.
- Dashboard může zobrazit aktuální stav i dobu, po kterou trvá.
- Stavový engine musí atomicky aktualizovat poslední kontrolu a případný
  přechod; odpovědnost bude v implementačním bloku fáze 4.
- JSON schéma `history.status` se mění z `[]Run` na `[]StatusChange`; starší
  soubory s původním formátem status běhů vyžadují migrační/ignorační pravidlo
  v bloku persistence nebo fáze 4.
