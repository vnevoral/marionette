# Development workflow

This document is the single normative description of the planning and
implementation process in the Marionette project. `AGENTS.md`, the roadmap,
the implementation templates, the DevOps checks and the prompts in
`.github/prompts/` use this process and must not introduce a parallel
methodology.

## Lifecycle

Every new feature or change goes through these steps:

1. **Request** — describe the problem, the user impact and the boundaries of
   the change.
2. **Requirement** — write or update a testable FR/NFR in
   `docs/requirements/requirements.md`.
3. **ADR** — if the change affects architecture, persistence, a public
   protocol, security or technology, prepare an ADR. Implementation can build
   on it only once it is `Accepted`.
4. **Roadmap** — place the work into a phase and determine the dependency
   order.
5. **Implementation block** — break the work down into a small,
   independently verifiable block per
   `docs/plans/template-implementation-block.md`.
6. **Approval** — the block must be in the `Approved` status; the `Proposed`
   status alone does not authorize implementation.
7. **Implementation** — change only the approved scope, add tests and perform
   the verification per the block and the Definition of Done.
8. **Closure** — update the status of the block and the roadmap, the
   documentation, the ADR/requirements in case of a deviation, and list the
   checks performed.

The planning steps (`/new-requirement`, `/new-adr`, `/plan-block`) do not
change source code or start implementation. The implementation step
(`/implement-block`) must not start without an approved block.

## Statuses and transitions

- `Proposed` — the artifact is ready for comments, it is not implementable.
- `Approved` — the scope, links and test plan have been confirmed.
- `In progress` — implementation of the approved block has started.
- `Done` — the Definition of Done is met and the verification is recorded.
- `Blocked` — work cannot continue because of a specific obstacle.
- `Rejected` — the proposal will not be implemented; the reason stays in the
  documentation.

The allowed regular transition is `Proposed` → `Approved` → `In progress` →
`Done`. A transition to `Blocked` or `Rejected` must state the reason. A
change of the approved scope is first reflected in a requirement, ADR or a
new implementation block; it is not added unannounced during implementation.

## Quality gates

Before implementation, verify:

- the requirement covers the requested behavior;
- the related ADR is `Accepted`, if needed;
- the block has a clear goal, a scope including out-of-scope, a proposed
  solution, a test plan, dependencies and done criteria;
- the block is listed in the roadmap and has the `Approved` status.

Before closure, verify:

- the implementation matches the approved scope;
- tests cover both the main and the error scenarios;
- `make verify` passed (the single definition of the validation suite for
  both Go and web, see `Makefile`);
- the public behavior and the documentation match reality;
- the roadmap and the block status are updated.

The detailed closure checklist is in the
[Definition of Done](definition-of-done.md). If a check fails, the block is
not `Done`.

## Deviations and urgent fixes

When implementation reveals an unresolvable conflict or a need outside the
scope, stop the change, describe the impact and create a new requirement,
ADR or block. For an urgent fix, a minimal intervention before full planning
is allowed only to restore functionality or security; the missing artifacts,
tests and a reference to the reason for the exception are added immediately
afterwards.

Historical ADRs, requirements and completed blocks are not changed because of
a new methodology without preserving history. Only factual inconsistencies,
links and status data needed for consistency are corrected.
