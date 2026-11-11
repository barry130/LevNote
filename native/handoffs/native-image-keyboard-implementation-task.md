# Native image keyboard increment

ACTIVE2026-11-10 after root accepted .23 and its closed evidence.
DeepSeek imageview/delivery owner implements. No child delegation.

Implement the concrete, already reviewed native-image-keyboard-next-plan.md.
Own only modules/imageview/input.go, modules/imageview/ui.go,
modules/imageview/imageview_test.go and new app/binary_keyboard_test.go.
Do not change app runtime, contracts, geometry, raster, document, session,
vendor, package scripts or frozen release/evidence directories.

Use existing focused canvas input: exact modifier handling, one action per key,
plus/equal and keypad zoom, minus zoom,0 fit,1 actual size,32DIP arrow pan.
Keep typing/IME and app shortcuts isolated. Return canvas focus only after an
explicit enabled zoom-toolbar action, not decode/tab/plugin events or bytes.
No timers, caches, decode work or persistent setting. Retain existing geometry
and memory limits. User-facing guidance fits tooltips; no added toolbar rows.

Deliver three substantive public Tester scenarios from the plan: focus/input
containment, event-driven geometry including bounds and tiny-fit behavior, and
real app byte/tab/release boundaries. Add a test only where it verifies a real
behavior rather than mirroring helpers. Record failures honestly and fix causes.
After changes settle, coordinate with root and Astra autosave owner: hold the
combined app/imageview suite and vet until both source sets freeze; one owner
will run that combined verification. Focused new imageview tests during debugging
are allowed through native/scripts/go.ps1. No captures, package, host, worker or resource runs
for this small source increment. Avoid repeating successful checks unless an
actual correction affects their result.

Write progress and final result in native/handoffs/native-image-keyboard-*.md,
with changed files, exact commands/exit evidence and final source identities.
Keep output logs under a fresh native/handoffs/native-image-keyboard-evidence-20261110
directory. Root will review the behavior/source and accept, then decide the next
product increment; this task does not create a new release harness.
