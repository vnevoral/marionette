# Implementační blok: API vrstva frontendu a shoda s UX specifikací

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-26, FR-27, FR-28, NFR-08, NFR-09, NFR-10
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Schváleno
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

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
