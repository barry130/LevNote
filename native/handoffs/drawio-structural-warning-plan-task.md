# Plan next drawio warning correction

ACTIVE read-only planning 2026-11-10 for DeepSeek
`/root/read_native_handoffs_resource_workload_next_task_md` after route source
freeze. Own only `drawio-structural-warning-plan.md`. No implementation/test/
build/launch/docs edits, no children, compact before200K. Route source remains
frozen while another agent reviews and `.20` delivery will follow.

Root observed `formats/drawio/parse.go` currently emits UnsupportedElement for
every mxCell lacking vertex/edge, including ordinary root/layer cells0and1 in
the sole new-document template. Tests consciously permit those warnings; the
actual `.19` second-page smoke returns two such warnings. This creates noise
for clean new documents. Analyze one small safe correction for a later stage.

Read only parser/types/template, relevant drawio and worker tests, and the
neutral warning-display path if needed. Do not browse/download or inspect old
full release logs. Produce at most2pages:

- Define exactly which structural root/layer cells are understood and may be
  omitted from unsupported-render warnings. Do not silence arbitrary nonvertex
  cells with labels/styles/geometry/unknown children. Decide conservative exact
  canonical cases vs generic root/layer relationships, and explain order/IDs.
- Preserve source bytes, duplicate validation, existing parser budgets, original
  edits and real unsupported warnings. No new permissive parser or render model.
- Identify minimal files and substantive regressions for clean new template,
  standard roots/layers, misleading extra attributes/children, existing warnings,
  selected-page filtering and warning limits. Keep fixture expectations explicit.
- Identify effects on accepted real-smoke literal warnings. `.19`/`.20` records
  remain historical; future smoke oracle must change only under later source
  acceptance, never rewrite those artifacts or call old warnings a failed run.

No benchmark/resource claim or new UI framework. This task is a plan only;
implementation needs explicit ownership after `.20` freeze.
