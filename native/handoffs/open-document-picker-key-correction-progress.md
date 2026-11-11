# Picker raw-key correction checkpoint

2026-11-10 Astra. Only app/open_documents.go/open_documents_test.go changed;
other three picker inputs verified against the previous frozen manifest.
Original47file evidence and inventory verified intact before correction.

Read public OnTextKey contract, wants, keyDownPhysical and editKey/commitCompose.
Composing branch now returns true for every non-text key delivered to OnTextKey;
Compose/Text observations remain unchanged. No raw-key exception is necessary.
Expanded real query test covers32key/modifier routes per commit/cancel path,
checking public native Composing/IME caret, app query/filter/focus, selected tab,
both document snapshots/versions/history/selections and clipboard. New ordinary
editing test proves caret/selection, delete/backspace, copy/cut/paste, undo/redo
and final query activation outside composition.

Fresh open-document-picker-key-correction-evidence-20261110 has original-source
baseline and five pre/post identities. Corrected focused union01 PASS:
Go/shell0, app2.619s,13behavior tests and1opt-in capture skip. All5source files
unchanged across run. Both original47evidence files and inventory verified
unchanged after correction. Final manifest SHA256
AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6.
Source ownership released; no further runs pending independent delta review.
No capture, broad app suite, vet or product execution. Next-list review is saved
separately and stopped. This context is well below200K; no pressure-triggered
compaction needed, all current progress preserved on disk.
