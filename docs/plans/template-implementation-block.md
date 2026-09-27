# Implementation block: {name}

- **Phase**: {phase number and name from roadmap.md}
- **Requirements**: {FR-xx, NFR-xx}
- **ADRs**: {ADR-xxxx, if relevant}
- **Status**: Proposed | Approved | In progress | Done | Blocked | Rejected
- **Dependencies**: {previous blocks, requirements or ADRs}

## Goal

One or two sentences — what will work after the block is done, and what
will not.

## Scope

- What is inside (in scope)
- What is deliberately outside (out of scope) — so the agent does not try
  to add it on top

## Approval

- **Approved by**: {user/project owner}
- **Approval date**: {YYYY-MM-DD, or "pending"}
- **Decision notes**: {optional}

## Proposed solution

Briefly: new/changed packages, public functions/types, data structures,
API contract (if relevant).

## Test plan

- Unit tests: what is tested and which edge cases
- Manual/integration verification (if automation is not possible/useful)

## Done criteria

See [Definition of Done](../devops/definition-of-done.md) plus specific
points for this block (if any).

## Closure

- **Status after implementation**: {Done | Blocked | Rejected}
- **Verification**: {commands and result}
- **Documentation updated**: {yes/no, links}
