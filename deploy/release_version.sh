#!/usr/bin/env bash
# Prints the release version (block 0051): the tag vMAJOR.MINOR.PATCH that
# points exactly at HEAD. Fails with a hint when HEAD has no such tag or the
# working tree has uncommitted changes, so a release always matches a tag.
#
# Usage: release_version.sh [-C <repository directory>]
set -euo pipefail

repo=.
if [[ $# -gt 0 ]]; then
	if [[ $# -eq 2 && "$1" == "-C" ]]; then
		repo=$2
	else
		echo "usage: $0 [-C <repository directory>]" >&2
		exit 2
	fi
fi

fail() {
	echo "release_version: $*" >&2
	exit 1
}

git -C "$repo" rev-parse --verify --quiet HEAD >/dev/null || fail "$repo is not a git repository with a commit"

if [[ -n "$(git -C "$repo" status --porcelain)" ]]; then
	fail "the working tree has uncommitted changes; commit them, then tag the commit with: git tag -a vX.Y.Z -m \"…\""
fi

version=$(git -C "$repo" tag --points-at HEAD | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n 1 || true)
if [[ -z "$version" ]]; then
	fail "HEAD has no release tag vMAJOR.MINOR.PATCH; create one with: git tag -a vX.Y.Z -m \"…\" (see docs/devops/ci-cd.md)"
fi

echo "$version"
