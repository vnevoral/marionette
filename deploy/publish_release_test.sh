#!/usr/bin/env bash
# Test of publish_release.sh (blocks 0069, 0077) with a fake gh on PATH: a
# new tag is created with its notes file, an already published release is
# skipped, a draft, a missing asset, missing or empty notes or a gh error
# fails without creating anything. Never calls GitHub.
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
publish_release="$script_dir/publish_release.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() {
	echo "publish_release_test: $*" >&2
	exit 1
}

# The fake gh answers "release view" from $FAKE_VIEW (the --jq output) and
# $FAKE_VIEW_EXIT, and records every "release create" in $work/calls.
mkdir "$work/fakebin"
cat >"$work/fakebin/gh" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
"release view")
	echo "$FAKE_VIEW"
	exit "$FAKE_VIEW_EXIT"
	;;
"release create")
	echo "$*" >>"$FAKE_CALLS"
	;;
*)
	echo "fake gh: unexpected $*" >&2
	exit 99
	;;
esac
EOF
chmod +x "$work/fakebin/gh"
export PATH="$work/fakebin:$PATH" FAKE_CALLS="$work/calls"

tag=v1.2.3
archive="marionette-$tag-linux-arm64.tar.gz"
mkdir -p "$work/repo/bin"
touch "$work/repo/bin/$archive" "$work/repo/bin/$archive.sha256"
mkdir -p "$work/repo/docs/releases"
echo "## Highlights" >"$work/repo/docs/releases/$tag.md"

run() {
	FAKE_VIEW=$1 FAKE_VIEW_EXIT=$2
	export FAKE_VIEW FAKE_VIEW_EXIT
	rm -f "$work/calls"
	(cd "$work/repo" && "$publish_release" "$tag") >"$work/out" 2>&1
}

expect_ok() {
	run "$1" "$2" || fail "$3: exited non-zero: $(cat "$work/out")"
}

expect_failed() {
	if run "$1" "$2"; then
		fail "$3: succeeded, want a failure"
	fi
	[[ ! -e "$work/calls" ]] || fail "$3: called gh release create"
}

expect_ok "release not found" 1 "new tag"
want="release create $tag bin/$archive bin/$archive.sha256 --verify-tag --title Marionette $tag --notes-file docs/releases/$tag.md --generate-notes"
[[ "$(cat "$work/calls")" == "$want" ]] || fail "new tag: got '$(cat "$work/calls")', want '$want'"

expect_ok "false $archive $archive.sha256" 0 "published release"
[[ ! -e "$work/calls" ]] || fail "published release: called gh release create"
grep -q "already published" "$work/out" || fail "published release: no notice"

expect_failed "true $archive $archive.sha256" 0 "draft release"
expect_failed "false $archive" 0 "release without the .sha256"
expect_failed "false $archive.sha256" 0 "release without the archive"
expect_failed "HTTP 401: Bad credentials" 1 "gh error"

(cd "$work/repo" && "$publish_release" --check-notes "$tag") >"$work/out" 2>&1 ||
	fail "--check-notes: exited non-zero: $(cat "$work/out")"
[[ ! -e "$work/calls" ]] || fail "--check-notes: called gh"

printf ' \n\t\n' >"$work/repo/docs/releases/$tag.md"
expect_failed "release not found" 1 "empty notes"
grep -q "is empty" "$work/out" || fail "empty notes: no message"
if (cd "$work/repo" && "$publish_release" --check-notes "$tag") >/dev/null 2>&1; then
	fail "--check-notes: accepted empty notes"
fi

rm "$work/repo/docs/releases/$tag.md"
expect_failed "release not found" 1 "missing notes"
grep -q "write the release notes" "$work/out" || fail "missing notes: no message"
echo "## Highlights" >"$work/repo/docs/releases/$tag.md"

rm "$work/repo/bin/$archive.sha256"
expect_failed "release not found" 1 "missing .sha256 file"

echo "publish_release_test: ok"
