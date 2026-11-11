# Independent pure Mindmap extraction review

Status: ACTIVE 2026-11-09. Shared pure source, graphview wiring and substantive
tests are present; implementation owner may still fix focused test findings.
Review the current narrow implementation and report the exact files read.
Review only actual changed formats/mindmap presentation/SVG and narrow graphview
wiring against mindmap-engine-pure-next-stage.md. Root and reviewer do not run
product/test/build/launch commands. Read baseline fixtures and original sources
from mindmap-pure-20261109-060918-baseline-attempt01; they are the old exporter.

Look for actionable correctness/ownership/regression issues: geometry/options
drift; lost marker/tag/Unicode/multiline/image/collapsed semantics; XML paint
injection or escaping change; extra source/model mutation; canvas invoking full
SVG or whole-label joining; warning order/count/content; count/fill cap checked
before result allocation and immediate error stop/no partial bytes; error path
and nil behavior; new dependencies. Check graphview cache/idle semantics remain
unchanged and all wrappers delegate actual shared truth.

Meaningful new tests should use stored pre-move SVG bytes plus independent
literal assertions, tiny budget boundaries, exact source serialization/history,
and existing label/color tests. Do not demand broad unrelated tests or a new
physical UI run for a behavior-preserving extraction. No speculative findings.

Write only native/handoffs/mindmap-engine-pure-independent-review.md with actual
source files reviewed, prioritized findings with line references (or no actionable
findings), test-oracle limits and unresolved concrete questions. Do not edit
implementation/evidence, use network, or delegate. Stop after this finite review.
