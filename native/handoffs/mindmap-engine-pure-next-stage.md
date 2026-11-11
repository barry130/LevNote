# Second format engine: pure Mindmap presentation

Status: ACTIVE 2026-11-09. Root reviewed and accepted alignment .5 final source
docs and separate acceptance87AFB26C...; Astra read this plan with no API/scope
objection. Root plans and reviews; Astra owns implementation and execution.
The accepted .5 package and all earlier artifacts are immutable.

## Product direction

Build a second separately packaged repository-authored export capability,
Mindmap source to SVG, and a fixed engine admission catalog. This stage is the
pure extraction prerequisite, not executable plugin installation or modular
preview/editing. The corrected design is
`native-format-engine-boundary-design.md`; this plan resolves its API sketches.

The current `modules/graphview/mindmap_svg.go` is NOT a pure format public API.
It calls label/color/object helpers in graphview. Extract the real shared
semantics into `formats/mindmap`, preserving native behavior and exact output.
Do not make canvas rendering generate SVG: cheap label/color/warning helpers
and the explicit SVG export entry must remain separate. No aggregate Present
function that allocates the whole SVG just to draw or report warnings.

## Owned scope

- New pure presentation/SVG source and meaningful tests in formats/mindmap.
- Narrow graphview wiring: mindmap_svg.go, render_cache.go, color.go, sync.go,
  and adapter.go only if needed for the bounded export call.
- Existing graphview export/label tests may move only with their actual pure
  helpers, or use thin wrappers. Preserve useful independent assertions.
- A small graphview integration regression if needed.
- This handoff and a new implementation report/evidence directory.

No parser, serializer, editing semantics, text core, MyGo/vendor, app, IPC,
runtime, packaging scripts, executable builds, actual process/UI launches,
resources or network changes in this first sub-gate. Report a concrete reason
before widening ownership. No child delegation.

## Contract decisions

Names may be chosen for clarity; keep the API small. Expose immutable-by-value
native presentation layout options (a function returning LayoutOptions, not a
mutable package variable): NodeHeight40, HorizontalGap72, VerticalGap24,
Padding28, MinWidth92, MaxWidth380, CharWidth8, LineHeight20. Canvas and future
worker will use exactly this one function. Do not alter general Layout's
existing default behavior.

Share the current 512-byte UTF8-safe topic/icons/tags display prefix, truncation
diagnosis, literal color admission (64-byte bound, hex/named/transparent/none),
unsupported-color counts and image-placeholder/long-label warnings. Preserve
unknown fields, duplicate icons/tags and existing fallback/color/geometry/SVG
semantics. Keep UI color conversion and text-layout cache in graphview. Helpers
must not construct an unbounded joined label merely to truncate it afterward.

Export takes Document plus explicit LayoutResult and a bounded maximum output
length; choose default/hard maximum 64MiB. A count-then-exact-fill writer is
preferred: counting must stop as soon as the budget is exceeded; no unbounded
bytes.Buffer then length check. Each write checks remaining budget before
growth, and refusal returns nil bytes with a sentinel error, no partial SVG.
Escaping and numeric output remain byte-identical for accepted inputs. Do not
claim the output budget bounds total process memory. The low-budget seam must
allow small deterministic refusal tests without 64MiB fixtures.

Warnings remain cheap structured data separate from SVG. Existing parse/layout
issues have no byte offsets; future wire mapping will explicitly use the whole
document [0,sourceBytes], not invented node offsets. Do not redesign wire here.
Keep current visible warning meanings and order/deduplication where applicable.

## Finite acceptance and handoff

Before moving code, capture a few representative current SVG outputs as retained
baseline fixtures (normal left/right geometry, Unicode/multiline/escaping,
markers/tags, literal/unsafe paints, negative bounds, collapsed/image node).
The oracle must not simply invoke the new production helper twice. Include
independent literal/XML assertions for geometry/escaping/labels/warnings; retain
exact hashes/bytes for pre/post equivalence. Existing tests already cover much
of this: reuse rather than duplicating all assertions.

Test output just below/equal/over a small exact boundary, UTF8 validity,
nonmutating document serialization/source/history, blank/multiline text, long
metadata's bounded prefix and precise warnings. Exercise graphview using the
same shared functions without rebuilding labels or SVG on idle frames.

Publish a concise API/source/oracle checkpoint once the extraction compiles;
continue focused meaningful tests in your scope. Then run full
`./formats/mindmap ./modules/graphview` and scoped vet once, with actual streams,
exit/time and before/after source inventory. Retain failed attempts and causes.
No app/whole-repository/process matrix or new screenshots needed for a behavior
preserving extraction. Root reviews actual sources and logs; next sub-gate will
add catalog/IPC/runtime/worker/app after this one is accepted.

Write `native/handoffs/mindmap-engine-pure.md` with API, moved helpers, exact
baseline comparison, tests, limits and unresolved issues. Stop at the gate;
root will immediately route review and the following implementation phase.
