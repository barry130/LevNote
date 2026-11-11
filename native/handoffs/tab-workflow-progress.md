# Tab workflow progress

2026-11-10, active Astra owner. Keyboard focused validation checkpoint saved.

- Added the two exact Ctrl+Tab / Ctrl+Shift+Tab bindings and a guarded
  visual-order helper in app.go/view.go; switching delegates to existing Select.
- Focused keyboard tests passed (six top-level tests) via wrapper:
  `go.ps1 -GoArguments @('test', './app', '-run', '^TestTabKeyboard', '-count=1', '-v')`.
  Actual output and exit 0: `artifacts/tab-workflow-20261110/08-keyboard-focused.log`.
- Original synthetic Find preedit failure and diagnostic/retry logs are kept.
  Root-approved `tab-workflow-panel-composition-decision.md` adds a bounded
  visible-panel guard; source/rich-editor switching still delegates to Select.
  Query/replacement/GoToLine refusal, commit/retry, cancellation/retry and hidden
  stale flags are covered. Tester text enumeration omits input contents; tests
  use public focus/caret observation and commit surviving preedit instead.
- Active-tab reveal is implemented: stable integer ID, frame-local button and
  default public ScrollIntoView. Close-button mutations run after tab iteration
  finishes, avoiding the live backing-slice tail being nil-cleared mid-loop.
- Five reveal tests passed in `09-reveal-focused.log`: 14 tabs at 760x480,
  wrap, manual scroll, same-tab focus, add/open/restore, both click directions,
  active/inactive close neighbors, failed Flush, empty and reopen.
- Additional rich composition, panel export retention and inactive-close ID
  cases pass (11/12 logs). One screenshot `tab-active-wrap-760x480.png` inspected:
  active tab-13 button is fully visible, ordinary source layout intact.
- Final combined `test ./app -count=1` passed in 16.738s; `vet ./app` passed,
  both exit 0 (13/14 logs). One final full suite only.
- Final identities saved; `tab-keyboard-implementation-result.md` and
  `tab-active-reveal-result.md` published. Source/tests frozen for root review.
  No README/package/launch changes.

Root-review correction checkpoint2026-11-10:

- A real GoToLine pointer-leave regression reproduces both shortcut directions
  failing: committed query is "12", while `GoToLineComposing` remains true.
  Original output/exit 1 retained in `15-goto-pointer-regression-before.log`.
- `source_navigation.go` now reconciles that flag using a frame-local input's
  public `Focused()`/`Composing()` state after drawing, matching Find's rule.
  Focused preedit remains protected. Focused correction/guard/commit/cancel
  cases all passed in `16-goto-pointer-correction-focused.log`, exit 0.
  Both pointer-leave directions preserve query "12", document/undo state,
  and wait for a later explicit shortcut.
- Corrected final `test ./app -count=1` passed in 17.278s (18 log) and vet
  passed (19 log), both exit 0. Source-navigation and test identities updated;
  result handoffs corrected. Source/tests frozen again for root acceptance.
- No package/host work or product-doc edit occurred.
