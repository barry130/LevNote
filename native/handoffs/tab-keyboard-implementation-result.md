# Tab keyboard + active reveal result

Corrected 2026-11-10 after root review, Astra. Source/tests frozen for root
acceptance. This is a combined implementation stage, not a new runnable
package; .21 stays current.

`app/app.go:378` adds guarded adjacent selection; `app/view.go:548` registers
exact Ctrl+Tab / Ctrl+Shift+Tab. Tabs wrap in visual order through unchanged
Select. Zero/one tabs return before panels, focus, composition or persistence.
Source and real Markdown composition still commit through Select/Flush; rich
Flush refusal keeps the active document/view, draft, undo and export job.

The original synthetic Find preedit-loss failure is retained in logs 03–05.
The approved [panel decision](tab-workflow-panel-composition-decision.md)
adds a shortcut-only refusal while a visible Find query/replacement or
GoToLine field composes. It keeps focus/preedit and reports
“输入法正在组合文本；请先完成组合后再切换标签”. Commit/cancel, then explicitly retry;
no switch is queued. Hidden fields do not block. No vendor or Select refactor.

Root review identified a stale GoToLine flag after pointer focus leave. The
new two-direction regression first reproduced it: committed query "12" with
`GoToLineComposing=true` (15 log, exit 1). `app/source_navigation.go:291` now
reconciles the flag using its frame-local field's public Focused/Composing
state, following Find. The flag clears only after actual composition/focus
have ended. Pointer leave preserves committed value plus preedit, does not
switch automatically, and the explicit next shortcut succeeds with unchanged
source/undo state. Focused preedit and existing keyboard commit/cancel tests
remain protected and passing; no string reconstruction or retained widget.

One focused file, `app/tab_workflow_test.go`, holds eight keyboard tests and
five reveal tests, plus an opt-in minimum-window capture. Cases include exact
modifier routing, zero/one no-op, both directions/wrap, failed Flush, success
release/export/cache/persistence/focus, source and real rich composition,
query/replacement/GoToLine refusal with export retention, commit/cancel retry,
hidden flags, and the geometry/lifecycle cases in the
[reveal result](tab-active-reveal-result.md).

Final validation, run once after the root-requested correction via the
required wrapper (root cwd):

- `native/scripts/go.ps1 -GoArguments @('test', './app', '-count=1')`
  — `ok levnote.local/native/app 17.278s`, exit 0.
- `native/scripts/go.ps1 -GoArguments @('vet', './app')` — exit 0.
- Actual commands/output/exits: [18 test](../artifacts/tab-workflow-20261110/18-corrected-final-app-test.log),
  [19 vet](../artifacts/tab-workflow-20261110/19-corrected-final-app-vet.log).
  Correction/guard/commit/cancel focused success:
  [16 focused](../artifacts/tab-workflow-20261110/16-goto-pointer-correction-focused.log).
  Original stage test/vet success (13/14) and earlier focused logs remain.

Other retained diagnostics are explicit: 02 relative Tee path failed while
the wrapper changed cwd (no trustworthy Go output/exit); 06 wrongly assumed
Tester.Texts includes input values; 07 showed GoToLine's composition flag
persists on pointer-only focus leave. The original test retained its explicit
Tester.Type commit; root required the separate pointer regression and
production correction above (15 red / 16 green), rather than accepting that
test exception as a resolution;
11 captured caret geometry before inserting the export-progress row, corrected
in 12. No production reentry issue was found. Physical OS/IME behavior remains
unverified; no package/build/host/resource run or README edit occurred.

Final product/test identities (bytes / SHA256):

| File | Bytes | SHA256 |
| --- | ---: | --- |
| app/app.go | 34047 | 0148CFA928154E440C22C3D3C9AE08CC5B34A791F1732C615F1932B3021BC1D3 |
| app/view.go | 22464 | BE5B492D1DA34AF246F447F5B61989BC90DD7ABFC4281DA6FBAC7FE4BCB62909 |
| app/source_navigation.go | 8453 | 76998D57326DB1ED75BC27C2E7C34BEB68DD204E54E16316002F0A928C8CB2F5 |
| app/tab_workflow_test.go | 22725 | 83F835E597A23238FE45408BDA55262995F77252ADB80122E971460EB8722A2C |

Machine-readable identities:
[final-file-identities.json](../artifacts/tab-workflow-20261110/final-file-identities.json).
