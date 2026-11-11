# Floating-point assertion after toolbar geometry change

ACTIVE 2026-11-10 for the imageview/DeepSeek owner. Pause .23 helper edits briefly.
Own only modules/imageview/imageview_test.go for the concrete failing regression,
plus separately named result/evidence. No product geometry/UI or app edits.

Integrated final-suite1 passed app and every other affected package but
TestManualResizePreservesCenterThroughTester compares computed zoom to exact1;
new viewport gives0.9999999999999999 after target/current.zoom multiplication.
Root read zoomTo/zoomAround and this failure: an ordinary floating rounding
residual, not a visual zoom error. Keep behavioral center-preservation checks.
Use explicit small tolerance for this test's computed zoom/pan comparisons,
including post-resize center clamps in the same test; don't loosen dimension,
boolean, callback, bounds, or release assertions. Explain tolerance relative to
DIP/pixel accuracy, no blanket replacement of all exact comparisons.

After editing/gofmt, publish file identity and rationale to
native-image-view-resize-assertion-result.md. Do NOT run another standalone
suite: Astra will run only the full imageview package once, then the already
planned combined vet. Its final-suite1 already passed all unchanged packages;
there is no reason to rerun those tests or regenerate captures for a test-only
correction. Preserve final-suite1 and earlier source identities/history. Resume
.23 preparation once the test source freezes; no product execution.
