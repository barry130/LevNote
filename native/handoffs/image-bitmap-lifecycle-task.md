# Image bitmap lifecycle prerequisite

ACTIVE2026-11-10 for Astra read_native_handoffs_tab_workflow_resume_task_md.
.22 delivery and docs are accepted/frozen. No children. Own this read-only
vendor lifecycle design; GLM binary owner now handles core/document contract.
No product/vendor edits/tests/build/launch yet. Avoid duplicating binary design.

Read native-image-view-next-plan.md and binary-image-api-review-guidance.md.
User's low-memory objective needs a concrete GPU resource release solution.
MyGo Bitmap has ReleasePixels but no disposal; normal D3D image textures age
only after >240unused frames in a120frame sweep. Do not force redraw loops or
claim that GC nils driver resources. Inspect full actual UI/render threading,
scene.Image identity and CPU/GPU caches, canvas dependencies and other backends.

Propose the smallest public terminal bitmap-retirement contract for a pinned
vendor patch: UI-thread idempotent Dispose/Release, immediate removal of bitmap
CPU/remake/source/mips, normal next-redraw retirement of backend-owned textures,
safe stale scene/level references, no upload/redecode after disposed, device loss
and close safety. Cover Update before/after retirement and multi-renderer cases.
No import of MyGo internals from app. Consider whether bounded active-only cache
is simpler; give one recommendation with exact files/API/tests and propagation
through dependency-patches scripts/checks/notices. No finalizer-only guarantee.

Backend contract must be explicit: no silently identical API with different
broken behavior across other supported renderers. Windows is the only accepted
product target, but patch must not create known upstream build breaks. Use pure
tests/fake cache where adequate; real driver lifetime claims need later separate
measurement. No network or UI helper work needed. Keep read-only review bounded
and publish image-bitmap-lifecycle-design.md promptly. Root decides/activates
implementation based on this design; user already authorizes routine choices.
