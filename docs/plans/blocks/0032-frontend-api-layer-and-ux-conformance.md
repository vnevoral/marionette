# Implementační blok: API vrstva frontendu a shoda s UX specifikací

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-26, FR-27, FR-28, NFR-08, NFR-09, NFR-10
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Blok 0030 (testy), 0028 (kontrakt `fields` u 422)

## Cíl bloku

Po dokončení frontend zpracovává síťové a serverové chyby jednotně a
srozumitelně, destruktivní akce potvrzuje PrimeVue dialogem, formulář akce
je plně přístupný a indikátor spojení odpovídá skutečnému stavu.

## Rozsah

- **Uvnitř**:
  - `api.ts`: oprava spreadu options (zachování `Accept`), třída `ApiError`
    (`status`, `message`, `fields`), mapování `TypeError` na „Server
    unreachable“, kontrola `content-type` před `json()`, `AbortSignal`
    s timeoutem 15 s, typ `AcceptedAction`; `CARD_ICON_OPTIONS` přesun do
    `src/ui/icons.ts`; `duration` pole zdokumentovat jako nanosekundy
    nebo přejmenovat na `durationNs` po dohodě s backendem (kontrakt);
  - `ConfirmationService` + `<ConfirmDialog>` v `App.vue`; `useConfirm` pro
    smazání karty (pojmenuje kartu, akce „Delete card“) a pro opuštění
    neuloženého formuláře (`UnsavedChangesDialog` dle spec §8.4); tlačítko
    smazání disabled během mazání;
  - `ActionEditor.vue`: `Select` s lidskými labely pravidel výstupu,
    `aria-label` pro argumenty a env, tlačítko odebrání řádku, PrimeVue
    `Button` místo nativního `<button>`, stabilní klíče řádků, jedna cesta
    pro přidání/odebrání (vlastní dítě);
  - `CardEditView`: zobrazení `fields` z 422 u konkrétních polí, sticky
    save bar, po uložení zůstat na stránce s „Card saved“ nebo navigovat
    s toastem (rozhodne UX spec §7.3);
  - `AppShell`: `ConnectionStatus` navázaný na `useStatusEvents().connected`
    (z 0029);
  - `CardDetailView`: rozlišit 404 („Card not found“) od jiných chyb
    („Try again“), odstranit mrtvý `sectionErrors.status`, přechodné
    zprávy mazat po čase; backend `checkedAt` jako `omitempty` místo
    kontroly `0001-01-01` (malá změna typu ve `config`);
  - `index.html` `lang="en"`; opravit neplatné `margin: -var(...)`.
- **Mimo rozsah**:
  - slovník a sdílené komponenty (0033);
  - Toast pro všechny zprávy (zvážit v 0033);
  - nové obrazovky.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava chyb. Nález revize
  2026-09-26 (M1, M3, M7, M8, L1,
  L2, L3, L4). UX spec §2.1 a §7.1 si odporují v roli dashboardu vs. „Manage
  cards“ — před schválením vyjasnit ve spec.

## Návrh řešení

- `web/src/api.ts`: `export class ApiError extends Error { status; fields? }`;
  `request<T>` s `const { headers, signal, ...rest } = options ?? {}`.
- `web/src/App.vue`: `<ConfirmDialog />` + `<Toast />`; `main.ts`
  `app.use(ConfirmationService).use(ToastService)`.
- `web/src/components/ActionEditor.vue`: `defineModel<ActionDraft>()`,
  interní `rows: {id, key, value}[]` s `crypto.randomUUID()`.
- Backend: `StatusSnapshot.CheckedAt *time.Time` nebo `omitempty` s
  `IsZero` — jednořádková změna + test serializace.

## Testovací plán

- Vitest: `ApiError` mapování (network, 4xx JSON, 5xx HTML, 204, timeout);
  `ActionEditor` přidání/odebrání řádku, emitované hodnoty, `aria-label`
  přítomné; `CardEditView` zobrazí `fields` chybu u správného inputu.
- Manuální a11y průchod: klávesnicí vytvořit kartu s 2 argumenty a 1 env,
  smazat kartu přes dialog, opustit rozpracovaný formulář; ověřit
  screen-reader názvy (VoiceOver/NVDA nebo axe DevTools).
