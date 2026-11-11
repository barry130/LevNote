# Draw.io SVG budget: independent review

Status: **CORRECTION REQUIRED** 2026-11-10. This was a read-only source and
evidence review. I ran no tests, builds, application launches, or product edits.

The six frozen identities match `drawio-svg-budget-result.md`:

| File | SHA-256 |
|---|---|
| `native/formats/drawio/types.go` | `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845` |
| `native/formats/drawio/svg.go` | `9C7FFFCEFFC5217E8E605F3EB2888B38E23F229DE73A551B1753D75010EE680E` |
| `native/formats/drawio/drawio_test.go` | `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189` |
| `native/formats/drawio/svg_budget_test.go` | `FDE94D3587E9753FB15D121B346DB012E5C831107AAEB1A85D7E69A5CCB4E33C` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |
| `native/modules/graphview/graphview_test.go` | `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0` |

## Definite defects

### 1. Successful no-warning exports change the issue slice from non-nil empty to nil

`native/formats/drawio/svg.go:104` declares `var warnings []Issue`. If no issue
is appended, `emitSVG` returns that nil slice at line 200 and `ExportSVG`
returns it at line 44. The accepted implementation contract requires
successful output and warnings to remain exactly unchanged; the review brief
records that the old exporter returned a non-nil empty slice constructed with
`make([]Issue, 0)`.

The exact oracle cannot detect this compatibility change because its fixture
produces four issues. `TestSVGSelectedPageAndPageLookupPrecedence` is also
vacuous for this property: `native/formats/drawio/svg_budget_test.go:250-252`
checks only `len(issues) == 0`, which is true for nil and empty slices.

Minimal fix: initialize warnings as `warnings := make([]Issue, 0)` so a
successful warning-free count pass returns a non-nil empty slice. Any failure
must still return nil issues through the existing outer error paths. Add a
warning-free export assertion for both `issues != nil` and `len(issues) == 0`;
a length-only check does not cover the compatibility contract.

### 2. Emission continues doing work after the writer has failed

The implementation task requires stopping at the first writer error:
`drawio-svg-budget-implementation-task.md:16-18`. The writer primitives
short-circuit, but higher-level code continues traversing and evaluating work:

- `writeSVGPath` (`native/formats/drawio/svg.go:406-415`) has no error check.
  After its first failed write it visits every remaining point and evaluates
  `formatFloat` arguments for each `dst.printf`.
- After `writeSVGPath` fails, the edge branch at lines 125-133 continues through
  the remaining attribute writes, computes `pathMidpoint`, and calls
  `writeSVGText` for a labeled edge.
- `writeSVGText` (`svg.go:373-404`) counts every newline at line 377 before
  checking whether the destination was already in an error state.
- After `writeShape` fails, the node branch at lines 182-185 can still enter
  `writeSVGText`, repeating the full-label scan after the cap has already
  refused output.

This does not leak partial SVG or issues, but it violates the explicit bounded
failure behavior and leaves potentially large post-refusal work in the count
pass.

Minimal fix: return immediately from `writeSVGText` when `dst.err != nil`;
check `dst.err` at the start of each `writeSVGPath` iteration; and have
`emitSVG` return as soon as `writeSVGPath` or `writeShape` fails, before
computing optional text locations or scanning labels. Existing writer-method
short circuits can remain. No new abstraction is needed.

## Evidence and limits

The exact 2,148-byte SVG oracle and its four ordered issues exercise byte,
Unicode, escaping, option, newline, and warning-order compatibility. Source
inspection confirms page lookup occurs first, limit failure returns nil output
and nil issues, the resolved limit is passed to the count writer, fill uses the
counted length, and selected page/source/model state is not modified.

`TestSVGEscapingChunksPreserveEncodingXMLSemantics` compares against the
independent `escapeAttr` implementation, which wraps `xml.EscapeText`. Its
small chunk sizes cover valid multibyte and invalid-byte splits. It does not
place a multibyte rune immediately across the production 4096-byte boundary.
That is a narrow evidence gap, not a demonstrated defect; one boundary case may
be added while correcting the above tests, but it is not a separate acceptance
gate.

No test pushes an export across the resolved 64 MiB default. This is not a
separate defect: the resolver is directly covered, lower caps exercise the same
count/fill refusal path, and the source unambiguously passes the resolved value
into `svgWriter`. A real 64 MiB fixture would add cost without testing a
different code branch, so no default-size matrix is required.

Both `drawio-svg-budget-final-test.log` and
`drawio-svg-budget-final-test-2.log` are successful and contain the same
drawio/graphview passes with only timing differences. `final-test-2` is the
later duplicate cited by the result; there is no failed first final-suite log
being hidden. Earlier genuine failures remain visible in `focused-red`,
`focused-1` (compile failure), and `focused-2` (an overly narrow test
selection, correctly disclosed by the result). The report could note the
duplicate rerun, but this is not itself a source defect.

## Acceptance scope

Accept after the two source corrections and the warning-free nil/empty
regression are present. Re-read the changed `svg.go` and test hunk, verify
updated hashes, and run the affected drawio test/vet once if requested. The
existing graphview result need not be repeated because no caller behavior or
selected-page path requires changing. A correction reconciliation should not
expand into a 64 MiB stress matrix or a general output framework.
