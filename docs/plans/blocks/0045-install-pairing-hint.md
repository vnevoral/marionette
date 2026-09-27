# Implementation block: Pairing instructions after installation

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-53, FR-01..04
- **ADRs**: ADR-0011
- **Status**: Done
- **Dependencies**: Blocks 0023, 0043, 0044

## Goal

After installation the UI is locked until the operator pairs the first
device with a code from the service log. `install.sh` did not mention this
so far and ended only with the message
"Marionette installed and started". Once done, on a host without paired
devices the installer prints the UI address and the command to find the
first code.

## Scope

- **In scope**:
  - `deploy/install.sh`: `print_pairing_hint` after a successful (including
    staged) installation. The instructions are printed only when
    `MARIONETTE_AUTH` is not `off` and the devices file does not exist. The
    path is derived from `MARIONETTE_DEVICES`, otherwise from the directory
    of `MARIONETTE_CONFIG`, the port from `MARIONETTE_ADDR`. The values are
    read from `/etc/default/marionette` with `sed`, the file is not executed
    (`source`);
  - `deploy/install_test.sh`: instructions on first installation, port
    according to `MARIONETTE_ADDR`, no instructions with an existing
    `devices.json` or with `MARIONETTE_AUTH=off`;
  - README (installation section).
- **Out of scope**:
  - printing the code itself by the installer (the service is only starting
    up and the code may not be in the log yet; the command from the
    instructions always works);
  - detecting the actual name or IP of the host (behind a VPN it would often
    be misleading, so the instructions show `<host>`).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: The answer "yes" to proposal no. 1 ("Pairing
  instructions after installation") after blocks 0043–0044 were closed.

## Proposed solution

See the scope. The instructions are printed in staged mode (`DESTDIR`) too,
so that the automated test covers them without root and systemd.

## Test plan

- `deploy/install_test.sh` (part of `make test` and CI).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `bash deploy/install_test.sh` passed, `make verify`
  passed. `make e2e` was not run: the block changes neither the UI flow nor
  the HTTP contract.
- **Deviations from the plan**: none.
- **Documentation updated**: yes — README (installation), roadmap.
