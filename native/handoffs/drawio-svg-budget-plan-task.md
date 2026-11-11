# Next slice: bounded draw.io SVG output planning

READ-ONLY ACTIVE2026-11-10 while .17 is built. Owner DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. No children. Root
plans/reviews; no source/test/docs edits or commands/build/captures/launches.
Own only drawio-svg-budget-plan.md; roughly two pages, code-backed specifics.

User goals are low memory and format plugins. Current three fixed workers cover
Markdown HTML, mindmap SVG and readonly mindmap scene. Draw.io export still runs
in host graphview.Export. It constructs an unbounded bytes.Buffer; parser
budgets don't directly bound expanded SVG. First establish a bounded output
contract, then consider worker extraction without changing selected-page semantics.

Review actual formats/drawio/svg.go/types.go/parse.go/tests and graphview.Export,
mindmap bounded exporter as reference. Propose smallest compatible first slice:
default hard64MiB SVG output cap, optional lower limit for caller/tests, errors
return nil payload rather than partial document, source/unknown metadata intact,
same successful output bytes/warnings/order. Avoid allocating escaped whole
user strings or whole final output just to count. Fixed ordinary options and
all user strings (labels/styles/font options/path) need same counted/fill rules.
Consider output amplification by many newline-separated spans and exact-cap
tests; compare memory allocation locally only after implementation, no RSS claim.

Identify actual helper reuse: escapeAttr also participates XML edits, so don't
silently change its behavior outside SVG. Keep parser/geometry behavior, warnings,
page selection and graph UI unchanged. Decide exact error/option contract and
meaningful tests against fixed golden bytes or pre-change oracle; avoid a copied
full duplicate exporter. Test exact limit, one byte short, default cap, Unicode/
xml escaping/newlines/multi-page/source immutability and no partial result.
Final affected suites only formats/drawio + modules/graphview unless actual
imports require another. Benchmark only if needed to support allocation claim.

Add one short forward dependency note: current engineipc.RequestMeta has only
engine/format, while drawio exports selected page. A later worker must carry
validated page identity and host version/lifecycle checks; never default to page0
or rewrite source to pass only a page without design review. This stage is
output bounding only, not a fourth engine/framework/protocol change.

Publish concrete plan with code locations, scope, expected successful-byte
contract and verification. Product source remains frozen until .17 accepted.