- `npm run lint`, `npm run build`, `npm test`; Go test serializace.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `window.confirm` se v `web/src` nevyskytuje;
- každý input v `ActionEditor` má přístupný název (test);
- indikátor spojení mění stav při zastavení backendu (manuální test).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (lint, `vue-tsc`, Prettier, Vitest 62
  testů v 10 souborech, `vite build`, `go test -race`, `go vet`). Nové a
  rozšířené testy: `api.spec.ts` (`ApiError` se `status` a `fields` u 422,
  404 → `isNotFound`, 2xx bez JSON, síťová chyba → „Server unreachable“
  se `status` 0, timeout 15 s s falešnými časovači), `cardEditModel.spec.ts`
  (`fieldErrorsFromServer`: mapování `primary.command`, `status.env.HOME`,
  `rule.pattern`… na klíče editoru, nezmapované cesty do souhrnu; id řádků),
  `components/ActionEditor.spec.ts` (přidání/odebrání řádků uvnitř
  komponenty, stabilní id, emitované hodnoty, lidské labely pravidel,
  chybové hlášky, **každý input i tlačítko má přístupný název**, Select
  má `aria-labelledby`), `components/ConnectionStatus.spec.ts` (Live /
  Reconnecting podle streamu, drží stream otevřený),
  `views/CardEditView.spec.ts` (422 `fields` u konkrétních polí + souhrn
  nahoře + zachované hodnoty, potvrzení opuštění rozpracovaného formuláře
  přes `useConfirm` s reject/accept, čistý formulář se neptá, nová karta po
  uložení zůstává v edit kontextu bez dalšího GET),
  `views/CardDetailView.spec.ts` (404 „Card not found“ vs. jiná chyba „Card
  unavailable“ + „Try again“, smazání až po potvrzení dialogem, který kartu
  jmenuje; chybová zpráva zmizí po 4 s). Go: `TestStatusSnapshotJSONOmitsCheckForUncheckedCard`.
  Smoke na reálném binárním serveru: `GET …/status` nekontrolované karty
  vrací `{"state":"unknown"}`, po kontrole plný snapshot; 422 nese `fields`;
  SPA má `lang="en"`. `window.confirm` se v `web/src` nevyskytuje. Manuální
  a11y průchod (screen reader/axe) a test indikátoru při zastavení backendu
  zůstávají na referenční host.
- **Odchylky od návrhu**: (1) Timeout požadavku je řešen vlastním
  `AbortController` + `setTimeout` (ne `AbortSignal.timeout`), aby byl
  testovatelný falešnými časovači a rozlišitelný od abortu volajícího.
  (2) `ToastService`/`<Toast>` se nezavádí — UX spec §7.3 rozhodla „zůstat
  v edit kontextu s Card saved“, toast tak nemá použití (zvážit v 0033).
  (3) Nová karta se po uložení přesměruje na `/cards/:id/edit`
  (`router.replace`), aby reload i další uložení mířily na uloženou kartu;
  formulář se přitom nenačítá znovu ze serveru (modulová proměnná
  `justSaved`, funguje i když router komponentu znovu vytvoří).
  (4) `ActionEditor` používá `defineModel` pro akci, argumenty i prostředí;
  řádky mají id z čítače `newRowId()` místo `crypto.randomUUID()` (deterministické
  v testech, bez závislosti na `crypto`). Typy řádků `ArgumentRow`
  a `EnvironmentRow` žijí v `cardEditModel.ts`. (5) `StatusSnapshot`
  zůstává s `time.Time`; vynechání `checkedAt`/`lastCheck` řeší
  `MarshalJSON` (Go 1.23 nemá `omitzero`), unmarshal je výchozí. Frontend
  typ má `checkedAt?`/`lastCheck?`; `isNewerCheck` už nezná `0001-01-01`.
  (6) `duration` zůstává v nanosekundách (Go `time.Duration`) — zdokumentováno
  v TS typech a v API kontraktu; přejmenování na `durationNs` by měnilo
  kontrakt bez přínosu pro jediného klienta. (7) `ConnectionStatus` je
  samostatná komponenta v `AppShell`; drží sdílený stream otevřený po celou
  dobu běhu aplikace (jedno SSE spojení pro celou SPA). (8) Rozpor UX spec
  §2.1 vs §7.1 vyřešen ve spec: navigace má jen **Overview**, správa karet
  probíhá z detailu (§7.1). (9) `CARD_ICON_OPTIONS` je v `src/ui/icons.ts`
  spolu s `DEFAULT_CARD_ICON`. (10) Falešný `useConfirm` pro testy
  (`src/test/fakeConfirm.ts`) čte `PrimeVueConfirmSymbol` přes cast, protože
  PrimeVue ho exportuje jen za běhu.
- **Dokumentace aktualizována**: ano — UX spec §2.1 (navigace, indikátor
  spojení), `docs/architecture/overview.md` (API kontrakt: `checkedAt`/
  `lastCheck` vynechány u nekontrolované karty, `duration` v ns),
  `docs/devops/testing-strategy.md` (nové testy a fake pro `useConfirm`),
  roadmapa.
