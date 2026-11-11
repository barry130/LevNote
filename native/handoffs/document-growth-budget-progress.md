# Document growth implementation checkpoint

2026-11-10 Astra. Runtime batch implemented before focused checks:
core/text/buffer.go,core/document/document.go,core/search/find.go,
ui/editor/editor.go. No other runtime source changed.

Text owns128MiB default, positive immutable-base NewWithLimit, fixed effective
max(base,seed bytes), Restore re-seeding, typed SizeLimitError with stable
errors.Is(ErrSizeLimit) and actionable Chinese text. ReplaceUTF16 validates UTF8,
range and surrogate before size, then copies removed content only after passing.
Undo/Redo/CancelGroup retain accepted-state replay; no new refusal/truncation.

ReplaceAll plans actual reverse order before BeginGroup, rejects any oversized
intermediate state; literals are shared and only regex outputs are retained.
Accepted regex-plan bytes fit the ceiling; source/match arrays and an individual
expansion/clipboard/codec string may already allocate and are not memory promises.
Indent preflights all positive growth by subtraction/division before unmark/group;
outdent remains available. First rejected composition opens no group; ongoing
composition keeps prior mark/state. Injected paste no longer reads native
clipboard first.

Next: tiny public-seam text/document/search/editor regressions and real Markdown
plus graph draft/app safety. Existing format Flush paths report and retain
rejected drafts on inspection; no format/app runtime correction yet identified.
Only focused test groups will run with explicit Go exit/streams in fresh
document-growth-budget-evidence-20261110. No final suite/vet/package/launch.

Root's composition correction implemented2026-11-10 before any focused run:
ui/editor/editor.go retains size-only composition error+buffer owner. First
refused preedit has no group but blocks Flush; ongoing/final refusal preserves
the accepted mark/group. UnmarkText/focus/ReleaseView cannot clear the gate.
Navigation, selection/edit commands and delayed cut cannot mutate it; accepted
marked update/final input clears it, Esc explicitly cancels its original owner.
Buffer replacement does not transfer a stale composition to the new buffer.
Ordinary paste refusal stays nonsticky. No rejected input string is retained.
Actual Tester/app first+ongoing retry/cancel regressions now cover Save/Select/
CloseTab/CanClose, pointer/focus/geometry release, exact buffer/disk state and
foreign-owner preservation. No app or format runtime files broadened.

01-focused actual Go exit1 preserved: core/text,core/document,core/search,
ui/editor,markdownview,graphview passed; App first/ongoing Esc cancel failed.
Cause: app window Escape shortcut outranked raw focused HandleInput. Narrow
editor-only correction registers element Escape while composing/refused; MyGo
routes focused element before global shortcut. Added a global-vs-local real
Tester assertion including ordinary Escape returning to host after resolution.
Next rerun only focused editor/app cases; prior six-package evidence retained.

Frozen:02 editor/app focused correction exit0;03 exact23existing compatibility
checks across search/editor/Markdown/graph/app exit0.11source final manifest and
implementation-result.md saved. Four runtime files only, seven new test files,
19new top-level tests. No more implementation/checks until root activation.
