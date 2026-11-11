# Image keyboard source review notes

2026-11-10 WIP read by root; current owner resolves narrowly before freeze.

1. The integrated app test repeats the same failed HasText("125%") assertion.
   The zoom ui.Text has Label("当前缩放比例"); Tester.Texts returns that label
   instead of displayed text, as the failure output shows. Use public Tester
   value/snapshot semantics or an actual image-geometry observation to verify
   zoom; don't change the runtime label or weaken the behavior assertion merely
   to make HasText pass. Avoid another unchanged repeat of the failing test.

2. In TestImageKeyboardFocusAndInputContainment, after typing/composition in the
   adjacent field the assertion only rejects fitMode=true. Capture the viewport
   before leaving canvas and assert it stays equal while the field receives
   commands (zoom/pan could otherwise change while fitMode stays false).

3. The sub-floor fit fixture uses height32768, beyond the admitted8192 side.
   Prefer an8192high narrow fixture and a short valid test viewport that fits it
   below2%, or explicitly label it as injected geometry-only coverage. Do not
   imply the product admits32768pixel images. Existing view geometry remains
   unchanged; do not broaden raster admission for this test.

4. Frozen evidence must include the actual invoked Go commands, exit status and
   all failed attempts, not only a PASS tail. Final combined suite/vet still
   waits for both owners' source freeze; root will assign one runner.
