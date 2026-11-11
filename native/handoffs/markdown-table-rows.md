# Table body-row actions — focused implementation handoff

Status: DELIVERY ACCEPTED2026-11-09; final source documentation complete.
Root accepted the focused pure/native and final consumer/source gates, including
the compact panel and rapid-click continuation correction. Current full Markdown
view/app plus vet pass; two integrated captures pass geometry/recovery oracles
and were visually inspected. The authorized fresh.3package and existing real
host-path acceptance have now completed and root accepted the actual12/12
aggregate, validator, RUN/PASS streams and shipped README/manifest. The separate
`../artifacts/acceptance/markdown-table-rows-20261109-040306-package/root-acceptance.json`
records that explicit acceptance with its own actual creation time and computed
digests. Product/tests/scripts and accepted packages/evidence remain frozen.
This follows root's source-reviewed plan, not an independent-agent endorsement.

Package evidence is in
`../artifacts/acceptance/markdown-table-rows-20261109-040306-package/`.
`aggregate.json` passes12/12, SHA256
`9C1FE658598A48D2B07AAB910AE45C22BDD4B0CAFE4572B98E700D78D3FAA345`;
the adjacent `aggregate.ps1`, command records and actual stdout/stderr describe
the checks. No failed package or metadata run was overwritten in this gate.

The clean launch directory is `../bin/levnote-native-worker-20261109.3/`, version
`native-worker.20261109.3`:19files,16,942,149bytes. It contains13exact source
license/attribution copies,2authored documents and the three runtime payload
files; the release manifest lists18unique relative entries and excludes itself.
All19authored local links resolve within the package. README now documents row
actions/F6, separate cell/row undo and the difference between a recoverable
failed cell draft and an already-written cell whose row operation was refused.
Configuration remains `%APPDATA%/LevNoteNative`; program-folder portability does
not make user configuration self-contained.

| Artifact | Bytes | SHA256 |
|---|---:|---|
| `levnote-native.exe` | 13,591,552 | `61512C03DDB9F63EB51B59A09C8960B3B52333F7B3E0F44BB0F3410DD3FA2E44` |
| `engines/markdown-export/levnote-markdown-export.exe` | 3,295,744 | `6CFE7157B40F974C08A8DC88D920E2619C5A814D4E1AF88070DFBCC5FF4F43F3` |
| `engines/markdown-export/manifest.json` | 337 | `0DF33AA2FD39B9A606AE4F24BEFFE56D18F21EBF6BFB951A5F0256FBB6D14B08` |
| `release-manifest.json` | 3,411 | `0AE777D6B1BE26095987E7422AAE86C3EC4064FBB4C12D314EA80B3291A3FFE2` |

One package invocation passed in3.9761635s; its worker/host builds took1.139006s/
1.9588919s with exit0 and empty stderr. The app tester compiled once in1.6212914s
into `host path copy/`, outside the clean launch directory. One invocation of
`TestWorkerExportBundledHostPathAcceptance` actually RUN/PASSed in0.19s (command
wall0.8690875s), making **two real worker requests**: export, then existing-target
refusal. The source-reviewed test verifies independent HTML semantics, exact
BOM/source/encoding/history/selection/baseline preservation and no overwrite;
actual output331bytes, one warning, two dialog callbacks, RuntimeJoined=true.
This is a registered app route with injected SaveDialog and real fixed-path
worker, not a packaged-host UI/physical-dialog or per-PID-observer claim.

All19disposable package copies stayed identical after testing. The aggregate
also checks complete inventories and hashes for1,025source/test/vendor/script/
template/assets inputs,15,657pinned toolchain files, and179protected earlier
files (including all preexisting bin contents, prior table-row successes and
failures, and original worker candidate records). Protected UTCmtime ticks also
match. Existing.1/.2/marker artifacts are untouched. No resource/runtime/full
repository suites or new UI launch were run; .1/.2resource observations are not
measurements of this new.3build. Source README/VALIDATION/API/ARCHITECTURE and root
README/ROADMAP/CHANGELOG now describe the accepted.3subset. No additional product
execution accompanied documentation; the portable files were not recopied.

