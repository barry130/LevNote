# Table-column UI and app integration — source frozen

2026-11-10. Implemented the active UI scope against the format owner's exact
`PlanTableColumnEdit` API. Eight owned files are frozen for root review and the
separately activated combined app/view/format test and vet. .24 package,
portable README and sealed evidence were not changed.

The cell panel now has `前插列` / `后插列` / `删此列` in a short wrapped row
between row and alignment actions. Accessible labels are `在此列前插入列`,
`在此列后插入列`, `删除此列`. Total columns 1/8 control disabled deletion/insertion;
readonly disables all. F6 still first focuses `在下方插入行`; enabled controls
traverse deterministically in both directions. Existing composition confirmation,
fixed-height hint, draft flush and finite recovery paths are reused.

The shared transaction minimally changes single-edit row/alignment adapters to
one-entry edit lists. All original byte ranges, replacement UTF-8 and boundaries
are checked before conversion to original UTF-16 offsets. Nonoverlapping edits
apply descending inside one owned group; both selection endpoints use the same
order and existing insertion affinity. Selection is set once before EndGroup,
Changed once afterward, with no callback inside the group. Foreign groups are
not consumed. CancelGroup restores bytes/selection/history/redo but may advance
revision. Owner/source/version/selection/epoch checks remain before and after
normal Flush. A changed cell and structural edit remain two separate undo steps;
post-Flush refusal explicitly says `单元格已写回，列未修改`.

## Files and exact identities

[Frozen inventory](markdown-table-columns-ui-evidence-20261110/04-source-freeze.json)
contains all eight paths, lengths, hashes and freeze time.

| File under native/ | Bytes | SHA256 |
|---|---:|---|
| modules/markdownview/table_actions.go | 13,048 | `1CFE1B0C2C876B39FE262ACF354B63162B36FBBCFBB0E2406E94CBE1BB9AD141` |
| modules/markdownview/table_rows.go | 2,311 | `2B59DDB8B16D607E1E535F20069DB2B09925995E55DEFCC1BDBBAF1AABEAC345` |
| modules/markdownview/table_alignment.go | 2,768 | `418356BD18CAC2FFFEB910036322FA3097BB5D49658CBE4FAE78397707D6C614` |
| modules/markdownview/table_columns.go | 2,353 | `8E39C1CD4D63BD7BF649BF4D20E5D659E43F828CF8242DC1336F8D12FAFF74DF` |
| modules/markdownview/table_columns_test.go | 23,456 | `165FC4221EB49735091FDF4B823A9CE371B56B8052EE13FC75A28E59C3533888` |
| modules/markdownview/table_alignment_test.go | 22,344 | `851A47223DDFB553D9CC8461A0BE218E980BD592EDB5CCAA8CF4397B544DE7A4` |
| app/markdown_columns_app_test.go | 4,096 | `2F414217DF18DEBE1D345EEDA81D608D1FBDE2DAC99739AEFB0392B278EF135A` |
| app/markdown_alignment_app_test.go | 19,927 | `AE0CFF94B3953ACA418410865F1597A784D1FE8DB93748039DBBF78EFFBB76DD` |

The two existing alignment test files changed only for focus/geometry affected
by the added row: traversal bound 6→9; optional capture route 4→7; rapid-pointer
fixture uses the same cell's lower-quarter point instead of the top quarter.
Its actual button-overlap, no source/history mutation from click continuation,
and successful deliberate later click assertions all remain and pass.

## Focused verification and preserved failures

All Go ran through `native/scripts/go.ps1` from the repository root. Commands,
stdout, stderr and exit records are preserved in
[the focused evidence directory](markdown-table-columns-ui-evidence-20261110).

1. [01 command/result](markdown-table-columns-ui-evidence-20261110/01-focused-result.json)
   selected `^(TestTableColumns|TestMarkdownColumns)` in view and app, `-count=1 -v`.
   App PASS 2.052s; view 2.209s failed only the new one/eight-column boundary
   fixture, which had clicked disabled buttons and blurred the draft before
   expecting its local F6. All other new tests passed. Original exit 1 and
   [output](markdown-table-columns-ui-evidence-20261110/01-focused.stdout.log) retained.
2. [02 command/result](markdown-table-columns-ui-evidence-20261110/02-corrected-focused-result.json)
   ran corrected boundary/geometry/rapid tests, strengthened app persistence,
   and relevant existing narrow/rapid controls. Boundary 1/8, 265/300/760-DIP
   geometry, new column continuation, old row continuation and old narrow
   alignment recovery passed. Strengthened app PASS 2.131s. View 1.750s failed
   only the old alignment coordinate overlap (old Y253.5, new button Y265.0).
   Original exit 1 and [output](markdown-table-columns-ui-evidence-20261110/02-corrected-focused.stdout.log) retained.
3. [03 command/result](markdown-table-columns-ui-evidence-20261110/03-alignment-route-result.json)
   selected only affected alignment native pointer/keyboard/undo, rapid/held,
   and real-app alignment routes after the coordinate adaptation. View PASS
   2.344s, app PASS 1.975s, exit 0; [output](markdown-table-columns-ui-evidence-20261110/03-alignment-route.stdout.log).

All three stderr files are empty. Runs 01/02 together cover every new view/app
test after its correction; 03 confirms preserved alignment behavior. No new
runtime correction was required after the initial implementation. The final
combined full suites and vet have not run and await root activation.

New view coverage includes native pointer/Enter/Space and repeat, enabled focus
order, live composition/fixed pointer geometry, clean one-undo/changed-cell
two-undo, original-coordinate selection mapping, invalid/conflicting drafts and
recovery, odd-backslash context refusal after successful cell Flush, stale
source/selection, readonly/foreign group, scroll/cancel/release, held key/new
panel, rapid pointer/new panel, callback replacements at both Changed stages
and Report, malformed original ranges/UTF-8, and rollback preserving prior redo.

The single app route uses the registered Markdown adapter with BOM+CRLF,
neighbor prose and another table. It exercises normal mode-switch cell Flush,
column insertion and deletion, deletion undo/redo, unchanged original disk before
Save, dirty state, then saves the restored three-column candidate. Reopen checks
the third header and empty middle header/body cells as well as exact bytes and
encoding/BOM metadata.

## Visual check and limits

One optional actual Tester image was captured during 02 and viewed:
[column-panel-300x480.png](markdown-table-columns-ui-evidence-20261110/column-panel-300x480.png),
SHA256 `C9C7ACEA906A4869770B1DD3D6FEB81FE09ECCA18CD5F7780DA846540E7FE931`.
The cell field, completion actions, hint, row/column/alignment rows are visible
and unclipped. Public accessibility geometry assertions also pass at 265 and
760 DIP. No screenshot was regenerated.

Synthetic native Tester events and method-level composition checks do not claim
physical OS keyboard/IME behavior. No parser/core/app runtime/vendor edits,
package/build/host/resource run, extra capture harness or child delegation.
Format implementation and its final evidence remain the separate owner's scope.
