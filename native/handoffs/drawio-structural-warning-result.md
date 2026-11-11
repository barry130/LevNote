# Draw.io structural warning implementation result

Status: **FROZEN FOR INDEPENDENT REVIEW**  
Date: 2026-11-10  
Owner: DeepSeek resource/implementation agent

This handoff covers only the canonical draw.io structural-container warning
increment. It does not accept itself, build or launch a release, edit immutable
`.19`/`.20` artifacts, measure resources, or begin retained-source optimization.

## Outcome

The draw.io parser now treats the exact ordered first direct `mxCell` pair

```xml
<mxCell id="0"/><mxCell id="1" parent="0"/>
```

as canonical structural containers only after both cells have parsed
completely. It removes exactly their two generic non-vertex warnings. Comments
and XML whitespace (`SP`, tab, CR, LF) may separate the cells or fill their
contents. Attribute order, attribute quoting, and explicit-close versus
self-closing spelling do not matter.

The following remain disqualifying and retain the ordered generic warnings:
processing instructions, non-XML whitespace or other text, namespaces on the
element or relevant attribute, extra attributes, children/geometry, a wrong or
missing identity, and an intervening non-comment root sibling. A preceding
unrelated sibling does not invalidate the first-`mxCell` rule, but its own
warning remains visible. Duplicate-ID checks happen before suppression. If the
second cell fails XML validation, neither warning is removed.

Recognition uses only XML space bytes. Existing generic issue emission still
uses the parser's prior `bytes.TrimSpace` condition, so Unicode NBSP/U+2003
content disqualifies the pair without adding a new text issue that the parser
did not previously emit. Canonical-warning removal clears the vacated slice
backing tail so removed messages and cell IDs are not retained.

No node/edge, source-byte, edit-offset, SVG-byte, budget, readonly, or error
semantics changed. This is not general layer/group support: later, reordered,
custom, or non-empty structural cells still warn as before. Source retention
remains out of scope.

