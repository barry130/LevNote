# Table-column UI checkpoint

2026-11-10. Implementation milestone reached; focused checks are next, not yet
claimed passing. .24 package, portable README and closed evidence remain frozen.

New `modules/markdownview/table_columns.go` adds short wrapped before/after/delete
controls with accessible labels `在此列前插入列`, `在此列后插入列`, `删除此列`.
They use captured total-column counts for one/eight-column disabled states and
the existing draft/gesture path. Row actions remain the first F6 target.

`table_actions.go` now accepts an ascending list of original byte edits. It
validates ranges, UTF-8 boundaries/replacement strings and nonoverlap, converts
all offsets before mutation, then applies descending in one owned buffer group.
Both selection endpoints use the existing insertion affinity in the same order;
selection is set once before EndGroup, Changed once afterward. CancelGroup
restores bytes/selection/history but can advance revision. Row/alignment only
adapt their single patch to the shared representation.

New focused view tests cover native pointer/Enter/Space, repeat, narrow
265/300/760-DIP geometry and complete forward/reverse enabled focus order,
one/eight-column/readonly state, live composition and fixed button geometry,
one/two undo transactions, selection across edits, malformed original ranges,
owned rollback preserving redo, invalid/conflicting drafts and recovery,
post-Flush context refusal, held gesture/scroll/cancel/release and callback
replacement. The app test uses the real registered adapter with BOM+CRLF,
normal mode-switch draft Flush, insert/delete, unchanged original disk before
Save, exact bytes/dirty state and reopen.

Ownership: new table_columns.go/table_columns_test.go and
app/markdown_columns_app_test.go; narrow table_actions.go/table_rows.go/
table_alignment.go edits. Per root extension, existing alignment view/app
keyboard helpers changed their bounded traversal 6→9 and optional capture's
exact route 4→7; their focus/behavior assertions remain intact.

No parser, core, app runtime, vendor or format-owner files changed. The shared
format API now exists; its owner is still correcting focused format cases.
Only focused new view/app checks will run now through `scripts/go.ps1` with
saved commands/stdout/stderr/exits. Combined final test/vet awaits root activation.

Freeze update: focused runs completed with all new behaviors now passing across
preserved runs 01/02; corrected existing alignment route passes in 03. Initial
boundary fixture blurred the cell before F6; assertions were reordered to test
F6 from the actual draft, with disabled pointer checks retained afterward. The
additional column row also moved an old rapid-click alignment fixture out of
overlap: use the same cell's lower-quarter coordinate while retaining the real
overlap, refused continuation and later deliberate-click assertions. App Save
now preserves the inserted three-column candidate after deletion undo/redo, so
reopen proves structural persistence. One 300×480 panel image was viewed and
is complete. Eight owned files frozen in 04-source-freeze.json. See the final
UI result; no combined suite/vet or product execution performed.