The implementation is confined to `formats/markdown/table_edit.go` and its
tests, `modules/markdownview/table_rows.go` and its tests, plus prepared cell
validation and local lifecycle/focus seams in `table_cell.go`, `draft.go`,
`view.go`. Parser/renderer/editor core/app/runtime/protocol/vendor/scripts and
accepted outputs are unchanged by this slice.

Consumer tests add only `app/markdown_rows_app_test.go`. The registered adapter
is activated through real MyGo pointer/F6/Tab/Enter/Space input. Independent
oracles verify header/body insertion and exact deletion, BOM/CRLF and surrounding
prose, clean-cell one undo vs changed-cell two undos/redos, oriented source
selection, source/split/preview switches, save/reopen and exact escaped/literal
HTML table row order. Reference `View.Export` is explicitly identified as the
in-process reference path; no worker invocation is implied. Stale source and
post-cell row-limit refusal preserve the correct source/history/recovery state.

Current consumer evidence:

- `../artifacts/acceptance/markdown-table-rows-20261109-035447-consumer-attempt02/`:
  focused app PASS1.549s, wall4.0140581s, exit0; both760x480 integrated split
  captures saved in `images/`. Every finish/discard/recovery/row button has an
  actual native RoleButton bound within x0..760/y0..451; dark recovery is then
  clicked and produces the exact failed draft. The light fixture confirms
  composition with Type, then Finish proves the exact value once, no duplicate.
- `../artifacts/acceptance/markdown-table-rows-20261109-035434-layout-view-attempt02/`:
  affected row+cell suite PASS1.425s, exit0.
- `../artifacts/acceptance/markdown-table-rows-20261109-035822-consumer-final-attempt02/`:
  final full view PASS2.306s and app PASS5.703s, test wall8.1742019s; scoped
  view/app vet exit0, wall1.2836549s. All eight input hashes unchanged through
  checks, no opt-in variables enabled; unrelated worker/image tests skip.
  The unchanged format source retains the accepted0.647s result below.
  The earlier complete consumer run remains preserved; the final repeat was
  required by the subsequently demonstrated multi-click guard correction.

Consumer attempt01 preserved the actual dark screenshot with row buttons at
y515.73 (below status451), plus a test-only disabled-focusability expectation.
Root authorized a constant48px **cell-only** editor, one short combined cell/row
error inside the draft, and full tooltip/Warnings/Issues details. The normal
block editor and app chrome are unchanged. Staged and error states keep the
same editor height and fixed hint slot; current controls all fit above status.

The first layout-view run and the bounded held-key diagnostic are also retained:
the test's repeated two-click opener inherited the previous click count, so its
first click reopened the cell and its second hit a control in the newly compact
panel. A distinct physical gesture now starts after one second of **tester
virtual time** (no sleep), with source/history/draft/focus assertions before the
repeat check. Root then required a separate regression for the actual rapid
continuation rather than accepting that timing separation as a product fix.
`035733-rapid-red-attempt01` proves a later continued click intersects the new
delete button and removes the row before the fix. The row-specific guard now
reads the **backend click count in the ancestor ObserveInput** before MyGo
renormalizes it for the new hit control, and declines continuation captures.
Clicked still requires a valid capture; keyboard remains unchanged. The flag
clears on release/new down/cancel/scroll/draft release. The regression uses no
time advance before the risky click, asserts intersection, exact unchanged
source/history/draft, then proves a later fresh single click still acts once.
`035759-rapid-focused-attempt01` row+cell PASS1.470s and the final full suites
above cover this guard. Capture images were not rerun because this input-only
guard changes no rendered state. These simulated controls do not prove physical
OSIME or all possible operating-system pointer sequences.

Native evidence:
`../artifacts/acceptance/markdown-table-rows-20261109-034215-native-attempt05/`
contains the actual command, stdout/stderr and UTC/wall/exit result. Eleven
top-level tests pass, including actual pointer and keyboard controls, composition
and two undos, fixed pressed-button bounds, callback invalidation, real scroll,
recovery and settled frames. Exact duplicate-row/selection mapping and history
expiry also have direct action tests; those are not claimed as native activation
coverage. Five unchanged frames in draft/refused/success/canceled states keep
the same parsed block/row backing arrays, empty gesture and no FramePending.

