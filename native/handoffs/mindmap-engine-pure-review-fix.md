# Preserve the SVG output-limit sentinel

Status: ACTIVE REPRODUCTION 2026-11-09. Owner: astra_marker_view. Root read the
independent review and current svg.go/svg_test.go. Astra then identified contrary
evidence: pinned encoding/xml/xml.go:1957-1966 returns the writer error unchanged,
and fmt/print.go:214-219 also returns Write's error. Root verified that source.
The review hypothesis is NOT established; reproduce it before product changes.

Scope: formats/mindmap/svg.go and svg_test.go only, new evidence and this result
handoff (or a separate -result.md). Preserve all baseline fixtures and prior
evidence. No app/runtime/catalog/IPC/graphview/build/launch/resource changes.

The review claims xml.EscapeText's returned error replaces the writer's
ErrSVGOutputLimit with io.ErrShortWrite. The pinned source contradicts that
mechanism. First run the new meaningful boundary oracle on unchanged svg.go.
If it passes, retain the added regression and report the finding as unconfirmed
or invalid with exact evidence; do NOT make a defensive product change without
a demonstrated failure. No red result may be invented. If it fails, preserve
the actual failure and implement only the minimal verified correction.

Add a substantive small-fixture regression proving refusal within escaped text
(each XML special entity and a plain prefix/suffix), not only before fixed SVG
fragments. Sweeping all positive smaller-than-complete caps on a small fixture
is acceptable and proves every boundary. Assert nil output and errors.Is with
ErrSVGOutputLimit. Include direct count/fill writer bounds/sticky behavior if
needed; do not claim allocation proof from an output nil check. Exact full
output and retained byte fixtures must continue to pass.

Prefer record the focused failure before the fix, then focused success and full
formats/mindmap package plus scoped vet once. Preserve real command/streams/
exit/time/source fingerprints. Existing graphview/full fixture acceptance is
reused; no graphview rerun unless a relevant new defect requires changes.
Write a brief final report with exact evidence paths, files changed and limits.
Stop this narrow assignment for root review; runtime/app activation follows.
