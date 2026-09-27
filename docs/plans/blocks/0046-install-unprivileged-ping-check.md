# Implementation block: Unprivileged ping check during installation

- **Phase**: 7 — Packaging and deployment
- **Requirements**: FR-05, FR-16, FR-02
- **ADRs**: ADR-0002 (no new ADR is needed: neither the architecture nor
  the unit changes, only the installer and documentation)
- **Status**: Done
- **Dependencies**: Block 0023 (installer, unit), block 0045 (the
  `print_pairing_hint` pattern)

## Goal

During validation on a Raspberry Pi (Ubuntu, 2026-09-26), a card with the
reference WoL + ping scenario permanently reported **Problem**, even
though the target PC was running. `sudo -u marionette ping …` succeeded,
but inside the service `ping` ended with
`socket: Operation not permitted` (exit 2): the unit has
`NoNewPrivileges=true`, so the `cap_net_raw` file capability on
`/usr/bin/ping` is ignored, and the host had
`net.ipv4.ping_group_range = 1 0` (unprivileged ICMP disabled).
The fix is `net.ipv4.ping_group_range = 0 2147483647`.

After this block, the installer detects this state and prints
instructions for fixing it; the README describes the cause and how to
verify it in the service environment.

## Scope

- **In scope**:
  - `deploy/install.sh`: a `print_ping_hint` function called after a
    successful (also staged) installation next to `print_pairing_hint`.
    It reads `/proc/sys/net/ipv4/ping_group_range` (path overridable by an
    environment variable for the test); if the range does not include the
    service user's primary group (in staged mode, when the group does not
    exist yet, it checks whether the range is empty, i.e. `min > max`), it
    prints an explanation and the commands:
    `echo 'net.ipv4.ping_group_range = 0 2147483647' | sudo tee /etc/sysctl.d/99-marionette-ping.conf`
    and `sudo sysctl --system`. An unreadable file (different kernel,
    container) is silently skipped;
  - `deploy/install_test.sh`: the hint is printed for range `1 0`, not
    printed for `0 2147483647` nor for a missing file;
  - README: section "Ping status action reports Problem" — the cause,
    verification with the command
    `systemd-run … -p User=marionette -p NoNewPrivileges=true /usr/bin/ping …`,
    the fix, and why
    `AmbientCapabilities=CAP_NET_RAW` is not recommended (all actions would
    get raw sockets);
  - block 0023: a reference to this finding in the ARM64 host
    verification section.
- **Out of scope**:
  - automatic `sysctl` change by the installer (FR-05: host system
    settings are not changed without the operator's knowledge; see the
    open question below);
  - unit changes (`AmbientCapabilities`, `CapabilityBoundingSet`);
  - general diagnostics of other action commands (`wakeonlan`, `nc` …).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved by the owner ("I approve everything"). The open question was decided in favor of the proposal: the installer only prints instructions and does not change sysctl itself.

## Proposed solution

See scope. The check is purely informational and never causes the
installation to fail. The values are read with `read` from a single line
(`min max`), without calling `sysctl(8)`, so the test needs neither root
nor a real kernel.

## Test plan

- `deploy/install_test.sh` (part of `make test` and CI) with three
  variants of a faked `ping_group_range`.
- Manually on the reference Raspberry Pi: with `1 0` the installer prints
  the hint; after the fix and a repeated installation it no longer does,
  and the status action of the WoL + ping card reports **Healthy** for a
  powered-on PC.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the scenario from the goal is described in the README so that the
  operator can resolve it without knowledge of systemd hardening.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `bash -n deploy/install.sh` and `bash deploy/install_test.sh`
  passed (variants `1 0` → hint, `0 2147483647` → no hint, missing
  file → no hint, existing user `root` with range
  `1 2147483647` → hint). `shellcheck` is not available in the
  environment. `make verify` passed (including `install_test`). On the
  reference Raspberry Pi (2026-09-26, release `9bb21bf`) the project
  owner confirmed the installation and a working `ping` status action
  (WoL + ping card **Healthy**).
- **Deviations from the plan**:
  - the environment variable for the test is named `PING_GROUP_RANGE_FILE`
    (consistent with `DATA_DIR`, `UNIT_FILE` …);
  - the primary group of the existing service user is checked
    (`id -g`), otherwise the GID of the `SERVICE_GROUP` group; if neither
    exists, only an empty range (`min > max`) decides. The test also
    covers the branch with an existing user;
  - the README section is in English ("Ping status check reports
    Problem") because the whole README is in English;
  - the reference from block 0023 to this finding already exists from the
    block's planning (item "Finding on Raspberry Pi").
- **Documentation updated**: yes — README (installation + section
  "Ping status check reports Problem").
