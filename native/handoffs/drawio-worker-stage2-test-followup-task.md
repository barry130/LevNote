# Stage2 test and evidence followup

ACTIVE2026-11-10 for DeepSeek app owner. Root read frozen result/source/tests and
viewed both actual760x480 images (controls fit). Preserve original stage2 result,
final logs/captures/hashes. Stage2 production is frozen for independent review;
own new/changed app test coverage and followup handoff/evidence only unless a
concrete failing test proves a source defect, then report and fix narrowly.

Two concrete evidence issues to resolve:
1. App subcase called A-B-A starts pageB then only clicks previous once (B->A).
   Either accurately rename or complete B->A->B with actual navigation and epoch
   increments; exact return-to-start invalidation matters, do the latter.
2. App tests currently prove clean second-page capture but don't wire a real
   pending graph field through Flush -> Prepare -> snapshot. Add real pending
   composition/field on selected second page, export, assert correct post-Flush
   snapshot/page and exactly one intended history transaction. Original disk/BOM
   unchanged until save. Add a real conflicting field draft refusal retaining
   draft/content/history if not already inherited from accepted graph tests.
   Use existing Tester graph field helpers, avoid implementation-mirroring mocks.

Narrow final-post-dialog view replacement and same-buffer version-change cases
should be included in existing stale table if not already inherited meaningful
coverage; no matrix expansion. Verify stale result has no file and payload slot
cleared. Existing editor/name cases added correctly. Don't repeat unchanged
graphview/editor suite/captures. One final app test/vet after correction, raw
commands/streams/exits retained. Preserve any actual earlier failed attempts or
state precisely if not retained; never assert 'one run' if focused runs occurred.

Publish `drawio-worker-stage2-test-followup-result.md` with actual final test hash,
case truth and scope. Correct original result's wording via new followup only:
PrepareExportRequest is NOT side-effect-free (may synchronize); only token getter
is pure. Synthetic runner publication asserts forwarding bytes; actual SVG
renderer exact oracle belongs lower-layer/real-worker tests, don't conflate.
Freeze after followup. No .19 execution yet.
