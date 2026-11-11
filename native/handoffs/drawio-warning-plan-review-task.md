# Read-only structural-warning design review

ACTIVE2026-11-10. GLM owner as listed in native/AGENTS.md. No children;
compact before200K. No product/docs edits, tests, builds, launch or measurement.
Only publish `drawio-warning-plan-review.md`. .20 packaging inputs and .19
artifacts are frozen. Work from current source; do not re-review the accepted
custom-suffix route.

Review `drawio-structural-warning-plan.md` against actual drawio parser, template,
SVG and selected-page worker tests. Root accepts a narrowly scoped first step:
recognize just the canonical 0/1 structural pair used by new-document template;
do not introduce broad layer/group support or suppress arbitrary unsupported
cells. Assess the ordered-first-pair policy as a disclosed conservative subset,
and note follow-up generalization separately rather than expanding this stage.

Identify concrete pitfalls and minimal ownership/test scope. In particular:
both cells must be fully validated before either generic warning is removed;
if the second fails recognition, retain first warning and all relative warning
order. Normal parser budgets/accounting/duplicate checks must run unchanged.
Unknown children/attrs/namespaces, processing instructions, non-XML whitespace,
geometry, metadata and malformed pairs must not accidentally qualify as empty.
No second decoder or whole-source copy should be needed. Unknown sibling
content must retain its own warnings. Attribute order/quotes/comments/XML
whitespace/self-closing vs explicit close should not arbitrarily change the
meaning of an otherwise exact pair.

Read bounded SVG oracle and worker tests: preserve immutable old oracle bytes;
when expected issues change, separate intentional two-warning subtraction from
exact SVG/source bytes. Identify affected fixtures and affected final suites,
including actual selected-page warning-budget behavior. Do not edit any oracle
or change .19/.20 historical evidence. The result should allow implementation
to start after .20 is frozen with no further broad planning round.
