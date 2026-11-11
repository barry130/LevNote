# Markdown selected-column alignment — implementation and consumer handoff

Status: FOCUSED, CONSUMER AND .5 DELIVERY ROOT ACCEPTED; final docs complete, 2026-11-09.
The API/shared-action design is [the accepted checkpoint](markdown-table-alignment-checkpoint.md).
Root read actual focused and consumer logs/oracles, both corrected app captures,
and the [independent source review](markdown-alignment-independent-review.md),
which has no actionable findings. Product/tests remain frozen after the
[accepted delivery](markdown-table-alignment-delivery-next-stage.md). Accepted `.4` and all
earlier bundles/evidence are untouched. No app production/core/parser/renderer/
runtime/protocol/vendor or packaging-script changes, UI launch or resource run.

## Implemented behavior

`formats/markdown/table_alignment.go` exports `PlanTableAlignmentEdit` and
`TableAlignmentPatch{Range,Text,Noop}` plus Value/Target/Limit/Context sentinels.
Only the selected delimiter marker changes, retaining hyphen count, padding,
pipes, every other source byte and exact endings. Current and full candidate
parse use the existing scanner; table shapes, all unaffected block semantics,
descriptor fallback and warning offsets are checked. Invalid/error returns zero
patch; semantic equality returns only `Noop:true` without a candidate snapshot.
Positive colon growth is checked before candidate allocation against4096cell/
32KiBtable/8MiBsource limits. Existing parser/global descriptor limits remain.

`modules/markdownview/table_actions.go` now owns the single native gesture,
composition, own-Flush receipt, selection/group commit, input cancellation and
error completion pipeline. Row/alignment files provide their planner and button
definitions; there is no second alignment lifecycle. `tableCellPlan.column` is
verified before Flush, and exact post-Flush table bytes are required. Ordinary
cell text is its own edit; alignment is a separate undo. Success and semantic
no-op both finish/close the panel. Clean no-op adds no version/history/Changed;
pending-cell no-op keeps only the cell write. No old draft is recreated after
callbacks. Failed cell Flush retains its recoverable input; post-cell refusal
explicitly says “单元格已写回，对齐未修改”.

Completion errors always remain eligible for Host.Report but attach locally only
to the captured source/lifecycle and expected source version/selection. Before
cell completion they also require the same draft buffer version/selection.
Changed/Report can replace a source/draft, release the view or edit the same
buffers in place without receiving a stale local error afterward. There are no
writes after Report. Old row callback tests now distinguish current ownership
from released/replaced/source-modified ownership while retaining exact source,
selection, group and host-refusal checks.

The panel keeps48px input and one fixed30px hint. The existing title reports the
selected column/alignment from bounded descriptors; stale version shows 待重验.
Four46×26alignment buttons with4px gaps fit196px, stay on one row, and have full
column-specific accessible names. Row and cell completion/recovery buttons use
short faces with their old full accessible names. Header-delete explanation is
visible in the existing hint slot; no extra height. F6 enters existing row actions,
Tab/ShiftTab reaches alignment. Marked composition keeps F6 focus in the editor;
pointer confirmation leaves pressed button bounds unchanged. UTF16 endpoint
mapping retains the row action rule; points inside the replaced delimiter marker
clamp to its start, disclosed in the tooltip.

## Actual finite verification

