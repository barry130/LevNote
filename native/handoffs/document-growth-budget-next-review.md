# Document growth budget audit

2026-11-10 read-only planning review. `.27` source is frozen; no source, tests,
product docs, builds, launches, benchmarks, or child delegation were performed.

## Present gap

The128MiB limit currently governs stable file reads only. It does **not**
govern live logical-document growth.

- `core/document/document.go:126` applies `MaxEditableBytes` in `open()`.
  `OpenLimitedRead` likewise caps its read at `document.go:168-190`.
- `core/text.Buffer.ReplaceUTF16` has no content-size admission at
  `core/text/buffer.go:231-255`. It validates UTF-8 and range, then can grow the
  logical buffer without checking `LenBytes()`.
- `ui/editor.Editor.ReplaceText` delegates directly to that API at
  `ui/editor/editor.go:270-289`. Therefore typing/IME, paste, delete, tab/indent,
  newline insertion, and all direct text/format writebacks can grow a live
  document past128MiB.
- Paste is particularly exposed because `Editor.Command("paste")` reads the
  full clipboard string at `editor.go:541-553` and calls `ReplaceText(nil,
  value)` without any incoming-size check.
- Current/all find-replace can also grow without a document ceiling:
  `core/search/find.go:101-115` and `:116-145` call `ReplaceUTF16`.
- Undo/redo can restore a previously accepted larger logical state:
  `core/text/buffer.go:295-325`. `Restore` can replace content from a snapshot
  at `buffer.go:196-206`; document reload/revert uses that path.

These paths are real logical-growth paths. The gap is not merely theoretical,
but no huge fixture was constructed and no runtime behavior was measured here.

## What the existing limits do and do not mean

- **History budget:** `text.New` gives each buffer256 undo entries and32MiB
  history at `buffer.go:92-99`; `SetHistoryLimits` trims old undo/redo entries
  at `buffer.go:327-365`. Truncating history means some older states can no
  longer be undone. It does not limit the current document: a large edit can
  remain live after its undo transaction is trimmed.
- **Parser limits:** Markdown8MiB, mindmap/draw.io source limits, table and
  outline budgets reject rich parsing/processing while preserving source. They
  do not prevent source editing beyond those limits.
- **Recovery limits:** a dirty draft above `session.MaxDraftBytes` (4MiB) is
  refused during session capture at `core/session/builder.go:59-64`; the
  aggregate draft budget is16MiB. This protects recovery output, not live
  content. A live draft can exceed those limits until the next recovery
  capture fails.
- **Live tab cap:**100 tabs bounds tab count only. It does not establish a
  worst-case content budget. No current write admission guarantees128MiB per
  live tab, so that claim must not be made.
- **Read-only byte fallback:** `document.Inspect` is capped to64KiB by the
  caller. It is not an editable-document admission route.

## Smallest useful next increment

Add **one buffer-level logical-content admission rule**, not a UI paste-only
check and not a new global scheduler/cache framework:

1. Reuse `document.MaxEditableBytes` as the live logical UTF-8 byte limit.
   Do not invent a second number.
2. Enforce it in `text.Buffer.ReplaceUTF16` before mutation/history/allocation.
   Reject when `current bytes - replaced bytes + len(value)` would exceed
   `document.MaxEditableBytes`. For ordinary edits, calculate from existing
   `LenBytes()` and `len(old)` without building a full new source string.
3. The same pre-mutation check must cover undo, redo, group commit, and
   snapshot restore where those can enlarge the logical buffer. This avoids
   bypasses through an API seam that only guards typing/paste.
4. Preserve:
   - below-limit normal typing, paste, shrinking, replacement, IME grouping, and
     exact source bytes;
   - selection/version/history on accepted edits;
   - existing parser and recovery limits as separate budgets;
   - readonly behavior and source-fidelity guarantees;
   - no silent truncation or eviction.
5. Error UX should be visible and actionable: report that the edit would exceed
   the current document-size limit, that no source/history change was made, and
   that shrinking/replacing content first can make room. `Editor.ReplaceText`
   already has `OnError`, so app-level errors can continue through the existing
   reporting seam.
6. A later docs owner should narrow the current wording
   `文本编辑读取上限为 128 MiB` to distinguish stable file reads from live
   content admission. No product doc is changed here.

### Compatibility notes

- Existing files at exactly128MiB remain valid and can be edited if an edit does
  not grow them. Deleting/replacing/shrinking must work.
- A paste or replacement that shrinks or equals the current size must work.
- If an earlier undo transaction would grow the buffer past the limit, undo
  should refuse without losing the current state. This is an unavoidable
  consequence of enforcing a live ceiling; it must be explicit, not silent.
- Some older undo/redo states may already be unavailable due to history
  trimming. That remains a history-budget disclosure, separate from logical
  admission.
- This change does **not** promise process memory, smoothness, or global
  workspace growth bounds. It only makes the existing editable-content number
  govern logical live growth.

## Focused small-limit test strategy

Use a small injected/parameterized limit, not a huge128MiB fixture:

- boundary: at limit−1, at exactly limit, and one byte over on insert;
- paste and direct `ReplaceUTF16` both accept a within-limit change and reject
  one byte over without source/history/version/selection change;
- shrinking/replacing at an over-limit candidate still succeeds after first
  reducing content;
- IME group insertion rejects before the first mutation, preserving the
  original text/history/selection and not leaving a partial composition;
- ReplaceAll rejects the first growth edit without committing earlier edits;
- undo/redo refuses a growth transaction while preserving the current state;
- snapshot restore/reload rejects a restoring state over the limit and keeps
   the existing buffer intact;
- parser and recovery limit behavior remain separately visible.

These tests should use tiny fixtures and public seams; no huge buffer
construction is needed. They should not claim performance or whole-process
memory improvement.

## Exact source identities read

- `core/document/document.go`12493B, SHA256
  `C405449A2BD981EEE09DEFB89F7C113CF40E0254D6673DC99E2945171C08E1F4`
- `core/text/buffer.go`16189B, SHA256
  `562F1293800CF147FE466F1B64E29FA5627591F1E8CCC698FEAE9163534E2E9A`
- `ui/editor/editor.go`19929B, SHA256
  `6438B438F0409BEEB26A9427C5CC8EBF971132242933B1D927BEBFEA4371AC81`
- `core/search/find.go`3782B, SHA256
  `6F1C5F326CD946C784A8D5F1CA5E3F9754B8A3AA31CB307751F6B2699D11E1B4`
- `core/session/session.go`7027B, SHA256
  `BD3593BCBA8E22FC96B1A27B8BD772C9E183D413723D9346D0AE4173AFCE9FE3`
- `core/session/builder.go`6301B, SHA256
  `3488EEAD7285501324DA9D75C418B9FC5D93EEBE9B22EFF1E4556480931D7121`
- `native/README.md`66579B, SHA256
  `438E01C4F549BA1CE8788BE1BB9F499005D0CC60A00A0AB2D20A1A07900E36F7`

Evidence limit: static source reading only; no product execution or resource
measurement.
