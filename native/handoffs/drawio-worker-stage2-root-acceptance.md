# Stage2 source and portable asset acceptance

ACCEPTED 2026-11-10 by root, planning/review only. No root implementation,
test execution, build or launch was performed. This accepts source and portable
assets, not a `.19` executable. `.18` remains the current accepted runnable.

## Accepted behavior

The fourth fixed `levnote.drawio-svg` capability requires a mounted reading or
split view and an exact selected page. App admission Flushes the view, validates
ownership, version, size and group state, prepares the page/token, then validates
again before enqueueing copied metadata and a snapshot. Publication checks the
same identities before and after the save dialog. Old engines keep `Page=nil`
and have no mounted-view requirement. Stale results do not publish files.

The app panel exposes four fixed engines and count-neutral reset wording.
Source mode provides visible guidance. The portable README matches these
controls, local package import semantics, four-engine selection migration and
actual limits. None of this asserts OS sandboxing or process-memory gains.

## Evidence reconciled

- Original Stage2 result SHA256
  `19DD58AB7589CC1DBDFE7A6B454A1C28F76968D24C2A4D74A30CF3BDC9BEA283`.
  Graph12.003s/app17.788s and vet exited0. Both760x480Tester captures were
  visually reviewed in the preceding root turn and by the independent reviewer.
- Test-only followup result SHA256
  `D7DBEB458CA995454901D72307056164F3985A41116EA3C0DDA63554B7A91DCE`.
  Final app17.000s and vet exited0; root read raw final evidence, including empty
  vet streams. The failed initial compile and focused correction remain recorded.
- Final independent review SHA256
  `3068620C3B95A7213F75A375007227FF10C5EBAA75A1AF0E8B3D6ADD5D0CBF01`.
  No source blocker. Its initial missing-followup finding is resolved by a dated
  appendix, preserving the historical record.
- Root rehashed all12Stage2 source/test files;11match the original result and
  the authorized test followup matches
  `F8FF9ADBF52F1ED68161306FB4D472A91C500FC59B3A90BA24278C579F4C2BD2`.
  Exact accepted files are in `drawio-worker-stage2-files.json`.
- Portable README11,411bytes SHA256
  `0C18FEEF04A2EE146A231BCF767BDB0E57938DA902AA150D4CEFC17AB58C02E5`.
  Prepared docs and wording results remain the evidence; five scoped local
  targets were checked. No links changed in the wording correction.

Original prose is superseded where inaccurate: Prepare may synchronize cache;
only the token getter is pure. The followup proves real B-A-B navigation and a
real second-page pending field, Flush to exactly one undo transaction, exact
post-Flush source forwarding, unchanged disk/BOM and exact Undo restoration.
Synthetic output assertions prove runner forwarding/publication; exact SVG
rendering remains lower-layer and actual managed-worker evidence.

## Delivery gate

Stage1 and view-contract acceptances remain in force. Production, tests,
portable assets and accepted evidence are frozen for `.19` delivery. The
prepared harness has an outstanding root-identified automatic `$Host` variable
collision; a fresh correction is assigned. Only after that is reconciled will
root explicitly activate the prepared `.19` delivery task. Physical Windows
IME/UI and resource measurements are outside these accepted gates.
