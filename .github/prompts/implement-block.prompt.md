---
description: "Implements one approved implementation block incl. tests"
name: "Implement block"
agent: "agent"
---

Implement one specific implementation block from `docs/plans/blocks/`.

Follow the [shared development workflow](../../docs/devops/development-workflow.md)
and the [Definition of Done](../../docs/devops/definition-of-done.md).

Procedure:

1. If the user did not specify a block, pick the first block in the
   `Approved` state in roadmap order, or ask if the choice is unclear.
2. Do not implement a block in the `Proposed`, `Blocked` or `Rejected` state.
3. Before making changes, mark the block as `In progress` and verify its
   requirements, accepted ADRs, dependencies, approval and out-of-scope
   boundaries.
4. Implement exactly the approved scope; stop any need outside the scope and
   propose it as a new requirement, ADR or implementation block.
5. Write unit tests according to the block's "Test plan" section and
   [testing-strategy.md](../../docs/devops/testing-strategy.md).
6. Run `make verify` (the single definition of the validation suite: UI
   build, Go and web lint, `go test -race`, `go build`, `go vet`) and fix
   errors. While iterating, the partial `make test` / `make lint` can be
   used.
7. Go through the [Definition of Done](../../docs/devops/definition-of-done.md),
   update the block, requirements/ADRs, public documentation and
   `docs/plans/roadmap.md` to match reality.
8. Only after the DoD is met, mark the block as `Done`; if there is an
   obstacle, state the reason and use the `Blocked` state.
9. At the end, list the changes, verification, any deviations and unmet DoD
   items.
