#!/usr/bin/env bash
# Test of release_version.sh (block 0051) in a temporary git repository:
# only a clean tree whose HEAD carries a vMAJOR.MINOR.PATCH tag yields a
# release version. Never touches the real repository or the global git config.
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
release_version="$script_dir/release_version.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() {
	echo "release_version_test: $*" >&2
	exit 1
}

# Isolate git from the user's and system configuration.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
repo="$work/repo"
git init -q "$repo"
g() {
	git -C "$repo" -c user.name=test -c user.email=test@example.invalid -c commit.gpgsign=false -c tag.gpgsign=false "$@"
}

expect_version() {
	local want=$1 got
	got=$("$release_version" -C "$repo") || fail "$2: exited non-zero, want $want"
	[[ "$got" == "$want" ]] || fail "$2: got $got, want $want"
}

expect_refused() {
	local output
	if output=$("$release_version" -C "$repo" 2>&1); then
		fail "$1: accepted as version $output"
	fi
	grep -qF 'git tag -a vX.Y.Z' <<<"$output" || fail "$1: no hint how to tag, got: $output"
}

echo one >"$repo/file"
g add file
g commit -q -m one
expect_refused "commit without a tag"

g tag -a v1.2.3 -m "release v1.2.3"
expect_version v1.2.3 "clean tree with annotated tag"

# Uncommitted changes, tracked or untracked, block the release.
echo change >>"$repo/file"
expect_refused "modified tracked file"
g checkout -q -- file
echo new >"$repo/untracked"
expect_refused "untracked file"
rm "$repo/untracked"
expect_version v1.2.3 "clean tree again"

# A commit after the tag is not a release.
echo two >>"$repo/file"
g commit -q -am two
expect_refused "commit after the tag"

# Tags of another shape are ignored.
g tag test
expect_refused "tag test"
g tag v1.2
expect_refused "tag v1.2"
g tag v1.2.3-rc1
expect_refused "tag v1.2.3-rc1"

# A lightweight tag of the right shape is accepted.
g tag v1.3.0
expect_version v1.3.0 "lightweight tag"

if "$release_version" -C "$work/missing" >/dev/null 2>&1; then
	fail "accepted a directory that is not a repository"
fi

echo "release_version_test: ok"
