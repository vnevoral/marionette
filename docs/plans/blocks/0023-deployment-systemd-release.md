# Implementation block: Phase 7 — systemd deployment and release artifact

- **Phase**: 7 — Packaging and deployment
- **Requirements**: FR-01..FR-04, NFR-02
- **ADRs**: ADR-0002
- **Status**: Done
- **Dependencies**: Blocks 0001–0022, existing `make build-arm64`

## Target environment

- production target: generic Linux with systemd, including Raspberry Pi ARM64
  (`linux/arm64`);
- reference validation environment: Ubuntu 24.x on Raspberry Pi ARM64 and
  Ubuntu 24.x `linux/amd64`;
- the target host must not require Node.js, Go or any other runtime.

## Goal

Once complete, it will be possible to build a linux/arm64 release, install
Marionette on a supported Linux with systemd using a single install script and
run it as a service after boot. Reference verification will take place on a
Raspberry Pi with Ubuntu 24.x; the target host will not need Node.js, Go or
any other runtime.

## Scope

- add a versioned systemd unit `deploy/marionette.service`;
- add an install script that creates the service user and the data directory,
  installs the binary and the unit and runs `daemon-reload` + `enable --now`;
- support configuration via `/etc/default/marionette`, in particular
  `MARIONETTE_CONFIG` and `MARIONETTE_ADDR`;
- set safe permissions on the data directory and the configuration file;
- add a release Make target for Raspberry Pi `linux/arm64` and verify that
  the artifact is static/has no runtime dependency for the target UI;
- update the README and `docs/devops/ci-cd.md` with the install and rollback
  procedure.

Out of scope:

- TLS termination, reverse proxy and authentication;
- changes to the HTTP API or the application configuration;
- automatic upload of artifacts to GitHub Releases;
- Debian/RPM package and support for init systems other than systemd.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: Deployment is to support generic Linux with systemd;
  Ubuntu 24.x on Raspberry Pi ARM64 serves as the reference validation
  environment.

## Proposed solution

- `deploy/marionette.service` will run as the dedicated user `marionette`,
  will have `WorkingDirectory=/var/lib/marionette`, will load
  `/etc/default/marionette` and set `Restart=on-failure`.
- The install script will accept a path to the binary or use the artifact
  `bin/marionette-linux-arm64`, verify the platform/ELF format, create
  `/usr/local/bin/marionette` and preserve the existing configuration.
- Default data will be in `/var/lib/marionette/marionette.json`; the
  configuration will be readable only by the service user. The script will not
  overwrite existing data without an explicit option.
- The Makefile will get a release target that builds the UI, cross-compiles
  the binary and creates a distributable archive containing the binary, the
  unit, the default config and the install script.

## Test plan

- verify `make build` and `make build-arm64`;
- verify the release artifact with `file`/`readelf` and the absence of a
  dynamic runtime dependency;
- shell test of the install script in a temporary root or container without
  touching the host's `/etc` and `/var`;
- on Ubuntu 24.x verify `systemd-analyze verify`, start after installation,
  `/api/health`, restart after the process is terminated and preservation of
  the configuration;
- the service configuration design must compare `/etc/default`, a systemd
  drop-in or another suitable mechanism and choose the option that ports best
  to generic systemd Linux;
- on the ARM64 validation host verify that the resulting artifact runs and
  that the embedded SPA is available without Node.js;
- verify rollback to the previous binary without deleting `marionette.json`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `deploy/marionette.service` passes `systemd-analyze verify`;
- the install script is idempotent and does not overwrite existing
  configuration;
- `make release-arm64` creates a documented linux/arm64 artifact for
  Raspberry Pi as well as generic ARM64 Linux;
- after a reboot or a systemd restart the service comes up automatically and
  the local health endpoint responds;
