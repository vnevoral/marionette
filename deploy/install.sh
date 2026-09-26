#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
binary_path=${1:-"$script_dir/../bin/marionette-linux-arm64"}
destdir=${DESTDIR:-}
prefix=${PREFIX:-/usr/local}
data_dir=${DATA_DIR:-/var/lib/marionette}
defaults_file=${DEFAULTS_FILE:-/etc/default/marionette}
unit_file=${UNIT_FILE:-/etc/systemd/system/marionette.service}
service_user=${SERVICE_USER:-marionette}
service_group=${SERVICE_GROUP:-marionette}

if [[ ! -f "$binary_path" || ! -x "$binary_path" ]]; then
	echo "executable Marionette binary not found: $binary_path" >&2
	exit 1
fi

# Check the ELF magic directly so the check does not depend on file(1),
# which minimal images do not ship.
if [[ $(head -c 4 "$binary_path" | od -An -tx1 | tr -d ' \n') != "7f454c46" ]]; then
	echo "binary is not an ELF executable: $binary_path" >&2
	exit 1
fi

if [[ -z "$destdir" && "$EUID" -ne 0 ]]; then
	echo "installer must run as root (or set DESTDIR for a staged install)" >&2
	exit 1
fi

rooted_path() {
	printf '%s%s' "$destdir" "$1"
}

binary_target=$(rooted_path "$prefix/bin/marionette")
data_target=$(rooted_path "$data_dir")
defaults_target=$(rooted_path "$defaults_file")
unit_target=$(rooted_path "$unit_file")
config_target="$data_target/marionette.json"

install -D -m 0755 "$binary_path" "$binary_target"
install -d -m 0750 "$data_target"

if [[ ! -e "$config_target" ]]; then
	install -m 0640 "$script_dir/marionette.example.json" "$config_target"
fi

if [[ ! -e "$defaults_target" ]]; then
	install -D -m 0644 "$script_dir/marionette.default" "$defaults_target"
fi
install -D -m 0644 "$script_dir/marionette.service" "$unit_target"

# default_value prints the last KEY=value line of the defaults file without
# sourcing it (the file is operator input, not a script to run).
default_value() {
	local value
	value=$(sed -n "s/^$1=//p" "$defaults_target" | tail -n 1)
	value=${value%\"}
	value=${value#\"}
	printf '%s' "$value"
}

# print_pairing_hint tells the operator how to pair the first device
# (FR-53): with access control on and no devices file yet, the UI is locked
# until the code from the service log is entered.
print_pairing_hint() {
	local config devices addr
	[[ $(default_value MARIONETTE_AUTH) == off ]] && return 0
	config=$(default_value MARIONETTE_CONFIG)
	devices=$(default_value MARIONETTE_DEVICES)
	if [[ -z "$devices" ]]; then
		devices="$(dirname -- "${config:-$data_dir/marionette.json}")/devices.json"
	fi
	[[ -e $(rooted_path "$devices") ]] && return 0
	addr=$(default_value MARIONETTE_ADDR)
	addr=${addr:-:8080}
	echo
	echo "No device is paired yet. Open http://<host>:${addr##*:}/ in a browser"
	echo "and enter the pairing code from the service log:"
	echo
	echo '  journalctl -u marionette | grep "pairing code"'
}

if [[ -n "$destdir" ]]; then
	echo "staged Marionette installation under $destdir"
	print_pairing_hint
	exit 0
fi

if ! getent group "$service_group" >/dev/null 2>&1; then
	groupadd --system "$service_group"
fi
if ! id -u "$service_user" >/dev/null 2>&1; then
	useradd --system --gid "$service_group" --home-dir "$data_dir" \
		--no-create-home --shell /usr/sbin/nologin "$service_user"
fi
chown "$service_user:$service_group" "$data_target" "$config_target"
chmod 0750 "$data_target"
chmod 0640 "$config_target"

systemctl daemon-reload
systemctl enable marionette.service
if systemctl is-active --quiet marionette.service; then
	systemctl restart marionette.service
else
	systemctl start marionette.service
fi

echo "Marionette installed and started"
print_pairing_hint
