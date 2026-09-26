# ADR-0005: Bezpečné spouštění akcí a limit souběžnosti

- **Stav**: Přijato
- **Datum**: 2026-09-25

## Kontext

Fáze 3 musí spouštět příkazy definované v `config.Action` na hostiteli bez
shellové interpolace, s povinným timeoutem, omezením výstupu a globálním
limitem souběžných procesů. Výsledek se následně uloží jako `config.Run`.

## Rozhodnutí

- Execution engine bude v balíčku `internal/exec` (název balíčku v Go bude
  `execengine`, aby se nepletl se standardním `os/exec`).
- Příkaz se spouští přes `os/exec.CommandContext(ctx, action.Command,
action.Args...)`; `Dir` a `Env` se nastaví přímo na `exec.Cmd`. Nebude se
  používat shell ani skládání příkazového řetězce.
- Každé spuštění vytvoří `context.WithTimeout` z `Action.TimeoutSec`.
  Po vypršení timeoutu se proces ukončí přes `CommandContext` a výsledek bude
  `RunOutcomeTimeout`; proces nesmí zůstat běžet na pozadí.
- Stdout a stderr se zachytí do jednoho společného limitovaného writeru o
  velikosti 4096 bajtů. Překročení limitu nastaví `Run.Truncated`, ale samo o
  sobě nezpůsobí chybu spuštění.
- Exit kód, výstup, čas startu a duration budou součástí výsledku. Selhání
  startu nebo běhu procesu se převede na `RunOutcomeFail`; chyba API bude
  vyhrazena neplatnému vstupu nebo interní chybě engine.
- Globální limit souběžnosti bude řešen buffered-channel semaforem. Každé
  ruční i budoucí polling spuštění musí získat slot a po dokončení ho uvolnit;
  akce nad limit čekají, nezahazují se.
- Engine bude závislý na úzkém rozhraní/factory pro vytvoření procesu, aby
  jednotkové testy nemusely spouštět reálné příkazy. Malý počet integračních
  testů ověří skutečný `echo` a timeout.

## Důsledky

- Uživatelský vstup se nikdy neinterpretuje jako shellový program.
- Timeout a limit výstupu jsou vynuceny na jednom místě pro primární i status
  akce.
- Engine nebude řešit konfiguraci karet, polling ani HTTP API; ty patří do
  config store, fáze 4 a fáze 5.

> Doplněno 2026-09-26 (blok 0024): terminace po timeoutu zabíjí celou
> **procesní skupinu** akce (`Setpgid` + `SIGKILL` na `-pgid`), ne jen přímého
> potomka, a `Cmd.WaitDelay` (2 s) ohraničuje čekání na uzavření výstupního
> pipe drženého případnými přeživšími procesy. `Executor.Execute` a
> `Runner.Run` přijímají kontext volajícího; jeho zrušení (shutdown,
> rekonfigurace karty) ukončí proces s výsledkem `RunOutcomeCanceled`, který
> se od `RunOutcomeTimeout` liší tím, že ho nevyvolal timeout akce. Status
> check zrušený volajícím nemění poslední známý stav karty.