- the documentation contains the install, update and rollback procedure.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make release-arm64`, `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`, `bash -n deploy/install.sh` and `git diff --check`
  passed. The archive contains the binary, the unit, the default
  configuration, the JSON template and the install script; the binary is
  ELF64 AArch64 and static. A staged installation verified idempotence and
  preservation of the existing configuration.
- **Verification in the devcontainer (x86_64, 2026-09-26)**: `sudo
  ./deploy/install.sh bin/marionette` (a real, not staged, installation)
  revealed a bug — the binary was installed to `/usr/local/marionette`,
  while the unit starts `/usr/local/bin/marionette`; fixed
  (`binary_target=$prefix/bin/marionette`). After the fix: `systemd-analyze
  verify /etc/systemd/system/marionette.service` (systemd 252 installed as a
  package, without a running PID 1) passed with no findings. The service was
  started with the unit's environment (`runuser -u marionette`,
  `EnvironmentFile`, `WorkingDirectory`, `umask 077`): `GET /api/health`
  → `{"status":"ok","version":"472dae3"}`, SPA `lang="en"`, `POST /api/cards`
  with `Origin: https://evil.example` → 403, `Content-Type: text/plain` → 415,
  TLS proxy scenario (`Origin: https://pi.local`, `Host: pi.local`) → 201
  (block 0036), primary action → 202 and a record in `runs`,
  `marionette.json` `0640 marionette:marionette`. SIGTERM: steps "stop HTTP
  server", "save history", "close action queue", "stop status scheduler" in
  the log, history in the file; after a restart both the card and the run
  are loaded. A third run of the installer (update/rollback by the same
  procedure) preserved `marionette.json` and `/etc/default/marionette`. In
  the container the installer reports "installed and started" because
  `systemctl` here is a shim returning 0 — this does not hold on a host with
  systemd.
- **Automated installer test (2026-09-26)**: added
  `deploy/install_test.sh` (`make deploy-test`, part of `make test`, CI job
  `backend`): staged installation into a temporary `DESTDIR`, checking paths
  against the unit (`ExecStart`, `EnvironmentFile`,
  `WorkingDirectory`/`ReadWritePaths`, `MARIONETTE_CONFIG`), permissions
  755/750/640, preservation of the configuration and defaults on repeated
  runs and rejection of a non-ELF file. Verified that the test fails with
  the original binary path bug. The test revealed a second bug: the binary
  format check depended on `file(1)`, which minimal images (including the
  devcontainer) lack, and without it the check was silently skipped;
  replaced by a dependency-free ELF header check (`7f 45 4c 46`). The
  architecture check against the host remains out of scope (a wrong
  artifact shows up when the service starts).
- **Verified on the reference Ubuntu 24.x ARM64 host (Raspberry Pi,
  2026-09-26)**: the project owner installed the release archive from commit
  `9bb21bf` (`make release-arm64`, SHA-256
  `8bea40a8…6a623b`) and, following the checklist, verified installation,
  `/api/health` with version, `systemctl restart`, `Restart=on-failure` after
  `kill -9`, startup after reboot and a WoL + ping card in the **Healthy**
  state; everything works. Detailed command outputs were not recorded.
- **Originally left to verify on the host**: real
  `systemctl enable/start/stop/restart`, `Restart=on-failure` after killing
  the process, startup after reboot, running the ARM64 artifact and the
  behavior of the hardening directives (`ProtectSystem=strict`,
  `ReadWritePaths`) with real systemd. Note: `systemd-analyze security` rates
  the unit 8.6 "EXPOSED"; further tightening (`ProtectKernelTunables`,
  `RestrictAddressFamilies`, `SystemCallFilter`…) is out of the block's scope
  and requires testing on the host.
- **Finding on Raspberry Pi (Ubuntu, 2026-09-26)**: reference scenario
  WoL + ping (FR-16) — `wakeonlan` from the service works, but the `ping`
  status action always ends with `socket: Operation not permitted` (exit 2).
  Cause: `NoNewPrivileges=true` cancels the file capability `cap_net_raw` and
  the host had `net.ipv4.ping_group_range = 1 0`. Verified with
  `systemd-run -p User=marionette -p NoNewPrivileges=true … /usr/bin/ping`;
  remedy `ping_group_range = 0 2147483647`. The unit does not change;
  detection in the installer and documentation are in the new block
  [0046](0046-install-unprivileged-ping-check.md) (FR-05), displaying the
  status check output in the UI in block
  [0047](0047-status-check-output-in-detail.md) (FR-21a).
- **Documentation updated**: README, `docs/devops/ci-cd.md`, requirements,
  testing strategy and roadmap.
