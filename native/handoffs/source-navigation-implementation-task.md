# Activate native source navigation implementation

2026-11-09. DeepSeek read_native_handoffs_release10_ui_feasibility_task_md owns
implementation/tests in the plan's app files now. .11 accepted and immutable;
GLM only updates .11 product docs. No other product source owner, no subagents.
Use scripts/go.ps1 -GoArguments; compact before200K. Write progress/results to
handoffs/source-navigation-result.md. No package build yet.

Root accepts the small native Ctrl+G/status-action plan with these corrections:

1. Use one-time focus (AutoFocus or a pending focus flag), NOT Focus every frame.
   Every-frame Focus prevents Tab reaching submit/cancel buttons and steals
   focus from other controls. Require actual keyboard Tab/Shift+Tab and pointer
   submit/cancel tests. Keep keys/queries scoped to the visible panel. Global
   Escape must not consume a composing field's candidate-cancel key.
2. Existing Editor.SetPosition only normalizes surrogate boundaries; it does
   not normalize all grapheme clusters. Correct the plan/report: columns are
   UTF-16 units; combining-code-point boundaries remain addressable. Reject
   inside-surrogate targets before any reposition, no nonexistent guarantee.
3. Opening must not unintentionally commit a document's native preedit by
   stealing its focus. Inspect actual focus behavior; use a KeepFocus status
   button and reject/defer panel open with a short message if source OR rich
   draft composition is active. A tiny optional composition-state interface
   in ui/editor/contract.go plus trivial implementations in markdownview and
   graphview is authorized if required; these are your additional disjoint
   files, not a new navigation framework. Don't introspect a widget tree.
   Prove via tester composition events. This remains synthetic IME coverage.
4. Parse syntax first, Flush once at the host boundary, recheck captured
   Tab/Doc/Editor/Buffer identity, then validate final coordinates. A successful
   draft flush can itself change the source even if the target later fails:
   local error must disclose that edits were synchronized. Do not imply no
   document change in that case, or silently roll back the flushed edit.
5. Do not leave a closed/stale panel holding a replaced buffer after reload,
   encoding reopen, SaveAs/rematch, plugin change, or tabs removal. Prefer one
   lightweight current-target validity check per frame plus exact lifecycle
   cleanup where needed. Invalidated old callbacks must not act on a new tab.
   Failed tab switch/close retains panel just as it retains the original tab.
6. Range validation: strict positive ASCII digits, surrounding whitespace only,
   no empty column, no clamp, CRLF/empty-final-line/EOL semantics. Bound the
   short query to a small documented value using existing toolkit capability
   if available; don't introduce a custom input engine for numeric text.
   Source and split preserve their mode; preview switches only after successful
   final validation; byte inspection explicitly says displayed byte-view lines.
7. Keep toolbar/status layout usable at760x480. Mandatory one light and one
   dark tester capture with input/error/buttons visible above status. Reuse
   existing capture helper; no physical UI/helper retries. New controls need
   readable accessible labels and no extra animation/dependency.

Implement the finite feature and meaningful focused tests; use actual app
input and real Markdown draft once, not only stubs. Include Flush refusal,
coordinate change after flush, valid readonly/byte navigation, source/split/
preview, lifecycle invalidation, cancel/history, non-ASCII, long-line segment
and late-line lookup without whole-buffer conversion. Existing SetPosition
may shape only the bounded target segment; don't profile whole process.

Run focused feature tests first; on final source run one affected full app
suite plus affected modules/ui-editor tests only if their sources changed,
and scoped vet. Preserve actual outputs/failures, don't repeat accepted tests
without a new change. Once green, publish exact changed files, tests, images,
capability limits and hashes for root review. Do not update product docs or
build a release until root reviews the actual feature.
