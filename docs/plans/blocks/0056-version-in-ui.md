# Implementační blok: Verze aplikace v UI

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze), oblast UI a provozu
- **Vazba na požadavky**: FR-41a, FR-41, FR-26
- **Vazba na ADR**: nové ADR není potřeba (čte existující endpoint)
- **Stav**: Hotovo
- **Závislosti**: Blok 0051 (verze z git tagu v `/api/health`)

## Cíl bloku

Po upgradu (README, „Upgrading and rolling back“) dnes operátor ověřuje
verzi příkazem `curl … /api/health` na hostu. Po dokončení bloku ji vidí
přímo v UI na každé obrazovce, i z telefonu.

## Rozsah

- **Uvnitř**:
  - `web/src/api.ts`: `getHealth()` pro `GET /api/health` (typ
    `{ status, version, uptime… }` podle `healthResponse`);
  - `AppShell.vue`: nenápadný řádek s verzí v patičce layoutu
    („Marionette v1.1.0“), tlumenou barvou a malým písmem. Verze se načte
    jednou po startu SPA. Selhání načtení verzi skryje a nic jiného
    neovlivní. Vývojový build ukáže `dev`, jak ho vrací server;
  - verze se znovu načte po znovupřipojení SSE (`Live`), protože to je
    typicky okamžik po restartu služby při upgradu. Stránka po upgradu tak
    ukáže novou verzi bez obnovení;
  - obrazovka párování (bez spárování) verzi nezobrazuje: `/api/health`
    je sice veřejný, ale obrazovka párování má zůstat minimální;
  - UX specifikace §2.1 (app shell);
  - testy: Vitest (`AppShell` zobrazí verzi, při chybě nic, po
    znovupřipojení načte znovu), E2E (patička ukazuje verzi serveru).
- **Mimo rozsah**:
  - upozornění, že je k dispozici novější verze (UI nemá odkud to zjistit
    a služba nemá přístup ven);
  - upozornění, že stránka běží se starším buildem UI než server
    (po upgradu se SPA načítá z nové binárky až po obnovení stránky;
    řešit, jen pokud to bude v praxi problém).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („souhlas se vším“).

## Návrh řešení

Patička místo horní lišty: verze není provozní informace pro běžnou práci
a na 320 px je v horní liště místo jen na navigaci a stav `Live`. Patička
je na konci každé obrazovky ve sdíleném `AppShell`, takže stačí jedno
místo v kódu.

## Testovací plán

- `cd web && npx vitest run`.
- `make e2e`.
- `make verify`.
- Ručně na Pi po upgradu: patička ukáže novou verzi po znovupřipojení.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- verze v patičce odpovídá `/api/health` a po restartu služby s novou
  verzí se změní bez obnovení stránky.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**: `make verify` a `make e2e` (20 scénářů) prošly. Nové testy:
  Vitest `AppVersion.spec.ts` (zobrazí verzi; při chybě nic; po
  znovupřipojení streamu načte verzi znovu, po prvním připojení ne),
  E2E „the footer shows the version the server reports (FR-41a)“
  (patička = `version` z `/api/health`); stávající test 320 px bez
  horizontálního scrollu prošel i s patičkou.
- **Odchylky od návrhu**:
  - verzi zobrazuje samostatná komponenta `AppVersion.vue` v patičce
    `AppShell`; layout shellu je sloupcový flex, takže patička je
    u krátkých stránek dole v okně;
  - znovunačtení verze sleduje přechod stream odpojen → připojen (ne
    `onRefresh`, který se volá i při každém pollingu během výpadku).
- **Dokumentace aktualizována**: UX specifikace §2.1, `docs/devops/ci-cd.md`
  (ověření verze v UI).
