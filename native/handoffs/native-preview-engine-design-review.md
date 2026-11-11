# Preview checkpoint: root decisions and required clarifications

2026-11-09. Read-only pilot direction accepted for planning only; implementation
remains inactive until .6 export delivery/docs acceptance. This file refines
native-preview-engine-design.md. GLM owns a single concise revision of that
design, no product edits/execution. Resolve these concrete points, not a broader
architecture study:

1. Exact allocation claim: the canvas path removes its parser allocations, but
   Adapter.Outline still calls mindmap.Parse when the outline panel is open.
   State this explicitly and preserve existing outline visible-node navigation
   by validated ID in the new view; folded readonly targets remain a visible
   refusal. Do not quietly remove navigation/metadata access that readonly
   users already have. If inspector remains source-only, disclose that limit
   in the pilot and keep a reachable full-source entry.
2. Existing DisplayLabel may return 512 UTF8-prefix bytes PLUS a three-byte
   ellipsis. Wire limit must admit515, with existing truthful truncation warning;
   do not truncate twice. IDs over512 must be refused visibly, never clipped.
3. Native fallback paint follows current light/dark theme, while the draft DTO
   requires concrete colors for every node. Use optional literal override
   colors (empty means renderer's existing theme role) plus root/selection
   state. No CSS expressions/theme names from workers. Normalize accepted hex
   to consistent RGBA; unsupported values retain existing fallback diagnostics.
   Include whether ImagePlaceholder is needed for the intended canvas parity.
4. Bounds must be checked before JSON materialization: 32MiB wire bound alone
   does not prevent millions of small elements or arbitrary nesting/key strings.
   Define a bounded shallow decoder with per-field/count/depth admission before
   retaining elements; do not unmarshal a full untrusted tree then count it.
   Encoder count-then-fill/hard cap; no partial payload. Nodes/edges uniqueness,
   one root, finite positive dimensions and endpoint membership are explicit.
5. Scheduling must share the ONE Runner with export. Specify how export gets
   priority, preview cancels/joins, busy does not become a retry/CPU loop, and a
   single pending newest snapshot is scheduled after admission is available.
   Release only signals/cancels and drops ownership on UI; it must not block the
   UI thread for up to30s waiting. App close owns final joins and cannot deliver
   callbacks into a released view. Theme changes need no new worker request.
6. Current Host func signature drops warnings/engine identity. Return a bounded
   response with scene/warnings/engine identity via an app-owned generic scene
   service or optional capability, not a Mindmap-named field per new format.
   Keep immutable source identity and adapter/view/request epoch checks.
7. Separate fixed binary is acceptable for this finite pilot; justify it by
   current operation-bound manifest/protocol simplicity, not an assertion that
   adding an operation necessarily changes old response bytes. Note disk cost
   and that installed multi-capability package identity remains later work.
   Reuse the process runner; no proliferation of independent admission slots.

Keep the revised checkpoint compact and report complete. Unresolved signature
details may be named as bounded source checkpoints for the implementation owner.
