#!/usr/bin/env bash
# Publishes the GitHub Release for a tag with the linux/arm64 archive and its
# .sha256 (blocks 0057, 0069). A release that is already published with both
# assets is left alone, so a second Release run for the same tag succeeds;
# a draft or a release missing an asset fails, so a broken release is never
# reported as published.
#
# Usage: publish_release.sh <tag> (run from the repository root after
# make release-arm64; needs gh with GH_TOKEN)
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <tag>" >&2
	exit 2
fi
tag=$1
archive="bin/marionette-${tag}-linux-arm64.tar.gz"

fail() {
	echo "publish_release: $*" >&2
	exit 1
}

for f in "$archive" "$archive.sha256"; do
	[[ -f "$f" ]] || fail "$f does not exist (run make release-arm64 first)"
done

if ! state=$(gh release view "$tag" --json isDraft,assets \
	--jq '"\(.isDraft) \([.assets[].name] | join(" "))"' 2>&1); then
	[[ "$state" == *"release not found"* ]] || fail "gh release view $tag: $state"
	gh release create "$tag" "$archive" "$archive.sha256" \
		--verify-tag --title "Marionette $tag" --generate-notes
	exit 0
fi

read -r draft assets <<<"$state"
[[ "$draft" == "false" ]] || fail "release $tag exists as a draft; publish or delete it"
for f in "$archive" "$archive.sha256"; do
	[[ " $assets " == *" ${f##*/} "* ]] || fail "release $tag exists without ${f##*/}"
done
echo "::notice::Release $tag is already published with both assets; nothing to do."
