---
description: "Navrhne nový záznam architektonického rozhodnutí (ADR)"
name: "Nové ADR"
agent: "agent"
---

Navrhni nové architektonické rozhodnutí podle
[šablony](../../docs/architecture/decisions/template.md).

Dodrž [jednotný vývojový workflow](../../docs/devops/development-workflow.md).
Tento krok pouze připravuje ADR; neimplementuj kód ani nevytvářej schválený
implementační blok automaticky.

Postup:

1. Over si v [docs/architecture/decisions/](../../docs/architecture/decisions)
   nejvyšší použité číslo a nový soubor pojmenuj `NNNN-kratky-nazev.md` (další
   číslo v pořadí, kebab-case název).
2. Vyplň Kontext (proč rozhodnutí vzniká, jaký FR/NFR nebo problém řeší —
   odkazuj na [requirements.md](../../docs/requirements/requirements.md)),
   Rozhodnutí, Zvažované alternativy (min. 1 zamítnutá) a Důsledky.
3. Nové ADR vytvoř se stavem **Navrženo**, dokud ho uživatel výslovně
   nepotvrdí jako **Přijato** — neimplementuj podle něj kód, dokud není
   přijaté.
4. Pokud rozhodnutí nahrazuje dřívější ADR, uveď to v obou souborech
   (staré: „Nahrazeno ADR-NNNN“, nové: odkaz zpět).
5. Aktualizuj [docs/architecture/overview.md](../../docs/architecture/overview.md),
   pokud rozhodnutí mění přehled komponent.
6. Na konci uveď závislé requirements a navazující implementační bloky, které
   bude třeba naplánovat po přijetí ADR.