Preserved failures in separate native attempt01–04 directories: attempt01
test-only float32/int tester width compile mismatch; attempt02 attempted Tab
from an editor that intentionally inserts indentation, plus a selection test
that set the already-current position; attempt03 exposed native focus-loss
UnmarkText and scroll bypass of ObserveInput; attempt04 exposed a composition
hint disappearing on pointer-down and moving the pressed buttons. The final
code adds local F6 composition protection, normal scroll handler cancellation,
and a fixed30px/two-line hint slot. No failed logs were overwritten.

Final focused evidence:
`../artifacts/acceptance/markdown-table-rows-20261109-034342-focused-final-attempt01/`
contains `run.ps1`, explicit-array `go-check.ps1`, actual streams, per-stage
results and before/after fingerprints for all seven implementation/test inputs.
`go.ps1 test ./formats/markdown ./modules/markdownview -count=1 -v` passes:
format0.647s, view2.518s, overall wall3.7646001s, exit0. Relevant two-package vet
passes exit0, wall1.0301581s. Seven source/test fingerprints are unchanged through
the checks, and no LEVNOTE opt-in environment variables were enabled. Existing
three screenshot tests correctly skip; no new image, app, bundle or worker run.

## Pure seam

Implemented `formats/markdown/table_edit.go` API:

```go
type TableRowAction uint8
const (
    TableRowInsertAfter TableRowAction = iota + 1
    TableRowDelete
)
type TableRowPatch struct {
    Range SourceRange // half-open byte offsets in the supplied full source
    Text  string
}
func PlanTableRowEdit(source string, tableRange, rowRange SourceRange,
    action TableRowAction) (TableRowPatch, error)
```

The helper checks UTF-8/8MiB before parsing the authoritative source, requires
a complete supported table at the exact table range and one exact physical row
range in its freshly parsed descriptors. Unknown action, invalid/forged ranges,
header deletion, malformed/partial/over-budget source return a zero patch/error.
The view supplies current descriptors after its own identity checks; row index
or equal row text alone never selects a row. No exported model is trusted merely
because its fields look plausible. No Buffer/filesystem/UI access.

Insert constructs only one empty leading/trailing-pipe row (`| | |` for two
columns), with the selected row's legal indentation. Header insertion anchors
after the delimiter. Prefer the insertion anchor's exact LF/CRLF/CR, falling
back to the closest preceding table ending then LF. At unterminated EOF append
separator+new row without a final ending; otherwise insert new row+ending.
Delete removes exactly a body row.Range including its own ending; the previous
row's ending is untouched. All other bytes, including BOM/prose/padding/escapes,
are retained.

Before returning, bounded candidate construction/checking requires the exact
one-row shape difference and existing8columns/256body rows/4096cell bytes/
32KiB table limits. Parse the **whole candidate**, verify !Partial and compare
unaffected blocks/tables at precisely delta-translated ranges: their Raw, kind,
table Problem, alignments and row/cell descriptors remain identical. This also
catches global16384cell descriptor exhaustion, which can make a later table
fallback while Document.Partial stays false. Checking only the edited table
or !Partial is insufficient. Meaningful oracle: an early valid table plus
later valid tables consume exactly the global budget; insertion must refuse
without a patch and without changing a later table into fallback. Deletion and
unrelated existing fallback behavior are also checked explicitly. Full parse/
candidate work occurs only on activation; unchanged frames do none of it.
The conservative unchanged-context policy also refuses a deletion when freeing
descriptor budget promotes a later fallback table into a supported table. This
is `ErrTableRowContext`, reported as changed parsing of other content, rather
than an exceeded-limit error. Unrelated malformed fallback remaining unchanged
does not prevent deletion. The accepted pure run is
`../artifacts/acceptance/markdown-table-rows-20261109-032721-pure-attempt01/`
(package0.521s, exit0).

## Native action and own-Flush proof

Two focusable/wrapping buttons appear only in the selected-cell draft panel:
`在下方插入行` and `删除此数据行`. Header delete is disabled with a short reason;
hint explains the current cell writes first and each resulting edit is undoable.
Native down captures action, exact cellEdit/draft+draft Buffer/version/selection,
host Buffer/version/selection, bounded target and lifecycle generation. Pointer
cancel/scroll/release/new draft or target clears capture. Enter/Space repeats
cannot double commit. A current-source resolution at capture accepts a valid
outside-table rebase already present then; an edit during the held gesture
invalidates it. Re-read authoritative context before acting, not cached Draw.

