# Implementační blok: Execution engine — terminace procesní skupiny a propagace kontextu

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-11, FR-19, FR-18, NFR-04, NFR-07
- **Vazba na ADR**: ADR-0005
- **Stav**: Schváleno
- **Závislosti**: Bloky 0004–0006 (execution engine), 0009 (scheduler), 0011 (orchestrace)

## Cíl bloku

Po dokončení timeout akce spolehlivě ukončí celý strom procesů spuštěných
akcí (nejen přímého potomka), slot souběžnosti se uvolní nejpozději krátce po
timeoutu a zrušení kontextu volajícího (scheduler worker, shutdown) přeruší
čekání na slot i běžící akci.

## Rozsah

- **Uvnitř**:
  - spouštění procesu ve vlastní procesní skupině (`Setpgid`) a `cmd.Cancel`,
    který při vypršení kontextu zabije celou skupinu (`kill(-pgid, SIGKILL)`);
  - `cmd.WaitDelay`, aby `Wait()` nečekal na uzavření stdout/stderr pipe
    drženého přeživšími vnuky;
  - signatura `Executor.Execute(ctx, action)` a `Runner.Run(ctx, action)`
    s kontextem volajícího; timeout akce se odvozuje `context.WithTimeout(ctx,
    …)`; čekání na slot semaforu reaguje na `ctx.Done()`;
  - scheduler worker a `BackgroundActions` předávají svůj kontext, aby
    `Reconcile`/`Stop` nečekaly na doběhnutí celého `TimeoutSec`;
  - horní mez `TimeoutSec` řeší blok 0028, zde se pouze dokumentuje;
  - integrační test s procesem, který spustí potomka a drží pipe.
- **Mimo rozsah**:
  - podpora Windows (není cílová platforma, `syscall.SysProcAttr` je Linux);
  - změna formátu `RunRecord` nebo API kontraktu;
  - změna pořadí kroků shutdownu (blok 0027);
  - omezení fronty a deduplikace (FR-18 upřesněno, řeší blok 0027).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava chyby (ne nová funkčnost).
  Nález revize 2026-09-26 (H-1, M-1). Reprodukováno
  mimo repo: `sh -c "sleep 30 & echo started; wait"` s timeoutem 1 s vrátilo
  `Run()` až po 30 s a potomek přežil. ADR-0005 slibuje, že proces nezůstane
  běžet; blok doplní do ADR-0005 poznámku o procesní skupině jako mechanismu.

## Návrh řešení

- `internal/exec/executor.go`, `osProcessFactory.New`: nastavit
  `command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}`,
  `command.Cancel = func() error { return syscall.Kill(-command.Process.Pid,
  syscall.SIGKILL) }`, `command.WaitDelay = 2 * time.Second`. Hodnota
  `WaitDelay` je konstanta balíčku s komentářem.
- `Executor.Execute(ctx context.Context, action config.Action) RunRecord`:
  vnitřní timeout `context.WithTimeout(ctx, action.Timeout())`; rozlišit
  `RunOutcomeTimeout` (vypršel timeout akce) od `RunOutcomeCanceled`
  (zrušil volající) — nový outcome je nutné promítnout do `config.RunOutcome`
  a jeho validace; UI ho zobrazí jako „Canceled“.
- `Runner.Run(ctx, action)`: `select { case r.slots <- struct{}{}: case
  <-ctx.Done(): return canceled }`.
- `internal/status/scheduler.go`: worker předává `worker.ctx` do `CheckNow`;
  `statusChecker` rozhraní dostane kontext. `Service.CheckNow(ctx, cardID)`.
- `internal/server/actions.go`: job dostane kontext `BackgroundActions`, který
  `Close()` zruší (konečné chování Close řeší 0027, zde jen předání ctx).
- Rozhraní `ProcessFactory`/`process` v testech dostanou kontext beze změny
  chování fake implementací.

## Testovací plán

- Jednotkové testy (`internal/exec`):
  - `TestExecutorKillsProcessGroupOnTimeout`: reálný `sh -c "sleep 30 & wait"`
    s timeoutem 1 s; `Execute` se vrátí do 3 s, outcome `timeout`, a
    `kill(-pgid, 0)` po návratu hlásí `ESRCH` (skupina neexistuje);
  - `TestExecutorCanceledByCaller`: zrušení ctx před timeoutem vrací
    `canceled`, ne `timeout`;
  - `TestRunnerWaitForSlotHonoursContext`: plný semafor + zrušený ctx →
    okamžitý návrat bez zablokování slotu;
  - existující testy semaforu a timeoutu upravit na novou signaturu.
- Jednotkové testy (`internal/status`): `Reconcile` odstraněné karty s workerem
  uprostřed dlouhé akce (fake runner blokující na ctx) skončí do 100 ms.
- `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`.
- Manuální ověření: karta s primární akcí `sh -c "sleep 60 & wait"` a
  timeoutem 5 s; po 5 s je slot volný (další karta se spustí) a `ps` neukazuje
  osiřelý `sleep`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- test s potomkem držícím pipe prochází pod `-race` do 3 s;
- `RunOutcomeCanceled` je zdokumentovaný v ADR-0005 a v typech;
- ADR-0005 obsahuje poznámku o procesní skupině a `WaitDelay`.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
