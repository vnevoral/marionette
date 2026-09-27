# ADR-0012: Entering an action as a single command line in the UI

- **Status**: Accepted
- **Date**: 2026-09-26

## Context

The action editor (primary and status) currently has a separate
**Command** field and repeatable **Arguments** rows, one row per argument
([UX specification §7.2](../../ux/ui-ux-specification.md)). During the
first configuration of a WoL + ping card on a Raspberry Pi (2026-09-26) it
turned out that this is very unpleasant for users: a command they know from
the terminal as a single line (`/usr/bin/ping -c 1 -W 2 192.168.1.10`) has
to be manually chopped into five rows. When they enter it in a single row,
the program receives a single argument `"-c 1 -W 2 192.168.1.10"`, the
action fails and the UI gives no clue why.

The action model (`Action.Command` + `Action.Args`, ADR-0004) and the
security rule NFR-01 a (execution via `exec.Command(name, args...)`, never
via a shell from a string) are correct and are to stay.

## Decision

1. The action editor replaces the Command field and the Arguments rows with
   a single **Command line** field. Working directory, Environment, Timeout
   and Output rule remain separate fields.
2. **Splitting happens only in the UI.** The API, domain model,
   persistence and execution engine do not change: before saving, the UI
   splits the line into `command` (the first word) and `args` (the rest)
   and on load it joins them back. The server never accepts or interprets
   the command line as text.
3. **The grammar** is a subset of the POSIX shell without any expansion:
   - words are separated by a space or a tab; the line must not contain a
     line break;
   - `'…'`: everything inside is literal, up to the next `'`;
   - `"…"`: everything literal except `\"` and `\\`;
   - outside quotes, `\` removes the special meaning of the next character;
   - adjacent parts without a space form one word (`a'b c'd` → `ab cd`);
     `''` or `""` is an empty argument;
   - nothing is expanded: variables, `~`, `*`, `?` or command output;
     `~`, `*` and `?` are ordinary characters.
4. **Shell characters outside quotes are an error.** `|`, `&`, `;`, `<`,
   `>`, `(`, `)`, `` ` `` and `$` without quotes or `\` are rejected by the
   editor with an explanation that Marionette runs commands without a
   shell, and with a hint (`/bin/sh -c '…'` for pipes, the Environment
   field for variables). Inside quotes they are ordinary characters. An
   unterminated quote and a `\` at the end of the line are errors as well.
5. **Joining back** an existing action: a word consisting only of the
   characters `A–Z a–z 0–9 _ @ % + = : , . / -` is written unchanged, an
   empty one as `''`, others in single quotes (`'` inside as `'\''`). For
   every action `parse(format(command, args)) = (command, args)` holds, so
   saving without edits does not change the action.
6. Below the field, the split result (the command and the individual
   arguments) is shown live, so that the user sees what will run before
   saving.

## Considered alternatives

- **Keep repeatable rows** — rejected; it is the main UX problem this ADR
  was created for.
- **Send the command line to the server and split it there** — rejected:
  it extends the API and backend validation, mixes the textual and the
  structured form of the action and moves security-sensitive parsing to the
  side where commands run. Splitting in the UI keeps the server contract
  structured.
- **Run the line via `/bin/sh -c`** — rejected; it violates NFR-01 a and
  opens up shell injection.
- **Split only on spaces, without quotes** — rejected; it would be
  impossible to enter an argument containing a space (a path, a regular
  expression) and existing actions with such an argument could not be
  displayed.
- **Silently treat shell characters as text** — rejected;
  `ping host | grep ttl` would look correct, but would pass `|` to ping as
  an argument and would fail without an understandable cause.

## Consequences

- Entering an action matches what the user knows from the terminal; the
  live split preview reveals errors before the action fails for the first
  time.
- The backend, API, persistence and the E2E API contract remain unchanged;
  existing configuration is not migrated.
- NFR-01 a still applies: the server runs structured `command` + `args`,
  never text via a shell. The parser is a pure function in the frontend and
  must have its own unit tests, including the join-back property.
- A user who really needs a pipe must write it explicitly via
  `/bin/sh -c '…'` and is responsible for its content.
- Server validation errors for `command` and `args[N]` are shown in the UI
  at the single Command line field.
- UX specification §7.2 changes: "Arguments … are repeatable rows" still
  applies only to environment variables.

## Addendum (2026-09-27, block 0053)

A line break is an error only **outside quotes**; inside `'…'` and `"…"` it
is part of the argument (the parser has done this from the start, item 3
just did not state it). An action saved via the API or in the file may have
a multi-line argument (e.g. a two-line script for `sh -c`). A single-line
text field would silently strip the line break when displaying it and the
first edit would change the script, so such a command line is edited in a
multi-line field that preserves line breaks. The field stays multi-line
until the editor is left, so that it is not swapped and does not lose focus
when the line break is deleted.
