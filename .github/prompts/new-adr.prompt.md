---
description: "Proposes a new architecture decision record (ADR)"
name: "New ADR"
agent: "agent"
---

Propose a new architecture decision according to the
[template](../../docs/architecture/decisions/template.md).

Follow the [unified development workflow](../../docs/devops/development-workflow.md).
This step only prepares the ADR; do not implement code or create an approved
implementation block automatically.

Procedure:

1. Check the highest used number in
   [docs/architecture/decisions/](../../docs/architecture/decisions) and name
   the new file `NNNN-short-name.md` (the next number in sequence, kebab-case
   name).
2. Fill in the Context (why the decision is being made, which FR/NFR or
   problem it addresses — reference
   [requirements.md](../../docs/requirements/requirements.md)), the Decision,
   the Considered alternatives (at least 1 rejected) and the Consequences.
3. Create the new ADR with the status **Proposed** until the user explicitly
   confirms it as **Accepted** — do not implement code based on it until it
   is accepted.
4. If the decision supersedes an earlier ADR, state it in both files
   (old: "Superseded by ADR-NNNN", new: a link back).
5. Update [docs/architecture/overview.md](../../docs/architecture/overview.md)
   if the decision changes the component overview.
6. At the end, list the dependent requirements and follow-up implementation
   blocks that will need to be planned after the ADR is accepted.