Keyboard entry is local **F6 行操作** from the focused cell draft. The first
native row button receives focus; Tab/Shift+Tab traverse buttons and Enter/Space
activate once. Tab's shared text-indent behavior is preserved. While a native
composition is live, F6 keeps focus, exact draft bytes/selection/version/group
and host source unchanged, showing “请先确认输入法组合，再按 F6 进入行操作。”
After the normal native-client confirmation F6 moves focus. This guard is needed
because MyGo normally UnmarkText on focus loss; no editor/framework behavior is
overridden. Pointer actions can confirm composition through that existing seam.
The hint occupies one fixed-height two-line slot so confirmation on pointer-down
does not move the button before pointer-up. A test compares its exact bounds.
Scroll/zoom cancellation uses local HandleInput returning false, since MyGo's
scroll delivery does not pass through ObserveInput. Real tester scroll both
cancels the held action and moves the preview without source mutation.

Narrowly factor the existing cell validation into a private prepared plan shared
by normal syncCellDraft and row activation. Finish native composition using the
draft editor's Flush first, then recheck the same draft/buffer/version/selection;
current UnmarkText ends its composition group without replacing source bytes.
Retain a bounded expected table string, source anchor, exact selected row range,
host version and expected selection for the subsequent normal View.Flush.
No parsed block/node pointer survives that Flush.

Own-Flush receipt rule: unchanged cell means zero host version steps and the
captured host selection unchanged. A real cell replacement means exactly one
version step, same host Buffer, exact predicted table at the resolved start,
and selection collapsed to `cellStartUTF16 + UTF16Len(value)` because the current
ReplaceUTF16 sets it there. After Flush, reparse the actual complete document
and recheck lifecycle generation, exact table/descriptors, Buffer/version and
this expected selection. Callback edits, selection-only changes, release or
replacement refuse the row even if visible text is identical. Refusal after a
successful cell write explicitly says the cell was written but the row was not;
failed cell Flush retains the existing pending draft and recovery controls.

The **row transaction's selection baseline is this verified post-Flush source
selection**. Thus a clean draft preserves its original (including reversed)
selection; a changed draft uses the normal collapsed caret left by the cell
write. Map Anchor and Head independently through the row patch, preserving
orientation and valid UTF-16 rune boundaries. For insertion, endpoints at the
insertion position have right affinity. For deletion, points in [start,end)
clamp to start; points at/after end shift by delta. Compare the unchanged
baseline again immediately before commit. BeginGroup only when no foreign
group exists, ReplaceUTF16 and SetSelection in the same group, EndGroup once;
CancelGroup only if this action successfully began it. This yields zero/one
cell undo plus one row undo, with row Undo restoring the post-cell selection.

Own normal clearDraft invalidates stored gestures but can be recognized by the
local receipt; a separate Release generation rejects synchronous release even
though it also clears the draft. New draft/buffer/readonly checks run after all
callbacks. Success preserves preview location with existing HoldPlace, clears
stale row/cell state and reparses normally. This slice need not automatically
focus an inserted cell; user selection keeps the identity contract simple.

## Finite ownership and checks

Pure helper/new tests in `formats/markdown/table_edit*`; no parser/renderer
behavior change. New row action/tests in `modules/markdownview/table_rows*`,
only narrow existing view/draft/table_cell lifecycle and prepared-plan seams.
Later app integration tests/images within the root plan, no runtime/protocol/
package changes until a separate integration gate.

First exact-byte pure tests cover LF/CRLF/CR, EOF,0/1/256body rows,1/8columns,
BOM/Unicode/escapes/no-outer-padding and adjacent text, forged target and all
local/global budgets. Then native pointer+Enter/Space once, pending composition,
own0/1version+selection receipt, same/outside-table mutations, duplicates,
callback selection/release/replacement, history expiry, readonly/foreign group,
recovery and oriented astral selections/undo. Verify unchanged-frame idleness.
Root reviews the final focused source/oracles/evidence before consumer/package
work. App consumer checks are recorded above; no new package/worker/resource
or physical OSIME acceptance is claimed here.