## Final source files

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/parse.go` | 36,101 | `92BDCA474CE49921FD88331079A186FB6CD6887F69DA8000B53C2217E1ACF8C9` |
| `native/formats/drawio/canonical_container_test.go` | 12,328 | `AC7D0C953D9DE15AA9BB310966A6E69233D51FD0103A2DEF4E7D8CFD860AE01D` |
| `native/formats/drawio/svg_budget_test.go` | 11,902 | `9EA52B0072F027E2CB4ACEBC607797723FD530F7451733F52B0969C655556EE8` |
| `native/cmd/levnote-drawio-svg/main_test.go` | 16,147 | `9D19F951FB39915A21DE334D932501E0133585115975AD4C0ECEE0C263234E74` |
| `native/modules/graphview/graphview_test.go` | 9,960 | `040068914B122C2A0A36B56EA6C045FE1BB7006D16620FE8CC89498CDE35E632` |
| `native/core/engineruntime/drawio_managed_bundle_test.go` | 20,800 | `CB678B7DA1D2C4F5B91E5873D156B97CC4FCBD248F0785FD7045674DBF1DD695` |

The immutable SVG oracle was not edited:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | 6,688 | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |

That hash is the previously recorded pre-change/accepted oracle identity. Its
fixture retains exact SVG/source bytes and the four cell-level parser/emitter
issues; only the two canonical-container parser warnings are intentionally
absent from the new expectation. No broad warning-filter helper was added.

## Intentional warning changes

| Fixture/check | Before | After | Preserved behavior |
|---|---:|---:|---|
| Exact second-page worker export | 2 | 0 | Exact SVG bytes and source bytes |
| Worker warning order/scope | 9 | 7 | Document warnings, later `<future/>`, HTML, and stencil order |
| Compressed draw.io page | 3 | 1 | Compressed-readonly warning only |
| Large unselected population | 2 selected-page | 0 selected-page | Large unselected population remains filtered |
| Combined worker bound | 256/257 | 256/257 | Now `252/253` substantive siblings plus four HTML/stencil issues |
| Graphview new-document template | structural warnings | 0 issues | One page and unchanged template |
| Managed real-worker clean fixture | 2 | 0 | Exact 598-byte SVG, source identity, process/selection checks |

The managed real-worker file was updated to require an empty warning list, but
its real process path remains gated and **was not executed**. The final package
test ran with `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE*` process variables unset, so that
test skipped. No current smoke was run against `.19` or `.20`.

## Verification chronology

The command and exit-code chronology below was recovered from this owner's
actual turn/tool history. The evidence directory contains captured
stdout/stderr only; no exit-code sidecars were written. Therefore the "source"
column is explicit: assertions about status `0` for empty logs come from the
recorded tool result, not from a fabricated marker inside the log. Failure
statuses are also taken from that recorded turn history.

| # | Command (through `native/scripts/go.ps1 -GoArguments`) | Exit | Log |
|---:|---|---:|---|
| 1 | `fmt ./formats/drawio ./cmd/levnote-drawio-svg ./modules/graphview ./core/engineruntime` | 0 | `gofmt-1.log` |
| 2 | `test ./formats/drawio -run '^(TestCanonicalContainer|TestSVGSuccessMatchesPreChangeOracle)' -count=1 -timeout=180s` | 1 | `focused-drawio-red-or-green-1.log` |
| 3 | `fmt ./formats/drawio` | 0 | `gofmt-2.log` |
| 4 | same focused draw.io parser/SVG-oracle test as row 2 | 1 | `focused-drawio-2.log` |
| 5 | same focused draw.io parser/SVG-oracle test as row 2 | 0 | `focused-drawio-3.log` |
| 6 | narrowed `TestDrawioWorker*` selected-page warning tests, exact IDs recorded in row description below | 1 | `focused-worker-1.log` |
| 7 | `fmt ./formats/drawio ./cmd/levnote-drawio-svg ./modules/graphview ./core/engineruntime` | 0 | `gofmt-3.log` |
| 8 | `test ./formats/drawio ./cmd/levnote-drawio-svg ./modules/graphview ./core/engineruntime -count=1 -timeout=180s` | 0 | `final-affected-test-1.log` |
| 9 | `vet ./formats/drawio ./cmd/levnote-drawio-svg ./modules/graphview ./core/engineruntime` | 0 | `final-affected-vet-1.log` |

Row 6 selected these tests:

```text
^(TestDrawioWorkerExportsExactSecondPageAndPreservesSource|TestDrawioWorkerWarningOrderAndScope|TestDrawioWorkerCompressedReadOnlySuccess|TestDrawioWorkerFiltersLargeUnselectedWarningPopulation|TestDrawioWorkerCombinedWarningBound|TestDrawioWorkerOutputAndWarningLimitsReturnNilPayload)$
```

The failures were useful and are preserved:

- Rows 2 and 4 were assertion corrections. Row 2 exposed nil-versus-empty issue
  slices, a wrong budget fixture, and an incomplete expected parser-issue list
  for the immutable SVG oracle. Row 4 exposed the final element-budget threshold:
  the accepted model plus the two cells permits exactly three counted elements,
  so `3` passes and `2` is over limit.
- Row 6 used `254+extra` substantive `<future/>` elements while an HTML cell and
  a stencil cell also emit parser and SVG-emitter issues. That produced 258 at
  `extra=0` and failed before export. The final corrected arithmetic is
  `252+extra`, producing the required 256/257 bound without weakening it.

## Final affected evidence

`final-affected-test-1.log` contains:

```text
ok  levnote.local/native/formats/drawio              0.725s
ok  levnote.local/native/cmd/levnote-drawio-svg       0.670s
ok  levnote.local/native/modules/graphview           11.369s
ok  levnote.local/native/core/engineruntime          38.040s
```

The four packages were tested with `-count=1 -timeout=180s`. The final vet result
is exit `0` with an empty stdout/stderr log.

Evidence files:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/handoffs/drawio-structural-warning-evidence/final-affected-test-1.log` | 213 | `4CCB9D3FA2864754B2B23DBF65C265202EFA363707618E63985D237FE75A6E7D` |
| `native/handoffs/drawio-structural-warning-evidence/final-affected-vet-1.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `native/handoffs/drawio-structural-warning-evidence/focused-drawio-red-or-green-1.log` | 1,994 | `35FF1B55238BE1D55611A157A1C1B591B434481D9DA1049A980AD61C4956A91F` |
| `native/handoffs/drawio-structural-warning-evidence/focused-drawio-2.log` | 416 | `BFB33B47932BB96E87BD039153183A9A0AEED80CCCE2BC39954467415E163122` |
| `native/handoffs/drawio-structural-warning-evidence/focused-drawio-3.log` | 49 | `F8C16106629BF233BBB77686FF909E821484E60FA52AC18AD541786D6900256C` |
| `native/handoffs/drawio-structural-warning-evidence/focused-worker-1.log` | 266 | `A24335A1BF26DDEACC9BF0F4CC941ADED29E85A754E096C1838484A5113E928C` |
| `native/handoffs/drawio-structural-warning-evidence/gofmt-1.log` | 44 | `C9A489B2E83024A899C5134991073DA842B6567CB9B8DCA9B0B6781B997B3E36` |
| `native/handoffs/drawio-structural-warning-evidence/gofmt-2.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `native/handoffs/drawio-structural-warning-evidence/gofmt-3.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |

## Freeze and limitations

- The source/test files listed above are frozen for independent review; this
  result does not claim or grant acceptance.
- No source or test file changed after final test row 8 or final vet row 9.
- No product build, package creation, application launch, worker launch, physical
  UI run, or resource measurement was performed for this slice.
- The managed real-worker oracle change is compiled by the affected runtime test,
  but its process-launching branch remained gated and unexecuted.
- `.19` and `.20` package/evidence directories and historical handoffs were not
  edited.
- Retained-source optimization and any broader draw.io structural classification
  remain follow-up work, not part of this increment.
