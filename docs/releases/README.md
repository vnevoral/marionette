# Release notes

Every release has a hand-written notes file `docs/releases/vX.Y.Z.md`,
merged to `main` before the tag is pushed. The Release workflow fails
without it (block 0077), and the file becomes the text of the GitHub
Release, followed by GitHub's generated list of changes.

The notes are for the people who run Marionette, not for its developers.
Use the sections that apply, in this order:

- **Highlights**: what is new or better, in a few bullets.
- **Breaking changes**: what stops working or behaves differently, and
  who is affected (configuration, API clients, installation). Any MAJOR
  release has this section.
- **Upgrading**: the steps beyond the usual installation, if any.
- **Fixes**: bugs that users could notice.
- **Under the hood**: dependency, build and CI changes worth knowing
  about, kept short.

Do not include the version heading; the release title is
`Marionette vX.Y.Z`.
