# Find composition focus-leave correction

2026-11-09. Active for implementation owner DeepSeek
read_native_handoffs_release10_ui_feasibility_task_md. Root accepts review finding
in find-keyboard-independent-review.md as material; .14 delivery remains staged.
Unfreeze only narrow Find lifecycle source/tests. No children/build/docs/launch.

Reproduce real App/Tester composing query and replacement -> source/tab focus
transfer. Current flags clear only from observer Compose/Text; toolkit release
calls UnmarkText without these observer events. Prove current Ctrl+F/Enter/close
can become incorrectly blocked after visible composition ends.

Fix by reconciling current field composition/focus through public toolkit APIs
and narrow hidden-subtree lifecycle, preserving Query/Replacement/options and
committed preedit text. No source editor ReleaseView, no synthetic discard, no
vendor edit. FocusLeft/Focused/Composing may help; ensure timing through real
frames, don't clear a currently active marked value before toolkit release.
Observers keyed to old document must not leave stale transient flags after tab
switch; document identity still guards actions, not lifetime of independent
Find field text. Hiding via ShowPlugins must cancel stale Find focus intent and
composition flags without later focus steal or value loss. Pending refs should
not retain old tab/doc/buffer when hidden/invalid. Preserve one-shot opening.

Minimal meaningful tests: each input compose -> click source; compose -> switch
tab; compose -> plugins hide -> show; verify expected single committed value,
subsequent Ctrl+F/Enter/Escape usability, no source/history change due to cleanup,
and no unsolicited later query focus after another source is active. Existing
buttons KeepFocus and active-composition Enter/Escape guards must remain.

Record actual red/new focused logs with new filenames. Then one final affected
app/editor suite and vet after code correction (prior passing runs remain).
Update find-keyboard-result.md with correction and exact frozen hashes; retain
prior evidence. Reviewer will inspect narrow fix and reconcile final identities
without rerunning tests. Report when source ready, no unrelated scope.
