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

if command -v file >/dev/null 2>&1; then
	binary_description=$(file -b "$binary_path")
	if [[ "$binary_description" != *ELF* ]]; then
		echo "binary is not an ELF executable: $binary_description" >&2
		exit 1
	fi
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

if [[ -n "$destdir" ]]; then
	echo "staged Marionette installation under $destdir"
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
