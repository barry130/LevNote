# Text and closed-tab retention fixes accepted

2026-11-09. Root read both changed production sections, both regression sources,
actual red/green transcripts and final combined dependency results. The minimal
fixes are accepted for delivery under retention-release11-delivery-task.md.

- core/text clears coalesced slice entries beyond logical length, removing stale
  chunk references while preserving copied snapshots and undo/redo history.
  Final regression distinguishes stale-only references from duplicate references
  to live chunks and checks snapshots before/after undo/redo/history reset.
- app.removeTab shifts remaining pointers, clears the obsolete final slot and
  truncates. Public CloseTab regression verifies last/middle/all removal and a
  canceled close preserving tabs, active selection, content and session bytes.
- No speculative copy was added to every replacement. The audit found no current
  production caller supplying a small substring backed by a large source to
  ReplaceUTF16; the public-API possibility remains documented, not dismissed.

Focused red/green, full core/text (0.635s), vet core/text and app,11existing close/
recovery/lifecycle cases and one combined dependency pass succeeded. Final combined:
core/document0.713s, ui/editor2.395s, app15.388s, exit0. Evidence is preserved in
text-retention-result.md and tab-retention-fix-result.md; no duplicate suite.

Frozen final SHA256:
- core/text/buffer.go 562F1293800CF147FE466F1B64E29FA5627591F1E8CCC698FEAE9163534E2E9A
- core/text/retention_test.go 8AE4894024E97B135FD49D32774AFAD7EDCA4CE01A87E7CF62101AD67F62A5A2
- app/app.go 6387CDF9938D3546C26CD8D387CA4B334A011A2A47456CBCB300E0AA5890527A
- app/tab_retention_test.go B0CA0D1E70A2C58F3F81EA943081550F6B0D3DF368A25B702E0562BCE241C316

This acceptance is about reference ownership, not immediate garbage collection,
RSS, working set, global leak absence or improvement against legacy Tauri. The
Computer Use helper is unavailable; real-window measurements remain deferred.
Accepted .10 stays current until the fresh .11 binary is actually delivered.
