# Definition of Done (implementation block)

This list is the exit gate of the shared workflow described in
[development-workflow.md](development-workflow.md). A block can be marked
as `Done` only after all items are met.

A block is done when all of the following items are met:

- [ ] Before implementation the block was in the `Approved` status, and
      during the work it is in the `In progress` status.
- [ ] The code matches the approved proposal in the block
      (`docs/plans/blocks/...`); if the proposal changed during
      implementation, the block is updated.
- [ ] `make verify` passes without errors (UI build, `golangci-lint`, `eslint`
      with `--max-warnings 0`, `vue-tsc`, `prettier --check`, `go test -race`,
      `go build`, `go vet`). It is the single definition of the validation
      suite; partial commands (`make test`, `make lint`) are only for fast
      iteration. A block that changes a UI flow or the HTTP contract also
      runs `make e2e`.
- [ ] New/changed domain logic has unit tests covering both the main and
      the error scenarios (see [testing-strategy.md](testing-strategy.md)).
- [ ] The public API/behavior is documented (a comment on the exported
      symbol, or an update of `docs/architecture/overview.md`).
- [ ] The requirements (`requirements.md`) and ADRs the block refers to
      agree with the actual implementation; any discrepancies are resolved
      (by updating the document, not by silently deviating).
- [ ] `docs/plans/roadmap.md` has an updated status for the relevant
      phase/block.
- [ ] After verification the block is marked as `Done` and the validation
      commands run, or the reason for an exception, are recorded.
- [ ] No extra functionality beyond the approved scope of the block (out of
      scope items are handled as a new block, not "in passing").
