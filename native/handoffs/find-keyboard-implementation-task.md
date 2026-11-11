# Next finite slice: native Find keyboard flow

2026-11-09. STAGED, not active until root accepts .13 delivery and explicitly
activates this task. Owner DeepSeek read_native_handoffs_release10_ui_feasibility_task_md.
No child agents. Read find-keyboard-review.md, accepted search/layout source and
the actual current MyGo focus/shortcut/input contract. Root plans/reviews only.

Problem confirmed by source review: Ctrl+F only shows the panel, Enter and
Shift+Enter don't navigate, Escape closes without source focus restoration and
the window shortcut has no Find-field composition guard. Reproduce meaningful
gaps through the real App in MyGo Tester before implementation, retaining logs.

Implement one small Find keyboard workflow using existing native TextInput,
search navigation, replace and Flush methods. Allowed files: app state fields,
Find panel/shortcut helpers, narrow Ctrl+G cross-panel guard if necessary, and
new app/find_keyboard_test.go. Preserve accepted search cache/matching/replace
authority, document/view lifecycles, two-row Wrap layout and .13 release/evidence.
No framework/dependency/new shortcut/mouse interaction redesign or core changes.

Expected behavior:

- Ctrl+F opens and focuses query. Repeated Ctrl+F focuses existing query without
  clearing query/replacement/options. Toolbar open uses the same focus request;
  toolbar toggle/close remains reachable. No source buffer edit on opening.
- Never steal focus from composing source/rich view or query/replacement (or
  the other small navigation input). Keep current values/drafts and provide a
  concise visible reason on an explicit blocked action. Use existing neutral
  Composer; do not inspect concrete format views/toolkit widget internals.
- Query Enter finds next, Shift+Enter previous. Use current key-event text, not
  stale last-frame binding. After an accepted query action keep query focus for
  repeated navigation; do not add automatic focus on every rendered frame.
- Replacement Enter performs replace-current using existing fresh match/hash
  validation and existing next-result behavior. Shift+Enter does not perform
  replacement. Never make Enter replace-all. Readonly remains unchanged.
- Normal Escape closes Find, clears its search cache and returns focus to the
  current live source editor in source/split mode. Preview should remain preview
  without creating an invisible source focus. Query/replacement/options persist.
- Enter/Escape while a Find field is composing must not search, replace, close,
  discard draft or force IME commitment. Let the toolkit/input method own its
  keys. Remove the relevant window-shortcut preemption; normal no-Find Escape
  and Ctrl+G behavior must remain working. Button/toolbar-close paths must not
  silently destroy active query/replacement composition either.

Focus/input identity:

Use stable Context keys before TextInput construction. An explicit one-shot
focus request is fine; AutoFocus only operates on appearance. Do not retain ui
Elements or TextKeys between frames. Avoid gratuitous remount on every Ctrl+F:
preserve text/selection/history when existing input can be focused safely.
Use small generation/identity guards for callbacks from a closed/reopened panel
or prior active tab/buffer. Find normally acts on the current document and can
stay open across tab switches; don't turn it into a permanently captured old
document. Existing failed Flush must preserve source/rich draft, selection and
view; identity-changing Flush must not refocus an abandoned field/tab.

Acceptance:

1. Actual App tester Ctrl+F from source, repeat from replacement, no per-frame
   focus stealing; preserved text/options; query typing routes to query only.
2. Enter/Shift+Enter adjacency/wrap and repeated key navigation; replacement
   Enter single transaction/freshness/readonly; Shift+Enter no replacement.
3. Query and replacement Compose+Enter/Escape retain panel, value and source;
   commit then normal action works; source/rich composition blocks opening;
   normal Escape/close focus behavior and next source typing works.
4. Failed/mutating/identity-changing Flush and stale close/reopen/tab/buffer
   callbacks do not navigate/replace the wrong target or lose drafts. Reuse
   existing substantive fixtures where useful, don't mirror implementation.
5. Existing min-window reflow tests and actual Ctrl+G tests remain green.
   No need new screenshots unless visual markup actually changes.

Run focused keyboard+Find+CtrlG/layout tests during development, then one final
affected app suite and vet after fixes. Preserve actual commands/exit/output and
failures under find-keyboard-evidence. Do not rebuild packages or rerun unrelated
workers/suites/benchmarks. Publish find-keyboard-result.md with actual behavior,
scope, exact final files/hashes, checks and remaining limits, then freeze for root
and a targeted independent review. Tester composition is not physical OSIME.

