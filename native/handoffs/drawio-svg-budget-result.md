# Draw.io SVG output budget result

Status: implementation and scoped verification COMPLETE 2026-11-10. Product
source, tests, and oracle are frozen below pending independent review. No build,
launch, UI action, capture, benchmark, protocol/catalog, worker, package script,
or product-doc change was made.

## Delivered behavior

- `drawio.SVGOptions` now has `MaxBytes int`. Zero selects the 64 MiB
  `MaxSVGBytes` default; lower positive values are honored; values above the hard
  cap clamp; negatives fail with `ErrSVGOutputLimit`.
- `ExportSVG(pageIndex, options)` keeps its signature and page-lookup precedence.
  Every error path returns nil SVG and nil issues.
- A private `svgWriter` performs one count pass and one exact-size fill pass. It
  counts every emitted byte before advancing and never creates a count-pass
  output buffer. A fill-length disagreement aborts with nil output.
- Issues are collected only in the count pass, so successful warning bytes and
  order remain exactly unchanged and fill cannot duplicate them.
- XML attributes and labels use `encoding/xml.EscapeText` through bounded
  UTF-8-safe chunks of at most 4096 bytes. Valid runes are never split. Invalid
  bytes, controls, quotes, apostrophes, CR/LF/tab, and escapable characters
  retain `encoding/xml`'s exact output. Only each bounded chunk is converted to
  `[]byte`; the whole user string is not converted or escaped into a second
  allocation.
- Shape/edge paint values and font options stream through that helper. Text lines
  are iterated without `strings.Split`, including the trailing-empty-line case.
  Edge point sequences are written directly with the old `M… L…` spelling; no
  complete path string is built. Numeric and fixed fragments retain the old
  formatting.
- The graphview caller still passes zero-valued options and its existing selected
  `v.pageIndex`. No graphview production source changed.
- Parser/source/model/unknown metadata are not part of the SVG byte budget and
  remain unchanged by export.

## Exact pre-change oracle

Before the emitter edit, a temporary old-code dumper ran:

```text
TestDumpPreChangeSVGOracle
PASS
ok levnote.local/native/formats/drawio 0.737s
```

Its log is
`native/handoffs/evidence/drawio-svg-budget-prechange-oracle.log`. The temporary
dumper was removed after it recorded
`native/formats/drawio/testdata/svg-prechange-oracle.json`:

- Exact serialized source and all SVG options.
- Unicode, XML-special, apostrophe/quote, and trailing-newline node/edge labels.
- Custom background and font-family escaping.
- A two-point interior multipoint edge.
- Exact SVG bytes and exact ordered issue values.

The permanent oracle regression parses that recorded source, reapplies the
recorded edge points, exports, and compares SVG bytes and `[]Issue` values
exactly. It also checks source fields, page/model state, and XML well-formedness
through EOF.

## Preserved failures and checks

The first focused contract run was genuinely red before emitter implementation:

```text
TestSVGOutputLimitContract
  cap-minus-one: err=<nil> output=2148 issues=4
TestSVGNewlineExpansionHonorsCapWithoutPartialOutput
  newline amplification: err=<nil> output=66600 issues=0
FAIL
```

Raw log: `native/handoffs/evidence/drawio-svg-budget-focused-red.log`.

The first implementation attempt preserved one compile failure:

```text
formats/drawio/svg.go:298:16: assignment mismatch:
2 variables but xml.EscapeText returns 1 value
```

Raw log: `native/handoffs/evidence/drawio-svg-budget-focused-1.log`. The next
run's regex happened to select only `TestHostileInputsAndLimits`; that green run
is retained as `drawio-svg-budget-focused-2.log` and is not presented as a
feature pass. Corrected contract runs are `focused-3.log` through
`focused-5.log`.

The final focused run covered the self-contained SVG test, exact oracle,
exact/one-byte-short/negative limits, normalization, escaping chunk boundaries,
newline amplification, page lookup precedence, and graphview selected-page
export:

```text
ok levnote.local/native/formats/drawio 0.706s
ok levnote.local/native/modules/graphview 1.344s
```

Raw log: `native/handoffs/evidence/drawio-svg-budget-focused-5.log`.

## Final commands

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./formats/drawio','./modules/graphview','-count=1','-timeout','180s')
ok  	levnote.local/native/formats/drawio	0.721s
ok  	levnote.local/native/modules/graphview	10.850s
exit 0
```

Raw log: `native/handoffs/evidence/drawio-svg-budget-final-test-2.log`.

```powershell
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/drawio','./modules/graphview')
<no stdout or stderr>
exit=0
```

Raw capture: `native/handoffs/evidence/drawio-svg-budget-final-vet.log`.

## Frozen hashes

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/types.go` | 5,089 | `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845` |
| `native/formats/drawio/svg.go` | 15,088 | `9C7FFFCEFFC5217E8E605F3EB2888B38E23F229DE73A551B1753D75010EE680E` |
| `native/formats/drawio/drawio_test.go` | 13,581 | `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189` |
| `native/formats/drawio/svg_budget_test.go` | 9,405 | `FDE94D3587E9753FB15D121B346DB012E5C831107AAEB1A85D7E69A5CCB4E33C` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | 6,688 | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |
| `native/modules/graphview/graphview_test.go` | 9,858 | `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0` |

## Limitations

- The budget bounds emitted SVG bytes, not process memory. Parse structures,
  source copies, page-model indexes, issue slices, formatting state, and the
  exact-size successful output remain outside that cap.
- Streaming escaping still allocates one bounded input slice per UTF-8-safe
  chunk. The two passes repeat deterministic geometry helpers; `byID` is built
  once, but edge point slices are rebuilt in the fill pass.
- The selected page is still host-side graphview state. A future worker protocol
  must carry validated page identity and lifecycle checks; it must not default to
  page 0 or rewrite the source merely to export one page.
