# Correct raw keys during picker query composition

ACTIVE2026-11-10. Astra owns only app/open_documents.go and its focused tests.
This concrete blocker takes precedence over next-list planning; save any read-only
checkpoint, then correct picker. Other three picker files remain frozen. No
children, vendor edits, package, host or full app suite/vet yet.

Read open-document-picker-final-review.md. Root confirmed public MyGo route:
OnTextKey returns false for Left/Right -> editor queue editKey -> commitCompose
before caret move, without InputText observation clearing app composing flag.
Existing07 omitted those arrows. Source/API finding has not yet been reproduced
by reviewer; direct public Tester regression is needed.

Do not patch only one arrow and leave the same route via other raw non-text
keys. Inspect editor_navigation.go:wants and textinput.go OnTextKey contract.
During active query preedit, the app has no intentional raw non-text editing
action. Prefer consuming every key delivered to this OnTextKey callback while
composing; native Compose/Text updates remain the explicit commit/cancel path.
If a specific exception is necessary, explain it and prove it cannot internally
commit preedit or leave a stale guard. Outside composition preserve ordinary
caret/selection/editing shortcuts and existing picker navigation.

Extend public native-input tests for Left/Right, Shift selection, Home/End and
representative delete/edit-command keys using same real query. Check native
Composing, caret, app composing/query, picker focus/open, active document bytes/
version/history unchanged. Explicit Compose cancellation and Text commit/retry
must still work. Keep tests bounded; no private toolkit fields or physical IME
claim. Normal non-composing Left/Right and editing should still operate.

Run one corrected focused picker union with pre/post identities and saved raw
streams/Go+shell exits. A concrete failure allows scoped repair and rerun with
all attempts preserved. No capture repeat: layout remains unchanged.

Existing open-document-picker-evidence-20261110 is closed47file evidence and
must remain unchanged, including old source-frozen.json. Use fresh
open-document-picker-key-correction-evidence-20261110 with corrected source
manifest covering all5picker inputs. Publish
open-document-picker-key-correction-result.md including original blocker,
actual final result, new hashes and remaining evidence limits. Freeze/release;
independent reviewer then checks only this delta before final app test/vet.
