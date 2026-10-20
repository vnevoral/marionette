#!/usr/bin/env bash
# Publishes the GitHub Release for a tag with the linux/arm64 archive and its
# .sha256 (blocks 0057, 0069). The notes are docs/releases/<tag>.md, written
# by hand; GitHub appends the generated list of changes (block 0077). A tag
# without a non-empty notes file fails before anything is published. A
# release that is already published with both assets is left alone, so a
# second Release run for the same tag succeeds; a draft or a release missing
# an asset fails, so a broken release is never reported as published.
#
# Usage: publish_release.sh [--check-notes] <tag> (run from the repository
# root after make release-arm64; needs gh with GH_TOKEN). --check-notes only
# checks the notes file, so the workflow can fail before it builds.
set -euo pipefail

check_only=false
if [[ ${1:-} == "--check-notes" ]]; then
	check_only=true
	shift
fi
if [[ $# -ne 1 ]]; then
	echo "usage: $0 [--check-notes] <tag>" >&2
	exit 2
fi
tag=$1
archive="bin/marionette-${tag}-linux-arm64.tar.gz"
notes="docs/releases/${tag}.md"

fail() {
	echo "publish_release: $*" >&2
	exit 1
}

[[ -f "$notes" ]] || fail "$notes does not exist; write the release notes and merge them before tagging"
grep -q '[^[:space:]]' "$notes" || fail "$notes is empty"
if $check_only; then
	echo "publish_release: release notes $notes found"
	exit 0
fi

for f in "$archive" "$archive.sha256"; do
	[[ -f "$f" ]] || fail "$f does not exist (run make release-arm64 first)"
done

if ! state=$(gh release view "$tag" --json isDraft,assets \
	--jq '"\(.isDraft) \([.assets[].name] | join(" "))"' 2>&1); then
	[[ "$state" == *"release not found"* ]] || fail "gh release view $tag: $state"
	gh release create "$tag" "$archive" "$archive.sha256" \
		--verify-tag --title "Marionette $tag" --notes-file "$notes" --generate-notes
	exit 0
fi

read -r draft assets <<<"$state"
[[ "$draft" == "false" ]] || fail "release $tag exists as a draft; publish or delete it"
for f in "$archive" "$archive.sha256"; do
	[[ " $assets " == *" ${f##*/} "* ]] || fail "release $tag exists without ${f##*/}"
done
echo "::notice::Release $tag is already published with both assets; nothing to do."
