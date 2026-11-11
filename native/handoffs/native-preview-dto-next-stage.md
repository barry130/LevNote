# Native preview DTO: root implementation plan

2026-11-09. Root has read the corrected authored contract in
native-preview-dto-contract.md. The old content at this path was an accidental
contract overwrite; this restores the root-owned stage plan. The corrected
contract supersedes the earlier design's number, paint and field-order details.

## Activation and ownership

ACTIVE: Root accepted .6 package, seven source documentation sections, final
handoff and separate acceptance record5978D7FD... . Astra alone now owns the new
native/core/enginescene package, meaningful tests, focused test/vet execution,
new scoped evidence and native/handoffs/native-preview-dto.md. No overlapping
source owner or child delegation. Root reads/reviews only.

Publish a short source checkpoint explaining parser/encoder structure and error
semantics before implementation, then proceed under this finite plan. Root will
read promptly. No catalog, IPC, runtime, worker, app, UI, existing formats,
module/dependency, package script or source launch-document changes yet.
Accepted .6 and prior evidence/bundles remain immutable. No builds, launches,
UI/resource measurements or unrelated repository test replay for this package.

## Exact contract and clarifications

Use corrected contract Scene/Node/Edge/Point/Rect API, limits, required exact-key
fields and rooted-tree invariants. Remain independent of concrete parser, UI,
runtime, file and process packages.

- Layout v1 allowlist: logicalStructureLeft, logicalStructureRight,
  logicalStructure, orgChart. These are wire values; no Mindmap package import.
  Later producer integration verifies effective layout mapping.
- Labels admit515 UTF-8 bytes including existing512-prefix+ellipsis; no second
  truncation. IDs nonempty opaque valid UTF-8 up to512 bytes.
- Every paint field required, either empty or lowercase #rrggbbaa. Empty means
  host theme fallback. No normalization or repair at the trust boundary.
- Counts required, equal actual lengths regardless of key order. Exactly one
  root, n-1edges, no missing references/self-edges/duplicate-parent, cycle or
  disconnected component. Use bounded iterative graph validation.
- JSON number syntax standard, finite, within declared bounds; schema/counts
  integral. No invented canonical spelling requirement. All widths/heights
  strictly positive, including bounds.
- Standard outer JSON whitespace allowed; trailing non-whitespace refused.
  Invalid UTF-8, lone UTF-16 surrogates, unknown/duplicate/case-altered fields,
  nulls, missing fields and unexpected nested shapes refused, not repaired.
- Local limits enforced before copying strings, allocating by declared counts
  or retaining excess elements, even with arrays before counts. No generic
  interface/map tree or unbounded allocation by ordinary JSON string tokens.
  Global tree invariants run after bounded parsing, before result publication.
- Encode validates, counts against32MiB, fills one exact output buffer with
  deterministic key order. Decode accepts arbitrary key order. No partial
  bytes/scene on failure or aliases to caller input bytes. O(nodes+edges)
  validation, no depth-proportional recursion.
- Small stable error classification for later invalid-scene versus resource-
  limit handling. Do not create a generic serialization framework.

## Meaningful acceptance

Independent literal wire and expected values, not just encode/decode roundtrip.
Unicode/escapes, all layouts, normalized and theme-empty paints, arbitrary field
order/late counts, exponent numbers, deterministic output, exact numeric/string
boundaries, malformed trees/UTF-8/escapes/duplicate/missing/null/trailing cases.
Exercise incremental string/element admission and payload limits with an oracle
showing admission, not merely a final error. Mutating caller bytes after Decode
must not mutate scene. Encode errors return nil; Decode errors return zero Scene.

Run focused regression as needed, then full new package -count=1 and scoped vet.
Retain actual command arguments/raw streams/exits/times and a small source/
module/wrapper before-after inventory in a fresh acceptance directory; preserve
failures. No need to hash every historical package/toolchain or rerun accepted
suites for this standalone pure package. Report actual results and limitations.

## Following stages, not activated yet

After root reads codec source and genuine evidence: fixed catalog/IPC entry and
bounded preview worker; then app single-runner scheduling/read-only native view;
then fresh complete bundle/real consumer acceptance and truthful documentation.
Export priority; nonblocking Release; at most one newest pending snapshot and
one explicitly stale scene. Source/tab/document/buffer/adapter/view identity is
app-local, never invented as worker echo. Editor optional response holds neutral
scene/warnings/engine identity, without runtime import. Readonly pilot preserves
editable graphview and outline parser path; visible-node navigation and reachable
source-only inspector are explicit. No full host-parser removal, arbitrary
editable plugins, sandbox or measured resource-benefit claim.
