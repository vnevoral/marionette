# Implementační blok: Zachycení výstupu a vyhodnocení výsledku

- **Fáze**: 3 — Execution engine
- **Vazba na požadavky**: FR-13, FR-14, FR-17
- **Vazba na ADR**: ADR-0005
- **Stav**: Návrh

## Cíl bloku

Po dokončení execution engine vytvoří kompletní `config.Run` z jednoho
spuštění: zachytí kombinovaný stdout/stderr do 4 KB, označí oříznutí a
vyhodnotí exit code nebo `OutputRule`.

## Rozsah

- **Uvnitř**:
  - společný limit stdout+stderr 4096 bajtů;
  - `Run.Output` a `Run.Truncated`;
  - `exit_code`, `match` a `not_match` pravidla;
  - výsledek `ok`, `fail` nebo `timeout`;
  - zachycení exit code i při neúspěšném procesu;
  - testovatelná čistá funkce pro vyhodnocení výsledku.
- **Mimo rozsah**:
  - spouštění procesu a timeout (0004);
  - globální souběžnost (0006);
  - ukládání historie do store, polling a HTTP API.

## Návrh řešení

Rozšířit `internal/exec` o limitovaný writer a evaluator výsledku. Evaluator
nebude znovu kompilovat regex při každém běhu; konfigurace je validována při
uložení a načtení. Výstup se vyhodnocuje po oříznutí, v souladu s ADR-0004.

## Testovací plán

- exit code 0/ne-nula;
- validní `match` a `not_match`, včetně prázdného a neodpovídajícího výstupu;
- přesně 4096 bajtů a výstup nad limit;
- kombinovaný stdout/stderr s limitem;
- timeout má vždy `RunOutcomeTimeout`, i kdyby proces vrátil jiný exit code;
- neplatné pravidlo je odmítnuto před spuštěním.

## Kritérium hotovosti

Viz Definition of Done. Specificky: jednotkové testy evaluatoru pokrývají
hlavní i hraniční scénáře a `go test -race ./...` prochází.
