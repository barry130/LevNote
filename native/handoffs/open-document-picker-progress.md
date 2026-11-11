# Picker implementation checkpoint

2026-11-10 Astra. Active app-only source ownership: new open_documents.go and
open_documents_test.go; small app.go/view.go/workspace.go wiring. .29 unchanged.

First runtime batch on disk: bounded metadata/ID list, Ctrl+P + external tab
button, DialogBase with KeepFocus backdrop/rows/cancel, native query and virtual
List with ScrollIntoView. Repeated Ctrl+P registered inside modal panel scope.
Activation uses selectChecked plus post-flush origin/destination validation;
same-slot explicitly flushes/checks/closes without changing focus.
Clicked actions consumed before the following View pass, because List can build
more rows during layout after its initial callback returns. Clear metadata only
after prior list callbacks are no longer being constructed.

Focused attempts01/02 are preserved failures: AutoFocus preceded Overlay opener
capture (replaced by deferred public Handle.Focus); duplicate row/button labels
prevented unambiguous Tester click; Access.Value excludes native preedit; raw
Enter reached widget commit when composing (consume navigation without acting).
03 modal PASS.04 all picker checks passed except invalid rich-test assumptions:
custom text client has no Access.Value; external clean edit correctly reconciles
and drops a clean draft, so it was not a failure fixture.05 corrected real rich
opening growth-refusal plus injected later Flusher refusal with mounted real
field focus PASS.06 Tab/ShiftTab preedit + metadata +100rows +capture PASS.

Original760x480capture saved and visually read; full controls/list/hint fit.
Root final refinements: retain exactly two opaque public handles across filter
reset and close/reopen, cancel pending focus and clear every other state field.
Added12reopening public-handle identity/focus regression and actual same-target
remove/replace/move during Flush, retaining previous unrelated-target cases.
Final focused union07 PASS: Go/shell0, app3.318s;12behavior tests pass and opt-in
capture skips. All5source identities match before/after07. Root then observed
low selected-row secondary contrast in06 and authorized the narrow AccentText
correction plus one replacement capture.08 capture PASS: Go/shell0, app1.655s;
replacement visually read, selected paths/badges legible. Original06 preserved.

Final5source freeze saved. Four files are unchanged from07; reversing only the
selected-secondary color block and two color uses reconstructs the exact07
runtime SHA256. Final runtime is3E0E332285E6B3B4C0A4B23FC56B8CF298A7366333C374DE022B6520DC079390.
No further source edits or checks. Implementation result and closed evidence
inventory are ready for independent review. Full app test/vet remains unrun
and requires root activation; no product builds/hosts/workers were run.
