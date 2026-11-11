# Bitmap retirement implementation activated

ACTIVE2026-11-10 Astra read_native_handoffs_tab_workflow_resume_task_md. Root
read and accepts the bounded recommendation in image-bitmap-lifecycle-design.md
as the implementation direction, with clarifications below. Implement now;
no extra design-only stage or user approval needed. No children.

Own only the exact vendored MyGo scene/ui/raster/GPU paths and dependency patch,
guard/provenance/portable asset files named by that design. No app/document/
session/raster-admission changes, no current package or accepted evidence writes.
Core binary and raster format helpers remain the two GLM owners' domains.
No application feature integration, product package build, actual UI/driver
launch or new resource measurement in this task. One shared source tree: do not
regenerate vendor while GLM Go checks run. Author patch against pristine pinned
source and verify in an isolated temporary tree, leaving active vendor in place.

Implement public UI-thread terminal Bitmap.Dispose() bool, ordinary nil/repeat
safe, borrowed Canvas false without mutation. Retire replaced-size owned base/
mips; post-disposal Update/ReleasePixels/Pixelated/render paths cannot allocate,
restore or revive. Maintain same-size Update behavior. Keep scene.Image IDs and
geometry tombstones safe for old scenes. Exact epoch contract must use all
backend checkpoints before no-damage return; epochs are independent per renderer
with weak entries, no global consumable queue. Clear compressed/remake/source/
CPU ownership immediately. Normal next drawable render or renderer Release
releases backend ownership, with minimized/idle and in-flight limits documented.

Derived canvas/blur/backdrop invalidation is accepted only to prevent stale
copies surviving retirement. Don't flush font/gradient atlas or unrelated live
base textures. Carefully include previous frame/damage and nested canvas caches;
no backward dependency framework. Metal must never submit cached commands after
their handles are released, GL deletion only on owning context, D3D bound COM
refs unbound safely. CPU parallel work must already be joined before UI mutation.

Before running checks, save short implementation checkpoint with exact files
and APIs; continue work without waiting. Tests target actual changed functions,
not mirrored implementations: nil/double, mips and replacement, post-dispose
no decode, stale scenes and effects, Canvas borrowing, nested derived invalidation,
independent renderers, no-damage checkpoint, teardown/device recreation. Where
driver APIs cannot be called, factor the smallest real shared ownership helper
with fake releases rather than making a broad cache abstraction. Explicitly
report what pure tests do not prove. Include actual package build validation for
Windows and feasible Darwin/Linux/js targets, use wrapper/environment properly,
retain unavailable-toolchain diagnostics and never silently claim unsupported
cross-platform runtime validation.

All Go through native/scripts/go.ps1. Its guard, refresh/check-build-guard and
verify_test.go must enforce both startup and disposal patch completeness. Keep
existing startup guard intact. Add new patch to future portable attribution and
notice list/validator without rebuilding or changing .22. Test patch application
once from pristine v0.8.3 in isolated scratch after final changes; report exact
commands/outputs/exits, no repeated broad suites. Focused final vendor suites,
guard/provenance and package-asset checks are authorized; product build/host is
later. Avoid repeated formatting/compile snapshots while known defects remain.

Publish image-bitmap-lifecycle-implementation-result.md and frozen current file
identities. Root will review before image view integration or real driver checks.
