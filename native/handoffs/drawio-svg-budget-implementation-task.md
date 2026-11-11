# Draw.io SVG output budget implementation

ACTIVE2026-11-10 after .17 root binary acceptance; implementation owner DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. No children,
compact before200K. Root plans/reviews only. Read reviewed plan
drawio-svg-budget-plan.md. Own formats/drawio SVG/types/narrow tests plus one
modules/graphview caller test and fresh handoff/evidence. No product docs,
protocol/catalog/worker/package script/UI changes. No build/launch/capture.

Plan accepted with these implementation details:

- Add MaxSVGBytes64MiB and SVGOptions.MaxBytes, zero/default or above-hard clamp,
  positive lower cap, negative sentinel refusal. Keep page lookup precedence.
  Any failed export returns nil SVG and nil issues. Successful output/warnings
  exactly unchanged. Original/source/model/page/unknown metadata not mutated.
- One private emitter count then exact-size fill; cap counts every emitted byte
  before writing/advancing. Stop at first writer error and never allocate the
  complete output on a count-pass refusal. Fill must match count. Shared edit
  escapeAttr must stay unchanged. No general serialization framework/dependency.
- Direct xml.EscapeText(writer, []byte(wholeHugeString)) STILL allocates a whole
  input byte slice. Escape SVG strings through bounded UTF8-safe chunks or an
  equally precise streaming helper; preserve encoding/xml's exact invalid UTF8,
  quote/apostrophe/control/newline semantics. Boundary tests must straddle a
  chunk at multi-byte rune and escapable characters. No custom unsafe conversion.
- Don't pass arbitrarily long user strings to fmt.Fprintf, concatenate large
  escaped attributes, split all newlines into a slice, or build full edge path
  just to count. Numeric/fixed fragments may use bounded formatting. Keep geometry
  and edge spelling exactly as before; per-page model indexes aren't output cap.
- Count issues once; fill shouldn't duplicate warnings. Source/parser/model and
  issue allocations are not capped by SVG byte budget; disclose this limitation.

Before emitter edit, preserve a compact pre-change exact oracle (source/options,
SVG and ordered issue values) as testdata or literal fixture using old code.
Do not copy whole old exporter. Test successful byte equivalence for Unicode,
XML escapes/options/trailing-newline labels/multipoint edges; exact cap and
cap-minus-one with nil output/issues; default/negative/hard normalization;
newline output amplification; invalid/multi-page selection; source unchanged on
success/failure. Strengthen existing XML parse verification to reach EOF with
no error if relying on well-formedness (current test stops on any error).
One graphview selected-page/export regression. Preserve focused failures and
only rerun affected tests after corrections. One final drawio+graphview test/vet
via wrapper. No benchmark required unless a new allocation claim needs evidence.

Publish drawio-svg-budget-result.md with actual changed hashes, commands/raw
outputs/exits, exact-oracle origin, meaningful assertions and limitations; freeze
for independent review. All prior packages/evidence immutable. This first slice
prepares worker extraction; selected-page protocol remains future design work.
