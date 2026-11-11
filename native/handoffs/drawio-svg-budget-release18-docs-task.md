# .18 bounded SVG: release documentation

ACTIVE after root .18 binary acceptance,2026-11-10. Owner DeepSeek
`/root/read_native_handoffs_resource_workload_next_task_md`.
Root coordinates/accepts; no children, compact before200K.

Update only root README/ROADMAP/CHANGELOG and native README/ARCHITECTURE/
VALIDATION, plus `drawio-svg-budget-release18-docs-result.md`. No source/test/
script/package/evidence edits, tests/builds/launches/benchmarks or capture reruns.

Use actual accepted source, correction/reconciliation and .18 delivery records.
`.18` current, `.17` historical. Explain: SVG output has64MiB hard limit, count
pass refuses oversized output without creating a full output buffer, exact-fill
success retains existing bytes/warnings/selected-page behavior. Errors yield no
partial SVG. Direct API lower limits honored, above hard cap clamped, negative
refused, zero default. Plain UI failure visible through existing export error.

Output cap is not process memory cap. Source/parser/indexes/warnings/point
slices and successful exact output still consume memory, two passes repeat work.
No measured whole-app memory, speed, physical input/IME or new worker execution
claim. .18 still has THREE worker engines; draw.io export remains host-side in
this package. A fourth-worker plan is future work, not shipped capability.

Keep root README launch and feature guidance concise. Put code/inventory/hash
detail in architecture/validation, not copied into every intro. Preserve earlier
release history and truthfully label old evidence. Add current evidence links;
one scoped local-link scan suffices. Record final changed document hashes and
actual validation performed in result. Freeze for root review.

Fourth-worker source development may proceed concurrently AFTER this .18
package freeze was lifted. Document only immutable accepted .18 facts; don't
rehash active changing source and mislabel it as packaged evidence. Current
HEAD/workspace development is not part of .18.

## Narrow root review corrections (active)

Root read actual documents; before acceptance fix these specific issues only:

1. native/ARCHITECTURE.md `.17` historical paragraph currently claims104files /
   47,774,882bytes direct evidence. Those are .16. .17 actual direct evidence
   is117files /47,784,783bytes, as the preceding .18 protected-history block
   correctly states. Correct misattribution; preserve .16 values.
2. native/VALIDATION.md calls0.721s/10.850s suite the final evidence beside the
   corrected SVG hashes. Explain original full drawio+graphview passed, then
   review fixes changed emitter/tests; final drawio-only test0.641s and vet0
   validated the corrections, unchanged graphview wasn't repeated. Include the
   correction result/log link. Original duplicate successful full run remains
   history; no invented claim all final files were tested by original suite.
3. Root README first feature paragraph should be user guidance. Keep current-page
   SVG64MiB refusal/no partial file, unchanged success and honest resource
   limitation in3-4 sentences. Move MaxBytes zero/negative/above-cap/count/fill
   details into architecture (already there). Remove meta sentence saying
   '首屏不再重复…'. Preserve launch/links/history.

Update docs result/current hashes, record this targeted correction. Scan only
new/changed local links if needed. No source/build/test/identity reruns.