Final evidence:
[markdown-alignment-20261109-053920-owned-final-attempt05](../artifacts/acceptance/markdown-alignment-20261109-053920-owned-final-attempt05/result.json).

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./formats/markdown','./modules/markdownview','-count=1','-v')
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/markdown','./modules/markdownview')
```

The evidence child clears LEVNOTE opt-in variables; existing screenshot tests
correctly SKIP. Actual full formats **PASS0.809s**, view **PASS2.499s**; combined
command wall4.4817699s, exit0. Vet wall1.0456751s, exit0, empty streams. All10owned
source/test fingerprints remain unchanged during final execution.

- `result.json` SHA256 `0A96BE7B3A5BBD6FEC902655BC9773D10B16AC3ECD114397BBC705DA2C07D917`.
- `test.stdout.log`81,462bytes SHA256 `FDB8520248F385B100E2D75ECAB9CB4E8BD928B9617D76F6C56FC05EFD5C3D8C`.
- `source-before.json` and `source-after.json` give all ten exact input identities.
- The unchanged pure helper first passed in
  [pure-attempt01](../artifacts/acceptance/markdown-alignment-20261109-052631-pure-attempt01/result.json),
  actual package0.538s. Its source SHA256 is
  `6A1C238D6B61CA306AE8AE229529C0716071EE4F1FFC45EDABD0A1B95E120E91`;
  test SHA256 `046707C84ABA5196B9227CE014CC8C54580836EDAC483C0922C68AE06DB4DD69`.

Five pure tests cover all16alignment transitions/no-op, exact syntax and unrelated
block context, invalid identities, exact size limits with growth refusal/shrink
success, and global descriptor exhaustion preserving a later fallback without
Partial. Expected bytes are independent literals/fixture assembly.

Eleven new native tests, all present in final RUN/PASS streams:

- `TestTableAlignmentNativePointerKeyboardAndUndo`
- `TestTableAlignmentNoopClosesPanelWithoutSecondEdit`
- `TestTableAlignmentSelectedHeaderColumnReadonlyAndPriorOutsideEdit`
- `TestTableAlignmentCompositionGeometryAndSeparateUndos`
- `TestTableAlignmentNarrowBoundsAndRecovery`
- `TestTableAlignmentPostCellLimitRefusal`
- `TestTableAlignmentRapidContinuationAndHeldRepeat`
- `TestTableAlignmentHeldSourceIdentityAndScroll`
- `TestTableAlignmentChangedAndReportOwnership`
- `TestTableAlignmentReportInPlaceChangesDoNotReceiveOldLocalError`
- `TestTableAlignmentSelectionAndIdle`

They use native Pointer/Key events for normal activation, not direct activation
as a substitute. Narrow checks assert actual enabled/Focusable/Offscreen/bounds
and same-Y46px controls at265/760wide, including invalid-draft recovery. Scroll
checks prove actual preview scroll advances while the held action cancels.
Continuation fixtures assert the old cell point lies inside the new row/delete
or alignment button before a rapid no-Advance click; a later single-click works.
Five settled frames retain parsed block/row/draft identity, no pending frame,
capture, source version or history changes. Existing row/cell package regressions
also pass, including foreign groups, callback replacement and separate undo.
These are isolated native-view checks; the451bottom threshold is not a substitute
for a real app status bar. The actual app geometry is verified separately below.

## Accepted app consumer

Only `app/markdown_alignment_app_test.go` was added during this gate; no product
or existing app tests changed. Its19,927bytes SHA256 is
`0CF4E9F4CD123BFCA70CFE29D51B58873E12B665D4C9A7279B9C3440E144D9B8`.
Four top-level tests contain three ordinary functional entries and one opt-in
capture entry with two themes:

- `TestMarkdownAlignmentRealAdapterExactNoopSaveReopenModesAndHTML`
- `TestMarkdownAlignmentRealAdapterPendingCellAndNoopUndo`
- `TestMarkdownAlignmentRealAdapterRefusalRecoveryReadonly`
- `TestMarkdownAlignmentAppNarrowCaptures`

The registered adapter is exercised by real native cell/button gestures, including
F6/Tab/Enter. Independent BOM+CRLF, two-table, Unicode, escaped-pipe and literal
HTML fixtures prove exact marker-only header/body changes, clean semantic no-op,
one or two undo/redo entries, mode transitions, save/reopen, and unaffected source
and disk. Split-source conflict retains the exact draft for real app RecoverDraft
into a new unsaved document. A cell write followed by a delimiter limit refusal
retains its one undo and explicit “单元格已写回，对齐未修改” message. Readonly source
does not expose editing actions. Reference View.Export runs in process here;
this gate did not start Markdown worker processes.

Actual evidence:

- [consumer-attempt01](../artifacts/acceptance/markdown-alignment-20261109-055040-consumer-attempt01/result.json):
  focused alignment plus existing row consumers PASS1.741s. Its images and
  initially lenient text-Y-minus-padding status formula remain preserved. The
  light image had a normal keyboard tooltip obscuring row controls; no product
  defect was found.
- [capture-attempt02](../artifacts/acceptance/markdown-alignment-20261109-055235-capture-attempt02/result.json):
  actual capture test PASS1.675s. A real neutral pointer Move plus Escape closes
  the native keyboard tooltip; alignment focus and uncommitted source remain,
  and the subsequent Enter commits the exact cell/alignment as two undos. The
  boundary uses the actual status text center and declared29px row height,
  yielding451px. Preview width264px, input226×48px, lowest dark action bottom
 443.8213px; all required actions are enabled/focusable, not Offscreen and inside
  the actual preview/status bounds. Pointer composition confirmation preserves
  held button bounds. Full draft/current-column/error/recovery context is shown.
- [consumer-final-attempt03](../artifacts/acceptance/markdown-alignment-20261109-055306-consumer-final-attempt03/result.json):
  full app PASS6.441s, command wall7.8643048s; scoped vet wall1.3845038s, exit0.
  All14recorded inputs unchanged, including the10accepted focused inputs.
  Screenshot/worker opt-ins were cleared for the ordinary full package run.

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMarkdown(Alignment|Rows)','-count=1','-v')
# capture-attempt02 sets only LEVNOTE_MARKDOWN_ALIGNMENT_SCREENSHOT_DIR:
.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMarkdownAlignmentAppNarrowCaptures$','-count=1','-v')
.\native\scripts\go.ps1 -GoArguments @('test','./app','-count=1','-v')
.\native\scripts\go.ps1 -GoArguments @('vet','./app')
```

