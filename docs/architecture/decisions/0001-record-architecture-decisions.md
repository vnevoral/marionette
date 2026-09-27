# ADR-0001: Decisions are recorded as ADRs in the repository

- **Status**: Accepted
- **Date**: 2026-09-25

## Context

Development is driven primarily by AI coding agents together with the
project owner. So that architecture decisions are traceable, justified and
not reopened without context, we need a single place and format for
recording them.

## Decision

Architecture decisions are recorded as separate Markdown files in
[docs/architecture/decisions/](.) in the format `NNNN-short-name.md` per
the [template](template.md), numbered in ascending order. New ADRs are
proposed with the `/new-adr` prompt, and until they are marked as
"Accepted", implementation must not build on them.

## Considered alternatives

- Recording decisions only in the README/requirements — rejected, it lacks
  the history and rationale of individual decisions over time.
- Using no formal process — rejected, AI-agent development risks
  inconsistent/contradictory decisions between individual sessions.

## Consequences

- Every non-trivial architecture decision (choice of library, persistence
  format, communication protocol, etc.) must have a corresponding ADR.
- AGENTS.md points agents to the ADR log as the source of truth about "why".
