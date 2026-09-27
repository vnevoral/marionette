---
description: "Records or updates one requirement in docs/requirements/requirements.md"
name: "New requirement"
agent: "agent"
---

Record a new or updated requirement in
[docs/requirements/requirements.md](../../docs/requirements/requirements.md).

Follow the [unified development workflow](../../docs/devops/development-workflow.md).
This step only prepares documentation; do not plan or implement code.
Procedure:

1. Find out from the user (if it is not clear from the request) whether it is
   a functional (FR) or non-functional (NFR) requirement and which section it
   belongs to.
2. Assign the next free ID within that section (e.g. the next `FR-1x`), do not renumber existing IDs.
3. Write the requirement concisely, unambiguously and testably (one
   sentence/paragraph), in the same style as the surrounding items.
4. If the requirement changes or cancels an existing item, do not cut the
   existing item without a trace — mark it as superseded/updated with a
   reference to the new ID, so the history is preserved.
5. If the requirement implies an architecture decision (a new dependency, a
   persistence approach, a protocol...), warn the user that an ADR should be
   created as well (`/new-adr`), but do not create it yourself without
   confirmation.
6. Check whether the requirement conflicts with an existing ADR — if so,
   point out the conflict instead of silently recording it.
7. At the end, state whether the requirement requires an ADR and an
   implementation block; do not approve either of them automatically.
