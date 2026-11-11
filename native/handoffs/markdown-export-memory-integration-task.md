# Markdown bounded HTML integration — active

Status: ACTIVE after root accepts focused checkpoint
A572ED95AFEB9F1D8089A235CDF4301BDF80B015DC37A715884CB06F59BB778F,
actual source/oracles/logs and same-fixture measurements. Sole integrator:
astra_marker_view. Root plans/reviews; all execution delegated. Do not spawn.
No further optimization is needed after optimized same-fixture comparison:
about80% lower escaped-fence allocation at observed12.8% time tradeoff; mixed
about66.8% lower allocation with similar time. Preserve all intermediate results.

## Freeze and minimal consumer gate

- Freeze current renderer/worker implementation and tests after root acceptance
  of focused evidence. No source changes except demonstrated review findings.
- Run full `./modules/markdownview` and `./app` tests once, plus focused current
  worker `TestWorkerTinyOutputBudgetFraming|TestWorkerParserCapAndParity` after
  final count optimization. Existing8MiB worker test already ran before the
  narrow count change; do not repeat it by default. All exporter tests passed
  after that change. Vet relevant consumers; exporter/worker vet already passed.
- No unrelated runtime lifecycle, missing/corrupt copies, screenshots, hidden
  launch or entire repository suite replay for this internal generation change.
  Existing source/app/input/close safety behavior remains covered by consumers.

## Fresh versioned package and real worker app path

- Update portable README template BEFORE packaging: output budget is checked
  during counting, result buffer is allocated exactly once, process memory is
  still not capped. No claim of future acceptance. This template/doc edit is
  explicitly owned by integrator; package scripts otherwise stay frozen.
- Package version `native-worker.20261109.2` into a new clean running directory
  with separate fresh evidence via accepted package-markdown-worker.ps1.
  Preserve both .1 directories, original markerexe and all prior evidence.
  Capture actual commands/streams/exits and exact new host/worker/manifest hashes.
- Compile existing app tester with working `test -c` wrapper into a disposable
  evidence bundle copy beside copies of all3actual new payload files. Do not put
  tester or logs into user output or change an accepted package. Run only
  TestWorkerExportBundledHostPathAcceptance with explicit opt-in and literal
  string-array arguments; exact existing source/HTML/warning/no-overwrite oracles.
- Verify clean file set,18release-manifest entries/19files, license copies and
  local links, frozen source and preserved old hashes. Keep validation concise,
  reuse reviewed scripts where appropriate without rewriting old artifacts.
  Do not add arbitrary assertions with hardcoded unrelated inventory counts.

## Gate and documentation

Present exact output/result handoff in markdown-export-memory.md; root reads
sources/oracles/actual logs and hashes before accepting new package. Then final
README/VALIDATION/ARCHITECTURE/API/ROADMAP/CHANGELOG updates only. State the
observed allocation tradeoff and distinguish cumulative B/op from process RSS.
Keep historical post-render-limit descriptions labeled historical, not silently
rewritten as if the old binary had the new behavior. Physical OSIME/GPU/dialog,
resource matrix, full parity and arbitrary executable plugin loader remain out
of this stage. No new program launch is required merely for the changed renderer.
