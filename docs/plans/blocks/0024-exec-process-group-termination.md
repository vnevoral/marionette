# Implementační blok: Execution engine — terminace procesní skupiny a propagace kontextu

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-11, FR-19, FR-18, NFR-04, NFR-07
- **Vazba na ADR**: ADR-0005
- **Stav**: Hotovo
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

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (golangci-lint, eslint, vue-tsc,
  prettier, `go test -race -count=1 ./...`, build, vet). Nové testy:
  `TestExecutorKillsProcessGroupOnTimeout` (reálný `sh -c "echo $$; sleep 30
  & wait"`, timeout 1 s → návrat za 1,0 s, outcome `timeout`, `kill(-pgid,
  0)` hlásí ESRCH), `TestExecutorDoesNotHoldSlotForDetachedGrandchild`
  (`setsid sleep 6` drží pipe → návrat za 3,0 s díky `WaitDelay`),
  `TestExecutorCanceledByCaller`, `TestExecutorRejectsNilContext`,
  `TestRunnerWaitForSlotHonoursContext`, `TestCheckNowCanceledDoesNotUpdateStatus`,
  `TestSchedulerReconcileCancelsInFlightCheck` (Reconcile smazané karty s
  blokovaným checkem skončí < 100 ms), `TestBackgroundActionsCloseCancelsRunningJob`
  (vč. idempotentního druhého `Close`). Manuální end-to-end na reálné
  binárce: karta `sh -c "echo $$; sleep 60 & wait"` s timeoutem 3 s → 202,
  po 3,0 s záznam `outcome: timeout`, v procesní skupině ani mezi procesy
  `sleep 60` nic nezůstalo.
- **Odchylky od návrhu**: `BackgroundActions.Close()` nyní kontext jobů
  ruší okamžitě (běžící akce při shutdownu skončí jako `canceled` a zapíší
  se do historie); ochranné okno pro doběhnutí běžících akcí doplní blok
  0027 dle svého návrhu. Chyba `ErrStatusCheckCanceled` je v balíčku
  `status`; `internal/server` ji kvůli vrstvení nerozpoznává přes `errors.Is`,
  ale přes `ctx.Err()`. Test s `setsid` se přeskočí, pokud binárka není
  dostupná. Horní mez `TimeoutSec` zůstává na bloku 0028.
- **Dokumentace aktualizována**: ano — ADR-0005 (poznámka o procesní skupině,
  `WaitDelay` a `RunOutcomeCanceled`), godoc na `Execute`, `Run`, `CheckNow`,
  `Close`; frontend zobrazuje `canceled` jako „Canceled“; roadmapa.
