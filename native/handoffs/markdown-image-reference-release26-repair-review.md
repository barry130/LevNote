# .26 metadata-repair consistency review

2026-11-10 16:55. Independent read-only review for
`markdown-image-reference-release26-repair-review-task.md`. No runtime or
package source, evidence, product doc, helper, test, build, launch, or product
run was changed or executed by this reviewer.

## Result

No additional actionable defect at the 16:51 helper state.

The repair/adoption path preserves the original `.26` package history as one
invocation with exit1, and both nested package wrappers also record exit1. It
does not backfill `package-result.json` or `package/package-result.json`, and
`Read-PackageGate` rejects direct/repair coexistence. The five original GUI
build records remain successful with empty streams. The source delta is pinned
to the one portable README repair and the candidate delta is pinned to
`README.md` plus regenerated `release-manifest.json`. The manifest is removed
only after its preserved failed bytes are verified, is regenerated outside
itself, and contains exactly25 payload entries with `release-manifest.json`
excluded.

At review time the adoption record itself was not yet published:
`portable-link-repair-result.json` and `repair-helper-provenance.json` were
absent. This is an expected incomplete state, not a defect. The final repair
parse was already present and passed:24 scripts,0 parse errors, exit0.

## Concrete state verified

- Final repair parse: `repair-final-parse-invocation.json`, exit0, and
  `repair-final-static-parse-result.json`, schema1, PowerShell7.6.9,24 scripts,
  `ParseErrorCount=0`, `BadScriptCount=0`, `Pass=true`.
- The earlier failed parse remains exit1 with `Pass=false`; the first correction
  parse remains exit0 with `Pass=true`. The final script does not discard either
  earlier result.
- `repair-adoption.ps1` verifies the failure inventory hash, retained direct
  evidence, archived failed README/manifest bytes, and all four archived
  original helper copies.
- Candidate-before facts are taken from `failureInventory.CandidateFiles`, so
  aggregate/closure cannot silently trust an unpinned before-state. If the live
  manifest is present, its bytes must match the archived failed manifest; if it
  was already removed, resume is accepted only for the exact approved repaired
  README identity.
- The repaired candidate README is pinned to
  `DCA36FED6CB7CBEB97D9A6C7C879C6886563703E4BB65BD5293D9C99AA692138` and the
  regenerated manifest to
  `EFF8DD2FAF05EC620AB5976C2A72E647F12570876B15B13606062B7DC9C4E20E`. The
  candidate set remains26 files; source copies, five PE payloads, and four
  engine manifests are checked against actual bytes.
- Aggregate independently recomputes the scoped source inventory and allows only
  the exact README before/after pair from the adoption record. It compares the
  live candidate identity, copied candidate receipt, host hash, and manifest
  hash against the adoption record, and retains the strict hidden-host checks:
  5000ms settling, fresh pre-close text+PNG session, one WM_CLOSE, exit0,
  Job total1/active0, process/job disposal, zero worker requests, and fresh
  normal-close publication.
- The closure helper requires `repair-parse` exit1, `repair-correction-parse`
  and `repair-final-parse` exit0, original stage/outer package exit1, and
  aggregates an explicit total across the evidence root plus all capture
  attempt siblings. It labels repair adoption as preserving the original failed
  package and propagates `ProductRerun=false`.
- The four pre-repair helpers are separately archived and verified:
  `delivery-lib.ps1`, `run-host-smoke.ps1`, `aggregate.ps1`, and
  `close-inventory.ps1`. The earlier concern that closure had no separate
  archived original is resolved.

## Files and versions inspected

- `native/AGENTS.md`
- `native/handoffs/markdown-image-reference-release26-repair-review-task.md`
- `native/handoffs/markdown-image-reference-release26-repair-activation.md`
- `native/handoffs/markdown-image-reference-release26-portable-link-failure.md`
- `native/handoffs/markdown-image-reference-release26-portable-link-failure-inventory.json`
- `native/handoffs/markdown-image-reference-release26-helper-originals-20261110/helper-originals.json`
- Active helper/evidence:
  `native/artifacts/acceptance/markdown-image-reference-release26-20261110/repair-adoption.ps1`
  (16:51 revision,29,497 bytes),
  `delivery-lib.ps1`, `run-host-smoke.ps1`, `aggregate.ps1`, and
  `repair-final-parse-*` records.
- External closure helper:
  `native/handoffs/markdown-image-reference-release26-close-inventory.ps1`
  (16:51 revision,5,707 bytes, SHA-256
  `E8ED8C27903B180CAEBBF19AD6D173A436D4BBDB6D80278AA32CFB20EBDC4A86`).
- Candidate `README.md` and `release-manifest.json` at the failed state at the
  beginning of review; these were expected to change only when the owner ran the
  reviewed adoption helper.

## Outstanding completion, not a review defect

The owner still needs to execute the reviewed `repair-adoption.ps1`, publish
`portable-link-repair-result.json`, `repair-helper-provenance.json`, and
`repair-candidate-copy`, then run the already authorized remaining strict host,
aggregate, closure, and delivery handoff. No repeated package/build/test is
needed merely because the metadata checks above are coherent.
