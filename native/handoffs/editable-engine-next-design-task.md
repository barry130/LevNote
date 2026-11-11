# Current assignment redirect — fixed-engine package core implementation

CURRENT2026-11-09: Broader store assignment is superseded. Read
`native/handoffs/fixed-engine-package-inventory-activation.md` and implement
ONLY its pure inventory decoder/test files. Astra owns remaining store. This
new narrower task takes precedence over the earlier redirect below.

2026-11-09 after .8 exact delivery acceptance. The design-only task below is
completed. Its assigned GLM agent now owns implementation under
`native/handoffs/fixed-engine-package-core-activation.md` (ACTIVE). Read that
file once and execute its finite source/tests slice. Do not repeat this design
or wait for a readable message if payload transport is opaque. Publish actual
API to `native/handoffs/fixed-engine-package-core-checkpoint.md`, then continue
its scoped filesystem tests/vet. Astra concurrently owns source docs only.

## Completed historical assignment

2026-11-09. Assigned to GLM task
read_native_handoffs_editable_engine_next_design_task_md. If task payload is
opaque, this file is authoritative. Read once and proceed; do not repeatedly
read this file or wait for another message. No child delegation. Context stays
below 200K; this assignment should use far less.

Root plans/reviews; Astra alone implements current schema3 app integration.
You may read source and write only
native/handoffs/editable-engine-next-design.md. No product/test/script edits,
test execution, build, process launch, package mutation, or network research.
The current app acceptance and .8 packaging must not depend on this design.

User wants supported formats as plugins with project-authored engines and MyGo
native UI, low memory/CPU, no WebView. Current .7 ships three fixed one-shot
workers (Markdown HTML, Mindmap SVG, readonly Mindmap scene) and keeps editable
format parsing and semantic outline in the host. Schema3 now binds local suffix
configurations to shipped Markdown/Mindmap adapters; it does not install code.

Read actual seams, using rg --files before guessing names:
- ui/editor contract, app lifecycle/preview/export/engine_lane
- core/enginecatalog, engineipc, engineruntime, enginescene
- modules/graphview draft/scene/commit, formats/mindmap edit/fields/serialize
- existing handoffs/native-preview-engine-design.md for prior decisions

Recommend the smallest NEXT usable editable-engine slice, preferably a single
Mindmap topic change, without pretending that adding another JSON alias or
export worker completes independently installable engines. Be concrete about:

1. Which exact operation first crosses a worker boundary and what user action
   triggers it; which parse/layout/outline work still remains in host.
2. Authoritative source snapshot, exact target identity and precondition,
   bounded request/response, unknown metadata, no-op byte fidelity, host-owned
   one-step undo and stale/cancel/error behavior. Existing protocol request is
   source-only with 4KiB metadata: do not invent unbounded command metadata or
   claim the existing readonly DTO contains the full editable topic.
3. Native UI draft lifecycle while worker runs. Existing Flush is synchronous
   and fallible; saving/navigation cannot drop pending edits or block the UI
   awaiting a process. State how an explicit apply/wait/refusal would work, or
   why a narrow additive asynchronous contract is necessary. Keep the existing
   single execution lane and no permanent per-tab processes.
4. Package identity/installation gap: which shipped fixed capability can be
   implemented first, and what still must exist before user-installed native
   executables can be truthfully supported. SHA256 isn't publisher trust or an
   OS sandbox. Do not add a third-party runtime or network feature.
5. A finite staged implementation with exact owners/files/meaningful oracles
   and one fresh deliverable gate. Reuse old protocol/resource evidence when
   unchanged; avoid another large generic validation framework.

Keep the result roughly 150-220 lines or less, distinguish recommendations from
implemented facts, include actual source references, and report the handoff path
when done. No permission request. If an interface cannot be used as expected,
explain the concrete mismatch and propose one choice rather than several vague
alternatives. Root makes final sequencing decisions.
