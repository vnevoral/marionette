---
description: "Breaks down a roadmap phase into a concrete implementation block"
name: "Plan a block"
agent: "agent"
---

Break down the given phase/topic from
[docs/plans/roadmap.md](../../docs/plans/roadmap.md) into one or more
concrete implementation blocks according to the
[template](../../docs/plans/template-implementation-block.md).

Follow the [unified development workflow](../../docs/devops/development-workflow.md).
This step is planning only: it must not change source code or mark a block
as `Approved` without the owner's explicit confirmation.

Procedure:

1. Verify that the phase/topic has a clear link to FR/NFR and, where
   applicable, to accepted ADRs. If an artifact is missing, stop planning
   and propose adding it first.
2. Split the phase into blocks small enough for a single implementation
   session, with a clear goal, dependencies, in/out of scope, a proposed
   solution and a test plan.
3. Save each block as `docs/plans/blocks/NNNN-name.md` (an ascending number
   across all blocks, not per phase), with status `Proposed` and the
   approval left blank.
4. Update `docs/plans/roadmap.md` with links to the new blocks and their
   status.
5. At the end, summarize dependencies, open questions and the exact step
   that requires approval.
6. Do not implement code and do not approve the block within this prompt.
