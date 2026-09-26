#!/usr/bin/env bash
# Staged test of install.sh (block 0023): installs into a temporary DESTDIR
# and checks that every path the systemd unit relies on exists there, that the
# modes match the documentation and that a second run keeps the operator's
# configuration. Runs without root and without systemd.
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
unit="$script_dir/marionette.service"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() {
	echo "install_test: $*" >&2
	exit 1
}

unit_value() {
	sed -n "s/^$1=//p" "$unit" | head -n 1
}

mode_of() {
	stat -c '%a' "$1"
}

# Any ELF executable passes the installer's format check.
binary="$work/marionette"
cp "$(type -P true)" "$binary"
chmod 0755 "$binary"

root="$work/root"
DESTDIR="$root" "$script_dir/install.sh" "$binary" >/dev/null

exec_start=$(unit_value ExecStart)
environment_file=$(unit_value EnvironmentFile)
environment_file=${environment_file#-}
working_directory=$(unit_value WorkingDirectory)
read_write_paths=$(unit_value ReadWritePaths)

[[ -x "$root$exec_start" ]] || fail "binary not installed at the unit's ExecStart $exec_start"
[[ -f "$root$environment_file" ]] || fail "defaults not installed at the unit's EnvironmentFile $environment_file"
[[ -d "$root$working_directory" ]] || fail "data directory $working_directory missing"
[[ "$working_directory" == "$read_write_paths" ]] || fail "WorkingDirectory $working_directory is not the writable path $read_write_paths"
[[ -f "$root/etc/systemd/system/marionette.service" ]] || fail "unit file not installed"

config="$root$working_directory/marionette.json"
config_path=$(sed -n 's/^MARIONETTE_CONFIG=//p' "$root$environment_file")
[[ "$root$config_path" == "$config" ]] || fail "MARIONETTE_CONFIG $config_path is not inside $working_directory"
[[ -f "$config" ]] || fail "default configuration not installed"

[[ $(mode_of "$root$exec_start") == 755 ]] || fail "binary mode $(mode_of "$root$exec_start"), want 755"
[[ $(mode_of "$root$working_directory") == 750 ]] || fail "data directory mode $(mode_of "$root$working_directory"), want 750"
[[ $(mode_of "$config") == 640 ]] || fail "configuration mode $(mode_of "$config"), want 640"

# An update keeps the operator's configuration and defaults.
echo '{"operator":"edit"}' >"$config"
echo 'MARIONETTE_ADDR=:9090' >>"$root$environment_file"
DESTDIR="$root" "$script_dir/install.sh" "$binary" >/dev/null
grep -q '"operator"' "$config" || fail "second run overwrote the configuration"
grep -q ':9090' "$root$environment_file" || fail "second run overwrote the defaults file"

# The first installation explains how to pair the first device.
hint='journalctl -u marionette | grep "pairing code"'
output=$(DESTDIR="$work/hint" "$script_dir/install.sh" "$binary")
grep -qF "$hint" <<<"$output" || fail "no pairing hint without paired devices"
grep -qF "http://<host>:8080/" <<<"$output" || fail "pairing hint does not name the port"

# The hint follows MARIONETTE_ADDR and disappears once devices.json exists,
# or when access control is off.
output=$(DESTDIR="$root" "$script_dir/install.sh" "$binary")
grep -qF "http://<host>:9090/" <<<"$output" || fail "pairing hint ignores MARIONETTE_ADDR"
touch "$root$working_directory/devices.json"
output=$(DESTDIR="$root" "$script_dir/install.sh" "$binary")
if grep -qF "$hint" <<<"$output"; then fail "pairing hint shown with paired devices"; fi
rm "$root$working_directory/devices.json"
echo 'MARIONETTE_AUTH=off' >>"$root$environment_file"
output=$(DESTDIR="$root" "$script_dir/install.sh" "$binary")
if grep -qF "$hint" <<<"$output"; then fail "pairing hint shown with access control off"; fi

# A file that is not an ELF executable is refused.
printf '#!/bin/sh\n' >"$work/script"
chmod 0755 "$work/script"
if DESTDIR="$work/refused" "$script_dir/install.sh" "$work/script" >/dev/null 2>&1; then
	fail "installer accepted a non-ELF binary"
fi

echo "install_test: ok"
