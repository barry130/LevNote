# Next user-facing stage: native go-to-line, preparation only

2026-11-09. Owner DeepSeek read_native_handoffs_release10_ui_feasibility_task_md.
Root accepts your retention result. While the other DeepSeek builds .11, read
the current app/editor APIs and prepare a concise implementable plan only in
handoffs/source-navigation-plan.md. No production/test/docs edits or commands
that run tests/builds/launch yet: .11 build inputs are frozen. No child agents;
compact before200K. Root reviews/plans only.

Rationale: mature text editors offer direct line navigation for long source and
error locations. LevNote currently shows line/column in the status bar but has
no Ctrl+G action. Add this common useful behavior with existing MyGo primitives,
no dependency, animation, command-palette framework or whole-file scan/copy.
Do not claim external software research was performed.

Prepare a minimal native panel: Ctrl+G and discoverable current line/column
status action, input `line` or `line:column`,1-based logical lines and UTF-16
columns consistent with the editor's existing status convention. Make that
convention explicit in panel help. Empty/invalid/zero/negative/overflow/out-of-
range input should be reported locally, leaving document, selection and history
unchanged; blank column means default1 only if the grammar explicitly permits.
Choose one clear grammar and test it. No implicit clamping for a typo.

Inspect existing LineRange/OffsetAt/LineColumn/reveal/selection normalization
and MyGo text input focus/IME contracts. Account for CRLF terminal columns,
empty final line, Unicode surrogate/grapheme boundary behavior, segmented long
lines and readonly source/byte-inspection distinctions. Reject a column inside
a surrogate pair or normalize only with truthful visible guidance; don't split
UTF-16 units silently. Need no per-character scan of the entire document.

Use existing draft synchronization/mode-switch barriers. The go-to action may
switch preview to source only on successful validation/Flush; split retains
split and source focus. Explain that target is source line, including JSON/XML.
If a draft flush changes text, interpret requested coordinates on final source.
Failed Flush or invalid final coordinates must retain pending content and make
the outcome truthful (a successful flush itself may be an existing edit).
Opening/canceling the panel must not write content, drop pending composition,
or create undo history. Respect existing modal/export/package lifecycle rules.

Identify a small test set: parser/boundaries, actual App native input/keyboard
focus, readonly source, dirty rich draft+Flush refusal, switch/close/tab changes
while panel open, large/long-line reveal without whole-text conversion. One
760x480 light/dark tester example if a visual panel is added; explicitly not
physical OS IME verification. No full app/build in preparation.

Name exact proposed file scope and APIs after inspection. If current contracts
make this too large or unsafe, state the precise reason and a smaller complete
variant; don't silently start a broad navigation subsystem. Root will activate
implementation after .11 inputs are no longer in flight.
