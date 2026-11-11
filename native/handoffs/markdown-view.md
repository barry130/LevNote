# Markdown view safety handoff

2026-11-08 ownership transfer acknowledged after root stopped GLM at a tool boundary. Astra now owns `modules/markdownview/**`; GLM retains Markdown parser/export and document/session/search/docs. Graph Stage A is complete and recorded in `graph.md`.

Read `markdown_view_safety_task.txt`, plan/formats handoffs, current view/parser/editor/text implementations. Implementing independent draft baselines, precise source range rebasing, native IME flush/recovery, current-block revalidation, release/cache safety, visible bounded parse fallback and actual input regressions. No optional features, subagents, commits or pushes.

## Safety checkpoint completed — 2026-11-08

- Draft authority is independent of the frozen preview cache: exact current field baseline, UTF-16 field/enclosing-block ranges, Buffer identity and version. Local IME remains local until native commit or `Flush`, avoiding a host transaction held across input events. Ordinary committed keys synchronize immediately. `Flush` finalizes local composition and rejects foreign groups/read-only/overlapping source changes without losing text.
- New small owned core API: `Buffer.MapRangeSince(version, Range) (Range, bool)` has a fixed 64-entry coordinate-only journal (no text/history retention). Start-boundary insertions are before the range and shift it; end-boundary insertions are after it; internal insertions, insertion into an empty target, overlapping replacement and Restore invalidate. Undo/redo/cancel journal their real edits; an edited target later restored through undo/cancel is conservatively invalid, while outside edits map back. Expired/future versions reject. Caller also verifies Buffer identity and exact baseline. In-place edits to block syntax are guarded too. Existing piece metadata gives inserted units without an additional full input-string scan.
- Unrelated source edits rebase pending fields without rewriting surrounding syntax/CRLF. Clean external source replacement/undo wins: a synchronized stale local representation is cleared and reparsed, never reapplied on Save. A dirty conflict survives refresh/Release and remains visible with separate recovery/discard actions. Recovery calls `Host.RecoverDraft` with exact local text and a `.txt` title, clears only after success, and never mutates authoritative source.
- Selecting a second block maps the clicked old block range, reparses current source, requires matching kind/exact current boundaries/content, then releases the former editor. Stale cached indices cannot target another block after length/newline changes. Export uses a temporary parse, leaving active draft bookkeeping intact. Adding a paragraph flushes first and follows the last actual newline, including CRLF documents without a final newline.
- Release clears parsed source/cache and local geometry; conflicting drafts keep only their independent field text (baseline is cloned to avoid retaining a whole source substring), while clean drafts are discarded. Scroll state remains on the lightweight View. Direct failed Release remains recoverable; host normally blocks it after failed Flush.
- Actual double-click entry uses the row input handler and reaches native selectable descendants. Tests explicitly selectAll when replacement is intended. No new table/metadata/source editing features were added. Existing unsupported syntax stays visible/preserved.
- Over-8-MiB parse failure displays parse-limit/source-mode guidance and refuses empty HTML export. Individual blocks over 32 KiB show a bounded placeholder and can enter the segmented editor; fallback never shapes the huge raw block. Supported Markdown suffixes now include `.mdtxt`/`.mdtext`. Export reports parser/export warnings once after the formats exporter began returning both.
- Narrow preview uses one concise helper line and wrapping conflict controls. Inspected `build-cache/markdown-acceptance/markdown-narrow.png` and `markdown-conflict-narrow.png` at 265×480: no header collision, local draft/conflict actions remain visible and scrollable.

## Exact validation

Passing commands:

```powershell
native/scripts/go.ps1 -GoArguments @('test','./core/text','./ui/editor','./modules/markdownview','./modules/graphview','./app','-count=1')
native/scripts/go.ps1 vet ./core/text ./ui/editor ./modules/markdownview ./modules/graphview
$env:LEVNOTE_MARKDOWN_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\build-cache\markdown-acceptance'
native/scripts/go.ps1 -GoArguments @('test','./modules/markdownview','-run','TestMarkdownViewScreenshots','-count=1')
```

Uncached final safety run: text 0.687s, editor 1.236s, Markdown view 1.208s, graphview 2.119s, app 2.662s; vet exit 0. After removing a redundant journal UTF-16 scan, follow-up text/Markdown/graph/app run passed (0.888s / 1.377s / 1.570s / 3.129s). Screenshot test passed (1.162s) and both images were opened for visual inspection.

Coverage uses actual `ui.Tester` pointer/Type/Compose commands for double-click entry; composition immediately followed by Flush/export/Release/native commit/Done/second block/add paragraph; exact one-step IME undo; multiline Chinese/emoji edits and deletion; export→more input→save; preceding-block size/kind changes; outside source edits; same-field, syntax, removed target, replacement Buffer, Restore, journal expiry and canceled-target conflicts; failed/successful recovery; foreign transaction retry; clean source undo/replacement vs dirty undo; readonly/untouched bytes; parse fallback; warning deduplication and CRLF append without EOF newline. Core range tests verify all mapping boundary/direction/history cases.

Integration note: existing app `TestSaveFlushesPendingNativeIMEComposition` covers source `.txt`; graph lifecycle fake-view tests do not constitute real-adapter IME coverage. Requested DeepSeek add real Markdown + graph compose→save/tab/mode regressions; view-level transport paths already pass. No exhaustive OS IME claim or process-memory claim. A stale preview selection older than 64 document changes is conservatively rejected; completing the current edit reparses the preview. Dirty uncertain targets recover to text instead of guessing.
