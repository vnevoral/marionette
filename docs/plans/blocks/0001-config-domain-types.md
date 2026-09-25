# Implementační blok: Doménové typy a validace konfigurace

- **Fáze**: 2 — Doménový model + config store (in-memory + JSON perzistence)
- **Vazba na požadavky**: FR-10, FR-11, FR-12, FR-13, FR-14, FR-15, FR-15a, FR-17, FR-18, FR-19
- **Vazba na ADR**: ADR-0004
- **Stav**: Hotovo

## Cíl bloku

Po dokončení existuje balíček `internal/config` s doménovými typy
(`Action`, `OutputRule`, `ActionCard`, `Run`, `Settings`) a jejich validací,
bez jakékoli in-memory správy (store) nebo I/O na disk — to řeší až bloky
0002 a 0003.

## Rozsah

- **Uvnitř**:
  - Typy `Action`, `OutputRule`, `ActionCard`, `Run`, `Settings` v
    `internal/config` vč. JSON tagů pro budoucí (de)serializaci.
  - `OutputRule.Type` jako `"exit_code" | "match" | "not_match"` (výchozí
    `"exit_code"` při prázdné hodnotě); u `match`/`not_match` povinný
    `Pattern` platný jako `regexp`.
  - Validační metoda (např. `(ActionCard) Validate() error` a obdobně pro
    `Action`, `Settings`) pokrývající: povinná pole (id, název, primární
    akce, příkaz), kladný timeout, kompilovatelnost regexu u `OutputRule`,
    `Settings.HistorySize > 0`, `Settings.MaxConcurrentActions > 0`,
    nezáporné `PollingIntervalSeconds`/`FastPollingIntervalSeconds`/
    `FastPollingWindowSeconds` a `FastPollingIntervalSeconds <
PollingIntervalSeconds`, pokud je standardní polling zapnutý.
  - Konstantní výchozí hodnoty (`DefaultHistorySize = 20`,
    `DefaultMaxConcurrentActions = 4`, `DefaultPollingIntervalSeconds = 60`,
    `DefaultFastPollingIntervalSeconds = 10`,
    `DefaultFastPollingWindowSeconds = 120`) jako doporučené hodnoty pro
    nově vytvářené karty (nastavuje volající, např. API vrstva ve fázi 5),
    ne vynucené uvnitř `Validate()`.
- **Mimo rozsah** (řeší jiné bloky): in-memory store/CRUD (0002), JSON
  load/save a atomický zápis (0003), spouštění příkazů (fáze 3), HTTP API
  (fáze 5).

## Návrh řešení

```go
package config

type OutputRuleType string

const (
    OutputRuleExitCode OutputRuleType = "exit_code"
    OutputRuleMatch    OutputRuleType = "match"
    OutputRuleNotMatch OutputRuleType = "not_match"
)

type OutputRule struct {
    Type    OutputRuleType `json:"type"`
    Pattern string         `json:"pattern,omitempty"`
}

type Action struct {
    Command    string            `json:"command"`
    Args       []string          `json:"args,omitempty"`
    Dir        string            `json:"dir,omitempty"`
    Env        map[string]string `json:"env,omitempty"`
    TimeoutSec int               `json:"timeoutSec"`
    Rule       OutputRule        `json:"rule"`
}

type ActionCard struct {
    ID                          string  `json:"id"`
    Name                        string  `json:"name"`
    Description                 string  `json:"description,omitempty"`
    Icon                        string  `json:"icon,omitempty"`
    Primary                     Action  `json:"primary"`
    Status                      *Action `json:"status,omitempty"`
    PollingIntervalSeconds      int     `json:"pollingIntervalSeconds,omitempty"`
    FastPollingIntervalSeconds  int     `json:"fastPollingIntervalSeconds,omitempty"`
    FastPollingWindowSeconds    int     `json:"fastPollingWindowSeconds,omitempty"`
}

type RunOutcome string

const (
    RunOutcomeOK      RunOutcome = "ok"
    RunOutcomeFail    RunOutcome = "fail"
    RunOutcomeTimeout RunOutcome = "timeout"
)

type Run struct {
    ActionKind string        `json:"actionKind"` // "primary" | "status"
    StartedAt  time.Time     `json:"startedAt"`
    Duration   time.Duration `json:"duration"`
    ExitCode   int           `json:"exitCode"`
    Output     string        `json:"output"`
    Truncated  bool          `json:"truncated"`
    Outcome    RunOutcome    `json:"outcome"`
}

type Settings struct {
    HistorySize           int `json:"historySize"`
    MaxConcurrentActions  int `json:"maxConcurrentActions"`
}
```

Přesné názvy/rozložení polí může implementace mírně upravit, ale musí
zůstat kompatibilní s JSON strukturou popsanou v ADR-0004 (`settings` +
`cards` na top-level, viz blok 0003). `FastPollingIntervalSeconds`/
`FastPollingWindowSeconds` mají efekt jen když `PollingIntervalSeconds > 0`
(FR-15a) — validace to nevynucuje jako chybu, jen dokumentuje chování pro
scheduler (fáze 4). `RunOutcome` je třístavový (`ok`/`fail`/`timeout`)
namísto booleovského `Success`, aby šlo v historii odlišit vynucenou
terminaci (FR-19) od běžného neúspěchu.

## Testovací plán

- Validace `ActionCard`: chybějící název/primární akce/příkaz → chyba.
- `OutputRule`: `match`/`not_match` bez patternu → chyba; neplatný regex →
  chyba; `exit_code` bez patternu → OK.
- `Settings`: nula/záporné `HistorySize`/`MaxConcurrentActions` → chyba;
  kladné hodnoty → OK.
- Polling pole: záporné hodnoty → chyba; `FastPollingIntervalSeconds >=
PollingIntervalSeconds` při zapnutém standardním pollingu → chyba;
  `PollingIntervalSeconds == 0` s libovolnými fast-polling hodnotami → OK
  (bez efektu, viz FR-15a).
- Validní minimální `ActionCard` (jen primární akce, bez status akce, bez
  pollingu) → OK.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
`go build ./...`, `go vet ./...`, `go test ./...` procházejí; nový balíček
nemá žádné side-effecty (žádné I/O, žádné globální mutable state).
