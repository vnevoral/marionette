# Marionette project documentation

This directory is the source of truth for requirements, architecture
decisions and the implementation plan. The code should not diverge from it —
if it does, the documentation is updated first (new requirement / ADR), then
the code.

- [requirements/requirements.md](requirements/requirements.md) — functional
  and non-functional requirements (SRS), source of truth for FR/NFR
- [requirements/glossary.md](requirements/glossary.md) — domain terms
- [architecture/overview.md](architecture/overview.md) — overview of
  components and their relationships
- [architecture/decisions/](architecture/decisions) — log of architecture
  decisions (ADR)
- [plans/roadmap.md](plans/roadmap.md) — project phases and implementation
  blocks
- [devops/testing-strategy.md](devops/testing-strategy.md) — how testing is
  done
- [devops/ci-cd.md](devops/ci-cd.md) — CI pipeline and release process
- [devops/development-workflow.md](devops/development-workflow.md) — the
  binding shared planning and implementation process
- [devops/definition-of-done.md](devops/definition-of-done.md) — exit check
  for when a block is done

## How the repository works with AI agents

See [AGENTS.md](../AGENTS.md) in the repository root — the main entry point
for agents. The binding process is in
[devops/development-workflow.md](devops/development-workflow.md) and
`.github/prompts/` contains its repeatable steps.
