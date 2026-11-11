# Draw.io structural-warning independent review

Date: 2026-11-10. Read-only review of the frozen warning increment against
[drawio-structural-warning-implementation-task.md](drawio-structural-warning-implementation-task.md),
[drawio-structural-warning-root-draft-notes.md](drawio-structural-warning-root-draft-notes.md),
the six changed source/test files, their raw evidence logs, and
[drawio-structural-warning-result.md](drawio-structural-warning-result.md).
No product source, test, package, historical evidence, or documentation was
edited; no test, build, launch, capture, or measurement was run.

## Conclusion

No actionable defect or evidence gap was found. The implementation satisfies the
assigned narrow warning-only contract. This review closes cleanly; root may
accept the warning increment and start the separately owned retained-source
optimization.

The corrected frozen result at review completion is
[drawio-structural-warning-result.md](drawio-structural-warning-result.md),
9,778 bytes, SHA-256
`0B9DC4B7676DE115BC9CA7A0F69F1253B25365CB3644E7329EFA433CE49C9C04`.

## Production semantics

`parseRootCells` records only a first direct `mxCell` whose start tag is exactly
`id="0"` and whose cell parse is empty without processing instructions. It
suppresses warnings only when the immediately next direct cell is exactly
`id="1" parent="0"`, also fully parsed, and both corresponding generic
non-vertex issues remain in the same issue list. Missing, extra, unknown,
namespaced, or text-bearing lookalikes retain both ordered warnings. The
malformed-second test preserves the first warning and returns the XML error.

The root override is honored: XML comments and XML-S whitespace
(`space`, tab, CR, LF) inside or between the pair do not disqualify it.
Processing instructions and non-XML whitespace, including the covered NBSP and
U+2003 cases inside and between cells, do disqualify. Attribute order, quote
spelling, and self-closing versus explicit close do not affect recognition.

Duplicate-ID insertion still occurs before any suppression, so duplicate `0`
returns `ErrDuplicateID` after the first generic warning and never reaches the
pair-removal path. Element/text budgets, source offsets, read-only status, edit
semantics, model output, serialization, and SVG bytes remain untouched. The
exact element-budget test also shows that pair recognition does not bypass the
normal element count: three counted elements pass at a limit of three, and the
attempt at two fails before the pair can be accepted.

`removeCanonicalContainerWarnings` removes only the two exact issues with IDs
`0` and `1`, in document order, and then clears both vacated backing-array
slots. The explicit backing-tail test proves removed issue values are no longer
reachable behind the logical slice.

The conservative ordered-first-pair scope is fairly stated. It is a template
special case, not general layer/group support: later, reordered, repeated, or
custom structural cells continue to warn pending a separate design.

## Test and oracle assessment

- `canonical_container_test.go` covers the canonical pair, serialization
  identity, preceding and following unknown siblings, intervening siblings,
  malformed second-cell XML, wrong identities, extra attributes, reversed
  order, comments/whitespace, processing instructions, namespaces, non-XML
  whitespace, children, duplicate IDs, budgets, source offsets, edits, and
  backing-tail cleanup.
- Lookalike cases retain correct `id`/`parent` distinctions rather than being
  rejected by an unrelated missing field: wrong second parent, first parent,
  extra attributes, and reversed pair all use otherwise valid cells.
- `svg_budget_test.go` leaves the immutable pre-change oracle bytes untouched
  and subtracts only the two known canonical-container warnings from the parser
  expectation. It separately asserts exact old SVG bytes, unchanged source
  fields, model/page identity, and emitter issues.
- Worker tests preserve exact SVG/source behavior, document and page warning
  ordering, the compressed-readonly warning, selected-page-before-budget
  filtering, and exact 256/257 warning bounds. The combined-bound fixture uses
  252 substantive sibling warnings plus four HTML/stencil issues, preserving the
  limit rather than weakening it.
- Graphview now asserts the shipped draw.io template parses with zero issues.
- The gated managed-smoke oracle requires an empty warning list for the
  unchanged clean second-page fixture while retaining the exact 598-byte SVG,
  source identity, process, selection, and lifecycle checks. Its real process
  path remained gated and was not run.

Historical warning records were not relabeled as wrong; they remain evidence of
the earlier parser behavior.

## Identity and evidence check

All six changed source/test hashes and the immutable oracle hash match the
result handoff exactly:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/parse.go` | 36,101 | `92BDCA474CE49921FD88331079A186FB6CD6887F69DA8000B53C2217E1ACF8C9` |
| `native/formats/drawio/canonical_container_test.go` | 12,328 | `AC7D0C953D9DE15AA9BB310966A6E69233D51FD0103A2DEF4E7D8CFD860AE01D` |
| `native/formats/drawio/svg_budget_test.go` | 11,902 | `9EA52B0072F027E2CB4ACEBC607797723FD530F7451733F52B0969C655556EE8` |
| `native/cmd/levnote-drawio-svg/main_test.go` | 16,147 | `9D19F951FB39915A21DE334D932501E0133585115975AD4C0ECEE0C263234E74` |
| `native/modules/graphview/graphview_test.go` | 9,960 | `040068914B122C2A0A36B56EA6C045FE1BB7006D16620FE8CC89498CDE35E632` |
| `native/core/engineruntime/drawio_managed_bundle_test.go` | 20,800 | `CB678B7DA1D2C4F5B91E5873D156B97CC4FCBD248F0785FD7045674DBF1DD695` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | 6,688 | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |

All nine evidence log sizes and hashes also match. The raw logs show the
preserved focused failures described by the handoff, followed by the passing
focused draw.io run and the final four-package test log. The final vet log is
empty.

## Evidence provenance limitation

The evidence directory contains stdout/stderr logs but no exit-code sidecars.
The result correctly discloses that exit statuses, including the reconstructed
success statuses for empty logs, come from the implementation owner's recorded
tool history. This review matched the log identities and contents to the handoff
and found no contradictory evidence, but the logs alone cannot independently
prove exit status. That provenance limitation is disclosed rather than treated
as newly generated execution evidence.

The author's authorized prose-only cleanup fixed the known garbled fragments and
the row-9 log-name typo (`final-affected-vet-1.log-progk` to
`final-affected-vet-1.log`) without changing source/test/evidence identities.

## Close

This is a warning-only source increment. It does not claim UI/OS, resource,
physical input, package, or broader structural-generalization validation. The
retained-source optimization remains separate and inactive until root assigns it
after warning acceptance.
