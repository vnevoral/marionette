# Implementační blok: Toast pro oznámení na úrovni stránky

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-26, FR-27, NFR-08, NFR-09
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0032, 0033, 0041

## Cíl bloku

Po dokončení UI rozlišuje dva kanály zpětné vazby: inline u karty, pole nebo
stránky, a toast pro výsledek operace, která stránku opouští nebo nahrazuje.
Uložení karty ukáže toast **Card saved**. Smazání karty, které dosud po
návratu na přehled nedávalo žádnou zpětnou vazbu, ukáže toast
**Card deleted** se jménem karty.

## Rozsah

- **Uvnitř**:
  - `ToastService` v `main.ts`, jeden `<Toast>` v `App.vue` (vpravo nahoře,
    na telefonu přes celou šířku mínus 16 px okraje);
  - `src/composables/useNotify.ts` (`success(summary, detail?)`, délka
    zobrazení `messageVisibleMs` sdílená s inline zprávami);
  - `CardEditView`: toast místo inline `notice`; `CardDetailView`: toast po
    úspěšném smazání, chyba smazání zůstává inline;
  - theme preset: úspěšný toast v barvách palety (AA kontrast);
  - `src/test/fakeToast.ts`, unit a E2E testy;
  - UX spec §4 (kanály zpětné vazby), §7.3, §8.4.
- **Mimo rozsah**:
  - výsledky akcí na kartách a v detailu (zůstávají inline, UX spec §4, §5.2);
  - chybové toasty (chyba se nikdy nezobrazí jen v toastu);
  - info/warn varianty presetu (nepoužívají se).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Pokyn „vyřeš ten toast“. Rozhodnutí o rozsahu
  (jen oznámení přežívající navigaci, ne náhrada inline zpětné vazby) je
  zdůvodněné v UX spec §4: více karet může hlásit výsledek současně a chyby
  patří k poli nebo kartě.

## Návrh řešení

Viz rozsah. `useNotify` obaluje `useToast` tak, aby views neznaly PrimeVue
API ani délku zobrazení. Toast po smazání se přidá až po navigaci na přehled.

## Testovací plán

- `CardEditView.spec.ts`: po vytvoření karty právě jeden toast `Card saved`
  (success, detail = jméno, 4000 ms), žádná inline zpráva.
- `CardDetailView.spec.ts`: po potvrzeném smazání toast `Card deleted`
  se jménem; při selhání smazání zůstává detail, chyba inline, žádný toast.
- E2E: toast po uložení a smazání (`role=alert`), ve 320 px se vejde na
  obrazovku, nezpůsobí horizontální scroll a má barvu z palety.
- `make verify`, `make e2e`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (Vitest 96 → 97), `make e2e` 7 → 8
  testů, prošlo. Kontrast úspěšného toastu: Aura výchozí `green.600` na
  `green.50` 3,15:1 (pod AA), nově `primary.600` (#397254) na accent-soft
  (#e6f1e8) 4,88:1, detail ink 11,34:1.
- **Odchylky od návrhu**: žádné. Při implementaci se ukázalo, že výchozí
  Aura barvy toastu nesplňují AA ani paletu; override v presetu je proto
  součástí bloku.
- **Dokumentace aktualizována**: ano — UX spec §4 (kanály zpětné vazby),
  §7.3 (uložení, smazání), §8.4 (sdílený Toast, `useNotify`), roadmapa.
