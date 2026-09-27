# Implementation block: Action command as a single line in the editor

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-22a, FR-11, FR-22, FR-26, FR-27, FR-28,
  NFR-01 a
- **ADRs**: [ADR-0012](../../architecture/decisions/0012-single-line-command-editor.md)
  (must be `Accepted`), ADR-0004, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0014, 0017, 0032 (card editor and error mapping),
  0041 (E2E)

## Goal

The user enters both the primary and the status action as a single line,
just like in a terminal (`/usr/bin/ping -c 1 -W 2 192.168.1.10`), and sees
below the field how the line splits into the command and the arguments. The
stored form of the action (`command` + `args`) and the API do not change.

## Scope

- **In scope**:
  - a pure module `web/src/views/commandLine.ts`:
    `parseCommandLine(line)` → `{ command, args }` or an error with a
    position and a message; `formatCommandLine(command, args)` → line; the
    grammar, forbidden characters and quoting rules exactly per ADR-0012;
  - `ActionEditor.vue`: a **Command line** field instead of Command and
    Arguments, a live preview of the split (command + arguments as readable
    items, accessible to a screen reader), a split error at the field;
    Working directory, Environment, Timeout and Output rule unchanged;
  - `cardEditModel.ts`: the form holds a line instead of `ArgumentRow[]`;
    client-side validation (empty line, split error) blocks saving;
    server errors `primary.command`, `primary.args`,
    `primary.args[N]` (and the same for `status`) are mapped to the Command
    line field; removal of `ArgumentRow` and `argumentRows` if no other use
    remains;
  - the text vocabulary in `src/ui/vocabulary.ts` (FR-26): label, hint,
    error messages (unterminated quote, shell character with the hint
    `/bin/sh -c '…'`, variable with a pointer to Environment);
  - UX specification §7.2 and the scenarios in §10 per ADR-0012;
  - E2E scenarios that currently fill in argument rows.
- **Out of scope**:
  - any change to the API, backend, server-side validation or persistence;
  - migration of the stored configuration (not needed);
  - expansion of variables, `~` or globs; execution through a shell;
  - entering Environment as `KEY=value` at the start of the line (variables
    stay in a separate field).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved by the owner ("I approve everything") together with the acceptance of ADR-0012. Shell characters outside quotes are an error, not text.

## Proposed solution

- The parser is a hand-written state machine over characters (states:
  outside quotes, in `'…'`, in `"…"`, after `\`), with no regular
  expressions over the whole line and no dependencies. It returns the first
  error with a character index, so that a clear message can be shown.
- `formatCommandLine` quotes per ADR-0012 point 5; it is called once when
  the form is initialized to load an existing action, the line is not
  reformatted while typing (the user's cursor does not jump).
- The form's dirty check compares the split form, not the text: adding an
  extra space does not make the form dirty.

## Test plan

- Unit tests of `commandLine.ts`: a simple command; multiple spaces
  and tabs; `'…'`, `"…"` with `\"` and `\\`; `\ ` outside quotes; joining
  parts (`a'b c'd`); an empty argument `''`; an unterminated quote; `\` at
  the end; every forbidden character both outside and inside quotes; `~`,
  `*`, `?` as text; an empty and a whitespace-only line; a line break in the
  input; the property `parse(format(c, a)) = (c, a)` on a set of cases
  including arguments with a space, `'`, `"`, `\`, `$`, `|` and empty ones.
- Tests of `ActionEditor` and `cardEditModel`: split preview, error at the
  field, mapping server errors `args[N]` to Command line, loading an
  existing card and saving without changes sends the same `command` +
  `args`.
- `make e2e`: creating a card with a single line, editing an existing card
  with an argument containing a space, rejecting `ping host | grep ttl`.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the action `/usr/bin/ping -c 1 -W 2 192.168.1.10` can be entered as a
  single line and is saved as `command: /usr/bin/ping`, `args: [-c, 1, -W, 2,
  192.168.1.10]`;
- the API contract and the backend are unchanged (`git diff` without
  changes in `internal/` and `cmd/`).

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (among others 30 tests in
  `commandLine.spec.ts` including the property `parse(format(c, a)) = (c, a)`);
  `make e2e` passed including the new scenario `edits a stored action as one
  command line` (loading `echo 'hello from' quoted`, rejecting
  `echo hi | grep hi`, saving `echo 'hello again' done` →
  `args: [hello again, done]`). `git diff` in `internal/` and `cmd/`
  contains only the generated `internal/webui/dist/index.html` from the UI
  build; the backend and the API do not change.
- **Implementation**: `web/src/views/commandLine.ts`
  (`parseCommandLine`, `formatCommandLine`, `quoteWord`) per ADR-0012;
  `ActionEditor` has a **Command line** field with a hint, a live preview
  (the list "Command and arguments") and a split error; `cardEditModel`
  holds a line instead of `ArgumentRow[]` (`commandLineOf`, `actionFrom`,
  `validate` accepts the editor state), server errors `args` / `args[N]`
  belong to the Command line field; the `ArgumentRow` type was removed.
- **Deviations from the plan**:
  - the field's labels and hint stayed directly in `ActionEditor.vue` like
    the other editor labels; the split error messages are in
    `commandLine.ts` next to the grammar, not in `vocabulary.ts`;
  - fixed a coupling that the change revealed: an enabled status action for
    which the user filled in only the command line would not be saved
    (`form.status` stayed undefined); the payload, dirty check and
    validation now use `emptyAction()` as default values (test in
    `CardEditView.spec.ts`).
- **Documentation updated**: yes — UX specification §7.2, ADR-0012
  (Accepted), roadmap.
