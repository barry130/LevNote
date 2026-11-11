# Picker WIP review points

2026-11-10 root static read of first open_documents.go. Not a final rejection;
implementation/tests are in progress. Owner resolves these within existing scope.

1. refreshOpenDocuments currently rebuilds/clones/sanitizes/folds every title
   and path each call, and the dialog calls it twice per frame plus navigation.
   At100long-path tabs this can allocate large transient metadata repeatedly
   while only focus/caret changes. Bound total work AND avoid unchanged-frame
   rebuilding: compare bounded current source metadata with cached owned rows,
   reuse unchanged clones/folded values, separate metadata refresh from filtering
   if needed. Do not retain full unbounded original labels or Tab pointers just
   for cache identity. Meaningful bounded regression should show stable unchanged
   frames/filter-only work do not regenerate all copied metadata, without claiming
   whole-process memory. Keep filtering fresh for typed-before-Enter ordering.
2. Existing selectChecked same-slot success calls hidden source Editor.Focus in
   preview. Exact next-frame focus should be proved; same-document picker choice
   may complete its explicit Flush/identity barrier then close without invoking
   unrelated focus changes. Cross-target choice must use the existing Select
   lifecycle. Avoid a generic Select rewrite just for this new UI.
3. Current input observer is on whole panel, reflecting whichever input field
   has focus. Only query is editable now; keep IME ownership/refusal checks tied
   to its actual public state, including immediate input-before-frame timing.
   Core DialogBase dismissal return alone is not proof of preserving preedit.

Independent review checkpoint's repeated Ctrl+P modal scope requirement is
already represented in this WIP by panel.Shortcut; real focus coverage pending.

Later narrow route2026-11-10: while query preedit is active, Tab/ShiftTab also
move focus through the toolkit unless explicitly consumed. Check actual public
Tester behavior, preserve preedit/focus on unsafe traversal, then allow ordinary
Tab navigation after explicit completion/cancellation. This is separate from
already passing pointer/Enter/UpDown/Escape guards; do not weaken their tests.
