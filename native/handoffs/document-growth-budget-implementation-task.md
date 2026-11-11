# Active document growth admission implementation

2026-11-10. Owner Astra /root/read_native_handoffs_tab_workflow_resume_task_md.
.27 source/package/evidence acceptance and docs are complete. This is a new
source increment, not in .27. Root coordinates/reviews only. No children.
Read native/AGENTS.md. GLM earlier audit is background; the concrete policy
below supersedes its rejected blanket-guard proposal. GLM design took too long
without an artifact and was interrupted; do not wait for it.

Implement this small policy in core/text and affected callers:

- Put the ordinary128MiB UTF8 logical-content default at the text/lower layer;
  document.MaxEditableBytes may alias it while remaining int64 and preserving
  its distinct stable disk-read role. No text-to-document import.
- Keep New(value) compatible and UTF8-validation behavior. Add an explicit
  positive-limit constructor for reuse/tiny tests, with an immutable base
  limit and an effective ceiling=max(base limit, initial decoded UTF8 bytes).
  Do not add a mutable post-edit limit setter. Small-limit constructor may
  follow existing panic-for-programmer-invalid-argument conventions.
- Preserve oversized decoded seeds in full. Ordinary edits may grow only up
  to this fixed seed ceiling, including after shrinking. This is intentionally
  a logical growth bound, not a promise all documents are<=128MiB.
- Restore(snapshot) is already explicit reload/reset and clears history. Keep
  its void signature/buffer identity; re-seed the effective ceiling using
  the destination buffer's fixed base and incoming snapshot bytes. Preserve
  existing snapshots, encoding/BOM/untouched-save and Revert behavior.
- Undo/Redo/CancelGroup replay previously accepted states without new refusal;
  the fixed history-lifetime ceiling proves them legal. No dropped history,
  new bool failure, or automatic content truncation.
- ReplaceUTF16 validates UTF8 and range/surrogate boundaries, computes replaced
  byte length with byteOffset, then checks prospective bytes BEFORE sliceBytes
  copies removed text or pieces/history are allocated. Use safe subtraction
  comparisons, not potentially overflowing sums. First rejection changes no
  text/version/selection/history/group contents. Expose a stable sentinel and
  enough limit information for visible actionable errors. Keep error precedence
  deliberate and no-op selection behavior compatible.
- Minimal size-query/check helper(s) may support caller preflight. Avoid a broad
  transaction framework. ReplaceAll must precompute/validate replacement sizes
  for its actual reverse-order edits before BeginGroup. Keep regex behavior and
  existing independent history edits. Refuse before mutation if any planned
  intermediate state exceeds the ceiling, even if the final net size would fit;
  document/test this conservative boundary explicitly. No huge full-result
  concatenation just to check size, avoid retaining duplicated literal strings.
- Indentation can calculate all positive growth before BeginGroup; overflow-safe
  line-count arithmetic. Outdent remains legal. Do not claim revision unchanged
  for existing CancelGroup after partial ordinary failures. Preserve transaction
  ownership, selection direction, CRLF, composition and redo.
- Incoming clipboard/regex/codec strings may already be allocated. Buffer guard
  is not a bound on those allocations or whole memory. If regex replacements
  require planned strings, state and bound within existing match/source budgets;
  do not add streaming regex/clipboard rewrite in this increment.
- Existing source/format OnError/Report paths must show size refusal. Use clear
  Chinese user text via minimal existing app error seam if needed; retain stable
  errors.Is identity. No new modal/panel. Source typing/paste/IME and rich-view
  rejected flush preserve current preedit/draft and prevent host action. Any
  discovered narrow draft-loss bug is in scope and must be reported/fixed.

Ownership: core/text/**, core/document/document.go plus focused document tests,
core/search/find.go plus tests, ui/editor/editor.go plus focused tests. Add
focused real Markdown/graph draft/app integration tests and only necessary
small source error/draft-preservation corrections in those packages; list such
edits in progress before broadening. Other app functionality/vendor/workers/
plugins/formats/portable/product docs/packages are frozen. No package/host/
worker/resource/benchmark run yet. GLM/DeepSeek read-only reviews won't edit.

Verify with meaningful tiny-limit tests through public seams: multibyte and
surrogate boundaries; one-byte-over rejection with version/selection/history
and snapshot equality; replacement/shrink/no-op; oversized seed+shrink+regrow;
undo/redo/group-cancel to accepted larger state; Restore re-seed on same object;
UTF16/GBK decode fidelity and unchanged save/revert using tiny fixtures; paste
refusal; first and ongoing composition refusal preserving proper ownership;
whole ReplaceAll/regex/indent preflight without partial version changes; actual
Markdown or graph draft refusal retained through save/switch and later recover
or retry. No128MiB fixtures, no tests that merely mirror formula internals.

Execute appropriate focused groups only during implementation; keep actual
commands/outputs/exits and failures in a new scoped evidence directory. Freeze
source and publish document-growth-budget-implementation-result.md with exact
files/semantics/tests/limits and identities. Root then requests one combined
affected test/vet pass after review/corrections. Do not run it early or build.
Publish a concise progress file within this task; finish actual implementation
rather than another design phase.
