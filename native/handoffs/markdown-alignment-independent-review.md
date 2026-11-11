# Markdown alignment independent review

Status: COMPLETE, READ-ONLY, 2026-11-09.

Scope: current `native/formats/markdown/table_alignment.go` and its tests; the
shared native action pipeline in `native/modules/markdownview/table_actions.go`;
alignment UI/planner in `native/modules/markdownview/table_alignment.go`;
cell planning and source identity in `table_cell.go`; row action compatibility
in `table_rows.go`; draft/refresh/release lifecycle in `draft.go` and `view.go`;
the changed native tests and the final focused handoff/evidence. No commands
that test, build, launch, or use the network were run.

**No actionable findings.** In particular:

- The pure helper accepts only the four existing alignment values, requires a
  supported full-table identity, uses the parser's delimiter scanner, and makes
  an exact marker-only patch. Semantic no-op returns only `Noop:true` without a
  candidate.
- Positive growth is bounded before candidate construction by delimiter cell,
  table, and document limits. The candidate is parsed completely and compared
  for selected-table shape, selected-column alignment, all other rows/cells,
  all other blocks/inline/doc-block semantics, warning identity, and expected
  offset translation. Existing fallback and global descriptor state is checked.
- Native capture includes source/draft buffer, versions, selections, column,
  current alignment, release epoch, and lifecycle epoch. It rejects readonly,
  foreign groups, and every same-source/draft identity change before/after Flush.
  The ordinary cell Flush receipt is exact: one lifecycle epoch advance, expected
  version/selection, predicted table bytes, no replacement draft/cell, readonly
  or foreign group.
- A successful cell write and alignment edit form separate undo groups. Clean
  and post-cell semantic no-op complete the panel without adding source history,
  a version, or `Changed`. Post-cell alignment refusal preserves the already
  written cell and reports `单元格已写回，对齐未修改`.
- `Changed`/`Report` synchronous release/replacement/in-place edits do not
  receive stale captured state or overwrite the replacement's local error.
  Foreign groups remain owned by their callers.
- Rapid continuation clicks, held key repeats, scroll, cancel, release, and
  stale source/draft changes cancel the action without mutating the source.
  Settled frames do not retain captures or schedule extra work.
- Existing row behavior remains compatible through the shared action seam;
  row-specific header/error semantics are preserved.

The remaining unverified areas are the already separate gates: real app
760x480 split consumer behavior/captures, physical OSIME, package payload, and
resource claims. Those omissions are scope limits, not findings.

Evidence consulted: `native/handoffs/markdown-table-alignment.md` and
`native/artifacts/acceptance/markdown-alignment-20261109-053920-owned-final-attempt05/result.json`
plus its test log. The two attempt02 failures were treated as corrected test
fixture issues, not product findings; the final attempt05 PASS log contains the
fixed oracles.
