# Picker raw query key correction frozen

2026-11-10 Astra. The independent review blocker is corrected in only
app/open_documents.go and app/open_documents_test.go. Other three picker inputs
remain byte-identical to the original freeze. Ownership released for the narrow
independent delta review; no further checks until root activation.

Previously OnTextKey returned false for Left/Right and other non-text editing
keys during query preedit. Public MyGo keyDownPhysical then queued editKey,
which calls commitCompose before caret/edit behavior without an InputText
observation clearing the app flag. This was a concrete source/API finding; no
pre-correction runtime reproduction was run or claimed in this correction.

During active preedit, openDocumentsKey now consumes every non-text key
delivered by OnTextKey. The public callback contract restricts delivery to these
keys. Native Compose/Text still explicitly cancel/commit; normal non-composing
editing and picker navigation use the unchanged branch. No exceptions, vendor
changes, menu-command interception or wider composition policy were introduced.

The actual public native-query regression now sends32key/modifier combinations
through each commit/cancel case: Left/Right, Shift and Ctrl selection/word moves,
Home/End, Up/Down/Page keys, Backspace/Delete, Ctrl deletion, Ctrl A/C/X/V/Z/Y,
Shift redo/Enter, Enter/Escape/Tab/ShiftTab. After each, it verifies native
Composing and IME caret, query focus/value/filter/selection/pending actions,
active document identity, both documents' byte snapshots/version/history/
selection and clipboard. Explicit Text commit then Enter selects the intended
tab; explicit empty Compose then Escape restores the original editor.

A separate real query test verifies normal Left/Right insertion, Shift selection,
Home/Delete, End/Backspace, select-all, copy/cut/paste and undo/redo, then Enter
activates the filtered destination. This covers the unchanged non-composing
path through actual public input events, not direct invocation of the callback.

One corrected focused union ran through the required wrapper from native:
`scripts/go.ps1 -GoArguments @('test','./app','-run','^TestOpenDocuments','-count=1','-v')`.
The saved wrapper explicitly removes the opt-in capture environment variable.

- Go exit0 and shell exit0; app2.619s,13top-level behavior tests PASS and the
  optional capture SKIP. Stderr is0bytes. No failed correction attempt.
- Started2026-11-10T12:29:07.8251921Z, Go finished12:29:13.5343644Z; owned
  hidden pwsh runner PID40456 exited0 at12:29:13.7385452Z.
- Pre/post5source identities match. Raw streams, command script and both exits
  are preserved in [correction evidence](open-document-picker-key-correction-evidence-20261110/).

The new [source-frozen.json](open-document-picker-key-correction-evidence-20261110/source-frozen.json)
SHA256 is AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6.

| Native-relative source | Bytes | Final SHA256 |
|---|---:|---|
|app/open_documents.go|13882|BF1789ACC0A9B885F358CD46F62251C5E2971A6DD351AC5209A3C82BC15B7DA6|
|app/open_documents_test.go|30122|A70444047DED384477F2DFDE2E83FE98AC53F06304633F4F65304A19CCC5444C|
|app/app.go|39003|FAD0EDE4C480E0C8F5C56F4E81CF2499BEF0007BA6EAE877FC516623BFFEA050|
|app/view.go|23425|64DCDC194510C6426112A7B6AAFE57A84DDE68E630A20D53C994D77CC34102C0|
|app/workspace.go|6739|7A9ACA4F8E807D7A0563B5595976839AA7DED706EFDC5171B259079D68A16DA4|

Original open-document-picker-evidence-20261110 remains closed and unchanged:
all47files/123554bytes verified before/after, plus original inventory hash
338A86C313FFEB268C71E57F9DFF5C92D8A3F593E8F7E39FBA5926B9F421764E. Both
earlier captures, initial failures and original manifests remain there. The
new [evidence-files.json](open-document-picker-key-correction-evidence-20261110/evidence-files.json)
closes the correction evidence separately and excludes only itself.

Limits: synthetic public Tester keyboard/Compose/Text transport, not physical
Windows IME evidence. Raw key shortcuts here are distinct from external native
menu commands; those are not intercepted or claimed. No capture repeat, full
app suite, vet, build, package, host or worker/resource run. Final app test/vet
remains held for root activation after independent review. The read-only next
three-list review is saved separately; no list runtime implementation began.
Context is well below200K and progress/result are on disk; no compaction needed.
