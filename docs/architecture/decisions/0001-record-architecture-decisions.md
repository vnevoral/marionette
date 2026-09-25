# ADR-0001: Rozhodnutí se zaznamenávají jako ADR v repozitáři

- **Stav**: Přijato
- **Datum**: 2026-09-25

## Kontext

Vývoj řídí primárně AI coding agenti společně s vlastníkem projektu. Aby
rozhodnutí o architektuře byla dohledatelná, zdůvodněná a nebyla znovu
otevírána bez kontextu, potřebujeme jednotné místo a formát pro jejich zápis.

## Rozhodnutí

Architektonická rozhodnutí se zapisují jako samostatné Markdown soubory ve
[docs/architecture/decisions/](.) ve formátu `NNNN-kratky-nazev.md` dle
[šablony](template.md), číslované vzestupně. Nové ADR se navrhují promptem
`/new-adr` a než jsou označeny jako „Přijato“, implementace na nich stavět
nesmí.

## Zvažované alternativy

- Rozhodnutí zapisovat jen do README/requirements — zamítnuto, chybí historie
  a odůvodnění jednotlivých rozhodnutí v čase.
- Nepoužívat žádný formální proces — zamítnuto, u AI-agentního vývoje hrozí
  nekonzistentní/protichůdná rozhodnutí mezi jednotlivými seancemi.

## Důsledky

- Každé netriviální architektonické rozhodnutí (volba knihovny, formátu
  perzistence, komunikačního protokolu apod.) musí mít odpovídající ADR.
- AGENTS.md odkazuje agenty na ADR log jako zdroj pravdy o „proč“.