Final result SHA256 `42F31655EA9D564608EFD6A3FC6911878A99BD34F8B5D18436BCAD74AB429FDE`;
source-after SHA256 `38D28E96CF7C4B3C68F88BE6003FC08BD6D5C2D839B27869E579A24CCDC750EA`;
full test stdout61,428bytes SHA256
`1AA940E88602B53CA77DE73AC4E0F5C9D546F10708952E2BC6D39F9B46519ADF`.
Final captures:

- [light selected column](../artifacts/acceptance/markdown-alignment-20261109-055235-capture-attempt02/images/markdown-alignment-light-column-760x480.png),
  41,831bytes SHA256 `6F740DEB52525100771A3D40E11104C7ACF9948778445CD17D683B5F1E2B1081`.
- [dark recovery](../artifacts/acceptance/markdown-alignment-20261109-055235-capture-attempt02/images/markdown-alignment-dark-recovery-760x480.png),
  51,010bytes SHA256 `4B65AC4F78647F3042F2B05C92FED047C328ABC3C50E82C82C7B9171FBC1039F`.

The independent review is source-only; these captures use MyGo simulated input.
Neither establishes physical OSIME/GPU, external accessibility, fresh package or
resource acceptance. Unchanged format/view acceptance is reused, not rerun.

## Fresh .5 delivery checkpoint

Candidate: `native/bin/levnote-native-worker-20261109.5`, engine version
`native-worker.20261109.5`. Only the portable README usage template changed after
consumer freeze. The existing packaging script ran once; no product/test/script
changes, new full-UI launch, resource measurement or unrelated suite replay.

[Package aggregate](../artifacts/acceptance/markdown-alignment-20261109-055547-package/aggregate.json)
is **PASS12/12**,19files17,038,894bytes. Its validator independently reads actual
Windows amd64 GUI PE headers, manifest identity/version,18relative release
entries,13exact source/license assets, substituted README and19local links.
The1037source/package inputs,15,657toolchain files,530protected prior files
(inventory/bytes/digests/UTCmtime ticks) and19disposable copies stayed unchanged.

