# Open-document picker implementation frozen

2026-11-10 Astra. Assigned implementation complete; five app source files frozen
and implementation ownership released pending independent review. No package,
product host, worker, resource run, full app suite or vet was performed here.
The accepted .29 package/evidence and other owners' files remain untouched.

Ctrl+P and the fixed `已打开文档` button open a compact native dialog outside the
horizontal tab scroll. It searches bounded Unicode title/path metadata by
case-insensitive substring, preserves tab order/IDs, shows current/dirty/readonly/
binary state and duplicate-name IDs, and supports query Up/Down, Enter, Escape
and single-click selection. Repeated Ctrl+P is registered inside the modal scope.
No tab creation, disk reads, indexing, background work or persisted picker history.

Opening and activation use existing composition/Flush barriers. Actions wait
until the next view pass so list layout cannot retain released row storage.
Post-Flush identity validation covers origin, generation and destination slot/ID;
same-tab activation flushes/checks/closes without Select or hidden-source focus.
Actual source, utility and query preedit tests cover commit/cancel, failed first
and ongoing source preedit, blocked pointer/backdrop/cancel/navigation and
Tab/ShiftTab traversal. Next-frame editor focus and real Markdown field opener
restoration are asserted. Actual same-target remove/replace/move during Flush
refuses visibly and preserves bytes/history/releases; unrelated-target accepted
cases remain. Stale async selection expires the picker.

Resources remain bounded to100rows, title256bytes/path4096bytes and query128UTF16
units. Labels are cloned/sanitized on UTF8 boundaries; unchanged bounded metadata
and folded strings are reused. One bounded prefix scan while visible is an
accepted CPU tradeoff; the unchanged helper measured zero allocations, not a
whole-app memory or CPU benchmark. Search covers only retained prefixes, with
the limitation shown. The virtual list uses public stable IDs/ScrollIntoView.
Exactly two opaque public handles survive filter resets and close/reopen;
close/Shutdown cancel pending focus and clear all metadata/actions/callbacks/list
internals. Twelve reopenings prove identity reuse and opener restoration without
private toolkit introspection.

Focused evidence is in [open-document-picker-evidence-20261110](open-document-picker-evidence-20261110/).
Each attempt retains exact scripts, stdout/stderr and Go/process result files:

| Attempt | Actual result | Observation |
|---|---|---|
|01-modal|Go1; shell exit unavailable|AutoFocus preceded opener capture; native preedit absent from Access.Value; duplicate accessible label broke click targeting. Original failures preserved.|
|02-modal|Go1/shell1|Opener/click fixes passed; raw Enter still committed query preedit. Consume unsafe forwarded navigation without action.|
|03-modal|Go0/shell0, app2.102s|Modal keyboard/opener, query commit/cancel and single-click pass.|
|04-focused|Go1/shell1|Metadata/100tabs/barriers/reentry pass. Rich fixtures assumed custom Access.Value and treated legitimate clean-draft reconciliation as a failed Flush; both assumptions corrected.|
|05-rich|Go0/shell0, app2.129s|Real rich growth refusal at opening, shortened retry, real mounted field focus and injected later Flush refusal pass.|
|06-final-seams|Go0/shell0, app1.931s|Tab/ShiftTab preedit, metadata/100rows and original760x480capture pass. Root later observed poor selected secondary contrast.|
|07-focused-union|Go0/shell0, app3.318s|All12behavior top-level tests pass, including final two-handle and actual same-target regressions. Optional capture skips.|
|08-contrast-capture|Go0/shell0, app1.655s|One authorized replacement760x480capture after only selected-secondary AccentText correction. Viewed: selected path/badges legible, controls and duplicate rows fit.|

07 command, from native: `scripts/go.ps1 -GoArguments @('test','./app','-run','^TestOpenDocuments','-count=1','-v')`.
08 sets LEVNOTE_OPEN_DOCUMENTS_CAPTURE_DIR to the fresh capture-contrast folder
and runs the same wrapper with `@('test','./app','-run','^TestOpenDocumentsMinimumWindowCapture$','-count=1','-v')`.
07 ran2026-11-10T12:11:43.1813438Z–12:11:49.8790258Z;08 ran
12:13:19.1585794Z–12:13:23.7816726Z. Immediate Go exits and actual process exits
are saved. Both final runner processes exited; stderr is empty.

07 pre/post manifests match5/5 and remain saved separately. Item5 then changed
only selected row path/badge colors. Reversing that exact block and the two uses
reconstructs07 runtime hash7CB014CDF3365AE0AD62CAF5F2D11E5EC38824B48E1D5C8D2C760C1B77930339;
the other four files still match07. No08 pre-run manifest is claimed. Final
[source-frozen.json](open-document-picker-evidence-20261110/source-frozen.json)
SHA256 is2B8361918331AE37192567AAF62F90679EE25575C677F47BDBD75AB3EAADCFAD.
[freeze-verification.json](open-document-picker-evidence-20261110/freeze-verification.json)
records the exact comparison and reconstruction proof.

| Native-relative source | Bytes | Final SHA256 |
|---|---:|---|
|app/open_documents.go|14010|3E0E332285E6B3B4C0A4B23FC56B8CF298A7366333C374DE022B6520DC079390|
|app/open_documents_test.go|25662|D6B88B91A7E7A79A72C49AC2CA6D804E5313FF4FE4A58325660E2782E327DFB2|
|app/app.go|39003|FAD0EDE4C480E0C8F5C56F4E81CF2499BEF0007BA6EAE877FC516623BFFEA050|
|app/view.go|23425|64DCDC194510C6426112A7B6AAFE57A84DDE68E630A20D53C994D77CC34102C0|
|app/workspace.go|6739|7A9ACA4F8E807D7A0563B5595976839AA7DED706EFDC5171B259079D68A16DA4|

Original [06 capture](open-document-picker-evidence-20261110/capture/open-documents-760x480.png)
SHA25678F6F56AEBB088FC69A1F0AD5B3399B5D0514C63CC492D2C0D40DB0F8784CB38;
replacement [08 capture](open-document-picker-evidence-20261110/capture-contrast/open-documents-760x480.png)
SHA2569C544622D078AEFD50AB295D3FC646B2E46AAB3315A07E2144E7A02712088DE0.
The closed [evidence-files.json](open-document-picker-evidence-20261110/evidence-files.json)
records and verifies every evidence file except the inventory itself.

Limits: public MyGo Tester synthetic input proves these control/focus routes,
not physical Windows IME behavior. Opening rich refusal is real growth admission;
the later activation refusal deliberately injects a Flusher error around the
real mounted rich view, with native input verifying restoration. No claim of
natural rich mutation producing that later error. Full affected app test/vet is
reserved for root activation after exact-source independent review.
