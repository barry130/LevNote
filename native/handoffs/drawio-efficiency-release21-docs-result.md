# `.21` current docs result

Status: completed and frozen for root review, 2026-11-10. This was documentation
and handoff copyediting only; no product source, test, build, package, benchmark,
launch, or capture was run.

## Changes

- Root README, ROADMAP, CHANGELOG, native README, ARCHITECTURE, VALIDATION, and
  format API now identify `.21` as the current accepted runnable package and
  keep `.20` and earlier packages clearly historical.
- Main README now states the concrete draw.io warning and source-ownership
  outcomes, the current launch path, basic limits, and links to detailed evidence.
- Detailed package identities, exact fixture hash, allocation medians, worker and
  hidden-host runs, retry/failure disclosure, and evidence links are recorded once
  in `native/VALIDATION.md`.
- The benchmark fixture hash uses the exact 64-hex raw-log value
  `04E533B0DA533713F51E106F820A7F4A62D0E0F06CA910EB6FEAA9857A3D3E9F`.
  `B/op` and `allocs/op` are described only as cumulative per-operation allocation.
- The `.21` delivery handoff now distinguishes the 202-file direct evidence root
  from the separate 4-file aggregate-retry sibling, clarifies that Job total
  processes 1 means no worker launched, repairs its relative evidence links, and
  fixes the four identified word glitches.

## Validation

One scoped markdown-link check was run after the final edits. It initially found
one incorrect relative path in `native/ARCHITECTURE.md`; that link was corrected
and the final check had no missing local targets. No test, build, package,
benchmark, launch, or capture was executed.

## Final file identities

## Root correction

After root review, three factual corrections were applied from the frozen
records: the aggregate SHA-256 was corrected to
`E34D2FE39096978E88FB5DB2B868EEB9B96315EA141D73E879B0910F191D1FB8`; the
delivery handoff now distinguishes the 202-file direct root from the separate
4-file retry sibling and states Job TOTAL processes 1 with no owned process
remaining after exit; and `VALIDATION.md` records the observed slower timing
medians while withholding any speed conclusion. No product, evidence, test,
build, or link-scan action was performed.

- `README.md` — 14,955 bytes —
  `733B20AA303EC17C5F1C199193354F04B252C52BA0731318224C1C58EF62C4DA`
- `ROADMAP.md` — 119,817 bytes —
  `DB4016AC418C15BC884112C58BA6591E02B4A1BDB23587D2B43D6B369E32C032`
- `CHANGELOG.md` — 146,552 bytes —
  `10C9C2D0334B91F52A5521A58C0A38BF5E403417F2195FF41A765B1E2D1E4C83`
- `native/README.md` — 61,024 bytes —
  `E1562B356E140B571D5A006E625F835E2CCD7DB185D65D40B838BD0BE8B0AD47`
- `native/ARCHITECTURE.md` — 62,996 bytes —
  `7FA42E4D3DE79CC512732C70A229F1DB6040528E817F6080102A6840D237956C`
- `native/VALIDATION.md` — 145,193 bytes —
  `72FD51BC1DBC16A8FA824BC38C1367660D3DFD9B07C71EED7A642C1A5A2FCEAA`
- `native/formats/API.md` — 54,437 bytes —
  `28BB4954F998302F7F039D7D8D806794B53FDD64D5E0DFEA779E23616E151186`
- `native/handoffs/drawio-efficiency-release21-delivery.md` — 5,981 bytes —
  `D8A937BF2D4D17FCA02E16FB0F60FFB4DFBF7C5A56FC8CFC0E05C608E2A12232`

No physical UI/IME, packaged interaction, process-memory, speed, installer, or
cross-platform claim is added. Future CtrlTab work remains outside this package.