- Aggregate SHA256 `A32B88174E17EF6BA50190EC9C83A4840F1E0C4A762C6AD5B2BFAB2960AA5C42`.
- [Validator](../artifacts/acceptance/markdown-alignment-20261109-055547-package/validator.json)
  SHA256 `EC2FF92D9E381B2D5A97D2F80D88A688241CB46E3858CB469803B4E28D80CFEB`.
- [Validator source](../artifacts/acceptance/markdown-alignment-20261109-055547-package/aggregate.ps1)
  SHA256 `B518999FAF99B6A2EB26278DBCAC01F914A334BF54751A726EF6A88E3BCFF304`.
- Host13,686,784bytes SHA256
  `1272E5B13D1331B81A92111E242D9ACCF153363C6BFDC9F8748A11DC19BE33C7`.
- Worker3,295,744bytes SHA256
  `DC3E0F42BA27961DFEE714C205AB48A8C0722BADCFB3546CE4C6D25CEA68AAAA`.
- Engine manifest SHA256
  `BA5BBD0B9D747319A549764580D6D93BF5AF967F0162270710B23A427A29E443`;
  release manifest SHA256
  `77D5BE4228ADF28D15637741FF8D966B28B5F6A90295E010D9742AC9D1C123E3`.

Actual package command4.330298s, tester compile1.8377442s, HostPath invocation
2.535937s, all exit0/empty stderr. The complete package was copied to evidence;
tester/logs/helpers stay outside the user bundle. Existing
`TestWorkerExportBundledHostPathAcceptance` ran once (actual PASS0.20s) and
made two real worker requests, successful export then exclusive-target refusal.
Its independent331byte HTML/BOM/source/history assertions pass, dialogs2 and
RuntimeJoined true. This is the source-reviewed joined-return contract; this
gate has no separate per-PID observer and does not claim packaged-host UI tests.
Original streams, command metadata and inventories are linked from the aggregate.
Root read these exact sources/streams/checks and accepted the package. Source
README/VALIDATION/architecture/format API, root README/ROADMAP/CHANGELOG and this
handoff now describe `.5`; old `.4` and resource results remain historical.
The [separate actual-time acceptance record](../artifacts/acceptance/markdown-alignment-20261109-055547-package/root-acceptance.json)
records root's scope and final doc snapshot. The package, aggregate and validator
remain immutable; no product execution followed acceptance.

## Preserved attempt history

- [native-attempt01](../artifacts/acceptance/markdown-alignment-20261109-053513-native-attempt01/result.json):
  compile FAIL from unused test variable `v`; no behavior tests ran.
- [native-attempt02](../artifacts/acceptance/markdown-alignment-20261109-053542-native-attempt02/result.json):
  focused suite had two fixture failures: requested selection split a UTF16
  surrogate pair (Buffer correctly refused), and old rapid-click point203.5no
  longer intersected compact delete X90..153. All other alignment/row/cell cases,
  including four Report in-place mutations, passed. Corrected the legal endpoint
  using UTF16Len prefixes; core remained untouched.
- [native-attempt03](../artifacts/acceptance/markdown-alignment-20261109-053706-native-attempt03/result.json):
  legal selection/idle passed. New alignment rapid point was1.9px below button;
  changed row opener point differed from initial opener, so native click count
  did not continue. Preserved non-vacuous hit assertions and corrected initial/
  continued physical points together; no guard weakening or time advance.
- [native-attempt04](../artifacts/acceptance/markdown-alignment-20261109-053747-native-attempt04/result.json):
  both rapid native regressions and held-repeat PASS. Final full owned gate above
  includes these corrections and the selected header/readonly/prior-rebase test.

Final focused changed files: format `table_alignment.go`, `table_alignment_test.go`;
markdownview `table_actions.go`, `table_alignment.go`, `table_alignment_test.go`,
`table_rows.go`, `table_rows_test.go`, `table_cell.go`, `draft.go`, `view.go`.
The focused source fingerprints remain unchanged through the accepted consumer
gate. Only the new app consumer file was added afterward. The fresh package
evidence above is separate; all prior results and images remain immutable.
