# Find keyboard flow: root source acceptance

2026-11-09. Root accepts the frozen bounded Find keyboard implementation and
post-review focus-leave correction. Actual .14 package acceptance is separate;
.13 remains the current runnable package until that delivery is verified.

Root read the implementation/result, the original independent review and final
reconciliation, all seven lifecycle regression cases, actual red/green/final
test/vet logs, and the public MyGo focus/composition lifecycle used by the fix.
Root independently hashed all eight source/test files; they match the final
table in find-keyboard-result.md and find-keyboard-independent-reconciliation.md.
Reconciliation SHA256:
BD109C08E4EFAA2438C714A6A1CE0F2057A6ABA1B06D0D8640B6CF86A0BB9A98.

Accepted behavior: Ctrl+F/toolbar opens or focuses the existing query once;
query Enter next and Shift+Enter previous retain query focus; replacement Enter
uses existing replace-current semantics, Shift+Enter does nothing; Escape
returns to the valid current source/split without focusing an invisible preview
editor. Values/options remain. Exact generation/tab/doc/editor/buffer guards
prevent stale actions, including replacement after a failed/identity-changing
Flush. The tiny neutral Editor.CancelFocus cancels the source reveal request,
not the document/editor lifetime.

The review's stale-composition blocker is resolved: after toolkit focus release
the field must be both unfocused and no longer composing before clearing its
flag. Plugins hiding the subtree clears transient handles/target references;
query/replacement/options and source text/history remain. Active composition
Enter/Escape/pointer protections and Ctrl+G deferral remain covered. Root found
no remaining material blocker in this bounded scope.

Actual focused lifecycle correction:7/7 cases, app1.634s, exit0.
Final affected suite: app14.326s/editor2.083s, exit0; final vet exit0/no warnings.
Raw final suite SHA256:
49F7F745B49E4D8551C48C74FB8F696EFBA6D1306BE99B59F8E1D29EBDE86B1B.
Final vet SHA256:
55136E372CC06096357CBA87F588CCC89781D15DC38B9BC7BEEF9078BF7A4185.
Earlier logs remain historical, including real reds and fixture/harness errors.

Synthetic App/Tester evidence is not physical Windows IME/candidate-window
acceptance. No new memory/CPU claim. Visual layout unchanged; accepted .13
minimum-window images remain historical captures. Root ran no test/build/launch.

Activate find-keyboard-release14-delivery-task.md now. All product source,
tests, build scripts and product docs stay frozen during packaging. Planning
handoffs may continue. Delivery owner verifies scoped inputs and .13 preservation
without repeating accepted suites or managed-worker execution. Only after root
accepts the actual .14 binary may six-doc follow-up begin.
