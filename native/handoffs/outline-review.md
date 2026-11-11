# Independent outline contract and documentation review — 2026-11-08

Scope: read-only review of `handoffs/outline-contract.md`, the current editor
contract, relevant app/view/format lifecycle code, and the final documentation
checkpoint. No product, test, or documentation source was modified. No broad
test, build, benchmark, or package-wide verification was run.

## Verdict

The proposed additive outline contract is directionally sound and already
fixes the major integration hazards: separate `AnchorID` and outline `ID`,
an adapter-level light controller for source-only discovery, bounded labels,
fallible collection, an error-returning navigator, explicit partial-item
disclosure, and source-versus-split semantics for negative offsets. The
remaining material issues are small, additive specification gaps rather than
a need for redesign.

## Material contract findings

1. Include buffer identity in every outline cache/version key.

   The proposal says the version is `text.Buffer.Version()` and that the app
   captures tab identity and buffer version. That is not enough if the host
   can replace its `*text.Buffer`: a fresh buffer can carry the same numeric
   version as the previous one. The view/controller's internal outline cache
   must therefore use the buffer pointer as well as its version, not version
   alone. The app lifecycle should explicitly revalidate the captured buffer
   pointer after Flush and before reusing an outline snapshot.

2. Specify `AnchorID` as optional and never fabricate it.

   The contract distinguishes `AnchorID` from `ID`, which is correct, but it
   should explicitly allow an empty `AnchorID` when a format has no link-anchor
   representation. Empty must mean “no anchor,” while empty outline `ID` should
   remain invalid. This avoids tempting implementations to reuse a slug or the
   empty string as a synthetic item identity.

3. Define `Offset` exactly as one UTF-16 point.

   State whether `Offset` is the UTF-16 offset of the heading block start or
   the heading text start after the `#` marker, then use that definition for
   Markdown, source reveal, and current-item matching consistently. The
   smallest useful choice is the block start: it is available from
   `Block.Range.Start`, survives marker-only edits without moving the caret to
   mid-heading text, and is naturally converted with `text.UTF16Len` from the
   original source prefix. If Astra intentionally chooses the text start, that
   choice should still be documented and tested with CRLF, astral characters,
   and surrogate boundaries.

4. Make readonly hidden-node navigation a distinct, typed limitation.

   The current wording requires an “explicit limitation,” but a plain error
   string can collapse missing/stale, unsupported, and readonly-hidden cases
   into the same user-facing “not found.” Since mindmap source data can list
   hidden nodes without a canvas, navigation should return distinguishable
   results for missing/stale target, no source position in source-only mode,
   and a hidden target in a readonly document. Writable uncollapse remains one
   undoable transaction that preserves metadata and view state; readonly must
   not mutate the file.

## Non-findings

- Duplicate Markdown headings no longer need a contract redesign: separating
  link `AnchorID` from snapshot-stable outline `ID` is the right fix.
- Source-only discovery is addressed by the proposed adapter-level controller;
  the important implementation rule is that it must not construct the preview
  or graph canvas.
- `OutlineSnapshot() (Snapshot, error)` plus non-empty partial items and
  `Problems` is sufficient to distinguish unusable collection from a partial
  but useful outline.
- Negative offsets now have the right mode-specific behavior: split can still
  navigate the provider pane, while source-only requires an actionable switch
  to reading/split.
- Label clipping is materially honest as currently proposed: the row,
  tooltip, and accessibility label use the same bounded text, and the UI does
  not secretly retain an unbounded full-label copy.
- Observational collection is adequately specified: no Flush, draft commit,
  selection/page/pan/zoom mutation, or cache release during outline snapshots.

## Documentation audit

The final documentation checkpoint in `handoffs/docs.md` is consistent with the
accepted 12:55 integration evidence and 12:56 process-resource evidence.

- `native/VALIDATION.md` records the same final executable metadata
  (13,084,160 bytes; SHA-256
  `8ADB375D3AE400B57AA07C73E441D4A13DACE7ADC7124F5F0F38DD457FDC98B9`),
  full test/vet/build commands, working-set/private-byte samples, tester
  allocation measurements, fixture sizes, and outstanding OS-level checks.
- The documentation does not claim a matched Tauri/native memory comparison,
  runtime HarfBuzz use from the Windows build, legal-risk immunity, complete
  format parity, real Windows IME candidate-window verification, or complete
  OS network verification.
- Root `README.md`, `ROADMAP.md`, `CHANGELOG.md`, ADR0063, and the ADR index
  clearly separate the current native evidence from legacy Tauri history.
- Third-party notices identify the actual vendored/build dependency scope,
  retained purego Go-derived BSD notices, the MyGo startup-cleanup patch, and
  the distinction between module-cache contents and distributed packages.

No material documentation discrepancy was found in the reviewed final
checkpoint. Historical failures and superseded in-progress handoff sections are
not current claims because the accepted 12:55/12:56 checkpoints explicitly
supersede them.
