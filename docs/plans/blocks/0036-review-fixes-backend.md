# Implementační blok: Opravy z code review — backend (same-origin za proxy, allowlist, store)

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: NFR-12, FR-30..33 (perzistence), NFR-05
- **Vazba na ADR**: —
- **Stav**: Hotovo
- **Závislosti**: Bloky 0026, 0028 (persistence), 0034

## Cíl bloku

Po dokončení funguje ochrana NFR-12 i za TLS-terminující reverse proxy a
allowlist `MARIONETTE_ALLOWED_HOSTS` porovnává hosty stejným pravidlem jako
`Origin`; config store po neúspěšném zápisu nehlásí falešně `Dirty()`.

## Rozsah

- **Uvnitř** (nálezy code review 2026-09-26):
  1. `internal/server/origin.go`: porovnání `Origin` × `Host` nezávisí na
     TLS stavu spojení — hostname se porovnává case-insensitive a porty se
     musí shodovat, nebo být oba „výchozí“ (`80`, `443` nebo nezadaný).
     Explicitní jiný port (`pi.local:8080`) se shodovat musí.
  2. Allowlist `MARIONETTE_ALLOWED_HOSTS` používá stejnou normalizaci:
     `pi.local`, `pi.local:80` i `pi.local:443` v seznamu odpovídají `Host`
     `pi.local`, `pi.local:80` i `pi.local:443`.
  3. `internal/config/store.go`: rollback po selhání `OnChange`
     (`CreateCard`, `UpdateCard`, `DeleteCard`) vrací počítadlo `changes`
     zpět, takže `Dirty()` neohlásí změnu, která se ve skutečnosti nestala.
  4. `internal/config/types.go`: odstranit wrapper `errorsAs`, volat
     `errors.As` přímo.
  5. Dokumentace: `docs/architecture/overview.md` (sekce NFR-12), README
     (`MARIONETTE_ALLOWED_HOSTS`), NFR-12 v requirements (jedna věta
     o proxy).
- **Mimo rozsah**:
  - důvěra hlavičce `X-Forwarded-Proto` / konfigurace trusted proxy;
  - autentizace (NFR-01, mimo MVP);
  - frontendové nálezy (blok 0037).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava stavu („naplanuj a oprav
  všechny nálezy“) po code review 18 commitů na `main`. Pravidlo
  „výchozí porty jsou rovnocenné“ je vědomý kompromis: `http://pi.local`
  a `https://pi.local` jsou obě operátorův host; jiná služba na stejném
  hostname s vlastním portem zůstává cizím originem.

## Návrh řešení

- `origin.go`: nová funkce `canonicalHost(host string) string` — lowercase,
  trim, `net.SplitHostPort`; port `80`/`443`/chybějící → jen hostname
  (IPv6 bez hranatých závorek), jinak `host:port`. `sameOrigin(origin,
  requestHost)` ztrácí parametr `tls`; allowlist se ukládá i porovnává přes
  `canonicalHost`. Komentáře `requireSameOrigin`/`sameOrigin` popisují
  skutečné pravidlo.
- `store.go`: v každé rollback větvi `store.changes--` místo `++`
  (paměť po rollbacku odpovídá poslednímu uloženému stavu).
- `types.go`: `errors.As(err, &validation)` na místě volání.

## Testovací plán

- `TestSameOriginComparison`: nové řádky — `https://pi.local` × `pi.local`
  → true, `http://pi.local` × `pi.local:443` → true, `https://pi.local:8443`
  × `pi.local` → false, `http://pi.local` × `pi.local:8080` → false.
- `TestAllowedHostsRestrictMutatingRequests`: allowlist `pi.local:80`
  přijme `Host: pi.local`, allowlist `pi.local` přijme `Host: pi.local:443`,
  `pi.local:8080` v seznamu nepřijme `Host: pi.local`.
- `TestStoreRollbackKeepsDirtyHonest`: po selhání `OnChange` je `Dirty()`
  false, pokud před pokusem bylo false.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- mutující požadavek s `Origin: https://pi.local` a `Host: pi.local` bez
  TLS je přijat;
- `docs/architecture/overview.md` už neuvádí odmítnutí za proxy jako známé
  omezení.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (lint, `vue-tsc`, Prettier, Vitest,
  `vite build`, `go test -race`, `go vet`). `TestSameOriginComparison`
  rozšířen o proxy případy (`https://pi.local` × `pi.local`,
  `pi.local:80`, `http://pi.local` × `pi.local:443` → shoda;
  `https://pi.local:8443` × `pi.local`, `http://pi.local` × `pi.local:8080`
  → neshoda), nový `TestCanonicalHost`,
  `TestAllowedHostsRestrictMutatingRequests` ověřuje `pi.local:80`
  v seznamu × `Host: pi.local`/`:443` a odmítnutí `pi.local` bez portu,
  když je v seznamu jen `pi.local:8080`. Nový
  `TestStoreRollbackKeepsDirtyHonest` (po selhání `OnChange` u create,
  update i delete je `Dirty()` false).
- **Odchylky od návrhu**: žádné. `sameOrigin` ztratil parametr `tls`;
  `request.TLS` se už v middleware nečte.
- **Dokumentace aktualizována**: ano — `docs/architecture/overview.md`
  (sekce NFR-12: kanonické porovnání hostů, odstraněno „známé omezení“ za
  proxy), README (`MARIONETTE_ALLOWED_HOSTS`, odstavec o cross-site
  ochraně), `docs/requirements/requirements.md` NFR-12, roadmapa.
