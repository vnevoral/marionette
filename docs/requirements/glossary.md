# Glosář pojmů

| Pojem                          | Popis                                                                                                                                                 |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Akční karta (Action Card)**  | Konfigurovatelná jednotka na dashboardu sdružující primární akci, status akci a metadata (název, ikona, popis).                                       |
| **Akce (Action)**              | Definice příkazu/procesu spuštěného na hostu (shell příkaz, argumenty, timeout, env) vč. pravidel pro vyhodnocení výsledku (exit code, výstup).       |
| **Status akce / Health check** | Akce stejného typu jako primární akce, jejímž účelem je zjistit aktuální stav entity, na kterou primární akce působí (např. `ping` po `Wake-on-LAN`). |
| **Stav karty**                 | Odvozený stav (např. `unknown`, `ok`, `fail`, `running`) vypočtený z posledního výsledku status akce.                                                 |
| **Host**                       | Stroj, na kterém běží instance Marionette a na kterém se reálně spouští nakonfigurované akce (Raspberry Pi / Ubuntu server).                          |
| **Config store**               | In-memory struktura držící všechny akční karty a akce za běhu, zálohovaná (persistovaná) do jednoho konfiguračního souboru (JSON).                    |
| **Dashboard**                  | Hlavní obrazovka webového UI zobrazující všechny akční karty a jejich stav.                                                                           |
| **Spuštění akce (Run)**        | Jeden konkrétní běh akce vč. zachyceného výstupu, exit kódu, času a výsledného vyhodnocení.                                                           |