Root will activate a fresh .14 delivery only after actual acceptance. No product
docs in this task; current accepted .13 docs remain accurate until packaging.

## Root interim review after initial red (21:18)

Initial-keyboard-red.log reproduces real gaps. Please send a short progress
checkpoint or write find-keyboard-checkpoint.md after a significant constraint.
Keep actual failed attempts (do not reuse/overwrite a filename called pass).

The initial implementation inspected by root still has these provisional issues:

- c.Shortcut registers/preempts a key even when a subsequent inner FocusWithin
  guard returns. Guard Escape registration before the call so the text field
  can own Escape. Actual toolkit route order is in ui/input.go and textinput.go.
- a.find invokes source Reveal which requests source Handle focus. A later query
  Handle.Focus request can compete with that request by handle registration
  order (ui/ref.go). Extra tester.Frame calls aren't a behavioral fix. Resolve
  one-shot focus ownership through public APIs, without editing vendor or
  exposing underlying editor handle. If a minimal neutral editor API is needed,
  report concrete reason before widening scope.
- Unconditional query focus after a.find must not refocus an abandoned panel
  after a Flush closes/reopens/switches tab or replaces the buffer. Check panel
  generation and exact final target identity. closeFind must cancel pending
  field focus requests so hidden controls cannot regain focus later.
- Existing replace-current then find-next selects the remaining match. Escape
  followed by typing replaces that selected match (fixture text should become
  'match !'), not insertion before it. For source-click typing tests, assert
  against captured selection or explicitly place caret, not assumed suffix.

These are review findings to route implementation, not an authorization for
general input refactoring or unbounded retries.

### Narrow scope allowance: cancel pending source focus

Root read actual ui/ref.go applyFocusRequests: it walks stored handles in
registration order, so merely issuing query Handle.Focus later (even at end of
Find draw) doesn't order it after source Reveal's pending handle. If current
app-only solution cannot remove this conflict, owner is authorized to add a
tiny neutral Editor.CancelFocus() forwarding to its existing handle.CancelFocus
in ui/editor/editor.go, documented as cancelling pending requests only. No
vendor edit or broader editor behavior change. After validating the exact
target/panel identity, a keyboard navigation action may cancel its own Reveal
source focus and make the one-shot field focus request. Never call ReleaseView
just to cancel focus, since it mutates composition/layout state. Add minimal
meaningful focused coverage through the App; include editor in final affected
suite/vet if that editor method is changed. This allowance is optional, not a
requirement to redesign working focus behavior.

### Replacement and callback identity review

Existing replace() captured t before Flush but, unlike find(), didn't recheck
Current/Document/Editor/Buffer after Flush before modifying that buffer. If an
actual identity-changing Flush regression demonstrates this, narrowly add the
same exact identity guard to replace() before any mutation. This safety guard
is allowed in this task; keep matching/transaction semantics otherwise intact.
Pre-frame input callbacks should reject their old target if tab/buffer changed
before a new frame, then the next frame can bind Find to the current document.
Checking only panel generation is not enough when Find stays open across tabs.

### Composition regression review (21:29)

broad-regressions-focused-2.log shows duplicated committed query/replacement
after synthetic composing Enter. This must not be dismissed as expected without
tracing toolkit semantics: returning false from OnTextKey lets Enter fall through
to the single-line control (ui/input.go editor.wants(KeyEnter)); that can commit
its marked text even though app composition flag remains true. App must not
search/replace/close or force an early commit. In the raw-key path reaching our
handler, consuming Find Enter/Escape while tracked composition is active is
acceptable to suppress the control's default action; real OS candidate handling
may intercept before that path, still unverified. Verify marked/visible text as
well as the app flag before/after, then normal explicit commit exactly once.

Fresh replace fixture needs an exact selected match before Enter; caret(0,0)
alone doesn't authorize replace-current. Select that match before testing
replacement, don't change existing replace semantics to satisfy wrong fixture.
After Select(new tab), editor focus naturally changes; test rebound query input
by explicit Ctrl+F/click after a new frame instead of sending Enter into source.
Retain a distinct before-frame stale callback test.
