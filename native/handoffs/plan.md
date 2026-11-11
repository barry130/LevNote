# Coordination decisions — root planner

## Task ownership

Use `native/AGENTS.md` and the three `*_finish_task.txt` files. The plugin/app
agent also owns `formats/code/**`; the format agent owns the other format
packages. User reauthorized Qiandao 6 Astra on 2026-11-08: root assigns one
Astra agent to editor work. Old interrupted agents stay stopped unless root
explicitly resumes them. No agent may spawn another agent.

## Compatibility contracts for the next integration stage

The editor agent owns `ui/editor/contract.go`. Add these small optional
capabilities so parallel implementations remain buildable:

- `Flusher interface { Flush() error }` and `FlushView(View) error` helper.
  If a view implements it, the host must call it before save, export, mode/tab
  switch, plugin reconfiguration, reload and close. On error retain the view,
  document and current selection; show the failure. Never call `Release` after
  failed `Flush`.
- `Host.RecoverDraft func(title, content string) error` gives a format a safe
  way to open a conflicting unsynchronized draft as a new unsaved text document.
  The view clears its draft only after this succeeds. This is a host callback,
  not direct filesystem access by an engine.
- `OutlineItem { ID, Text string; Level, Offset int }`, where Offset is UTF16
  only when meaningful; `OutlineProvider` optionally returns entries and
  implements `GoToOutlineItem(id string) bool`. Use stable node IDs for graphs
  and source offsets/identity for Markdown. Do not duplicate parsers in App.

Prefer these optional interfaces to adding mandatory methods to `View` midway
through concurrent development. No additional engine imports in app UI.
App should expose an outline only if the current view offers it.

## Review findings requiring follow-through

- Current plugin manager still rejects disabling builtins. That violates the
  intended user-controlled format plugins. Allow every rich/syntax builtin to
  be disabled while the unconditional source fallback stays available.
- `Manifests`/`Syntax` must return copies; mutable slice exposure must not change
  live matching or styling. Sort order remains builtins, local formats, code.
- `stat` then `ReadFile` does not impose a hard read bound if a file grows. Use
  a bounded reader; include settings and install directory entry count.
- `configureTab` currently skips every readonly document. Distinguish byte
  inspector explicitly from readonly text; readonly text still has previews.
- Per-line syntax that forgets multiline comments/strings is misleading.
  Either add bounded cached lexer state or explicitly describe the limitation.
- Review `formats/code/lexer.go`'s non-ASCII dispatch: it converts the first byte
  to a rune before `isIdentStart`, then decodes the real rune only inside the
  loop. An emoji whose leading UTF-8 byte is a Latin letter can enter that
  branch without advancing `i`, repeatedly appending zero-width tokens until
  the work budget is exhausted. Decode the rune before categorizing, guarantee
  forward progress on every path, and test Unicode identifiers plus unquoted
  emoji/malformed-looking source. This matters directly to memory use.
- Equal explicit priorities must preserve registration order. Do not use
  alphabetic ID order to change which plugin wins for the same extension.
- App may not mutate the user-visible plugin setting before current views
  flush successfully. Invalid/colliding plugin IDs must not activate another
  manifest's syntax under the same ID.
- Root will review behavior test evidence and rendered screenshots. A green
  build alone is insufficient for caret geometry, dirty state and save safety.
- The root README still tells users to launch Tauri and quotes its 3.2 MB
  installer. The formats/documentation owner should add a clear native entry
  point and distinguish those legacy figures from measured native artifacts.
- Unknown-encoding files have a byte fallback but need a concrete way to use
  `document.OpenWithEncoding` (GBK/GB18030/UTF-8/UTF-16) in the application.
  This must be an explicit choice. Do not guess and silently reinterpret bytes.
- A readonly text document should support Save As to a new file: core SaveAs
  already allows that safely. The app's current early ReadOnly return blocks
  this valid workflow. A binary inspector must never save its hex presentation
  over the original; track it separately.
- ADR 0063 is free in the current index. MyGo v0.8.3's local LICENSE is MIT.
  Preserve exact required notices in native distribution documentation.

Write API-ready notices and completed test commands/results to your handoff
file. Read the other handoffs before integrating a cross-owner dependency.

## Review addendum — bounded work and stage order

Write a short acknowledgement/progress file now, even before the full task
is complete. It is the reliable communication channel when tool messages
arrive unreadable. Finish the data-safety/build stage first; label later
format/UI additions separately so they do not hide a failing baseline.

Additional concrete review findings (fix with the owning task):

- App: `refreshTree` checks the 3000-entry limit only on recursive entry, so
  one wide flat directory can still append every file. Enforce the budget in
  the entry loop and disclose truncation. Prefer bounded directory reads.
- Formats/search: `Workspace` converts the complete matching line to
  `[]rune` for every result before truncating the preview to 240 runes. A
  multi-megabyte single line with 2000 matches can allocate that line
  thousands of times. Bound preview extraction before copying/decoding and
  reuse previews for repeated hits on a line. Test this workload.
- Formats/session and app: recovery's 4 MiB draft limit is checked only after
  `Capture` copies a potentially 128 MiB document. Reject/report the limit
  before materializing oversized drafts; do not misreport them as protected
  by recovery. Preserve the existing recovery file when capture cannot
  produce a safe replacement.
- App (next resource stage): watcher callbacks run `CheckExternal` and
  `refreshTree` inside `Window.Update`; `CheckExternal` fingerprints every
  open document synchronously. Move/coalesce disk work off the UI thread
  with an immutable request/generation guard, retaining conflict detection.
  Do not read a mutable Document from a worker concurrently with Save.

## Resumed review — concrete draft/lifecycle blockers

- Formats: Markdown `syncDraft` currently slices the UTF-8 string `current`
  using `editStart`/`editEnd`, which are UTF-16 offsets. This corrupts Chinese
  and emoji bookkeeping and can panic when deletion shortens the document.
  Keep byte/UTF-16 coordinates explicit and derive prefix/suffix via validated
  buffer slices before the replacement, or use versioned original text.
  Add multi-step Chinese/emoji edit, deletion and outside-block-change tests.
- Formats: graph `Release` now clears model/layout but does not invalidate
  `lastSynced`, so returning to the unchanged document skips parsing and the
  view/export is empty. Ensure Release -> Draw/Warnings/Export rebuilds from
  the same Buffer version while retaining selection/page/pan preferences.
- Formats: current Markdown `RecoverDraft` calls the same failing sync path,
  so its conflict action cannot recover. The new host callback must actually
  create a separate unsaved copy and retain the draft if that fails.

Current owners: `astra_editor_finish`, `glm_formats_finish`,
`deepseek_plugins_app`. Root writes plans/reviews only. Every owner must write
an acknowledgement and staged results to its handoff file so absence of a
tool-message response never hides blocked work.

Graph ownership update: GLM acknowledged handoff in formats.md. Astra now owns
modules/graphview/** under graph_safety_task.txt; GLM keeps underlying format
engines, Markdown view, core document/session/search, and migration docs.

- Formats parser blocker: in `formats/markdown/inline.go`, footnote/image
  branches use `strings.IndexByte(p.source[i:], ']')` as an absolute offset
  and set `i = closing + 1`. For `prefix [^1]` this moves the cursor backwards
  and revisits the same token indefinitely. Image parsing also indexes the
  wrong suffix. Convert relative results to absolute coordinates, enforce
  forward progress for every branch, and test nonzero-prefix footnotes,
  images, malformed brackets and repeated markers with bounded output/work.
- Formats parser: `splitLines` increments `i` from CR to LF before slicing
  `source[start:i]`, so CRLF line contents retain `\r`; headings include it
  and fenced-block regexes can fail. Keep the content end before advancing
  over CRLF, preserve original ranges/bytes, and test CRLF headings/fences.
- Formats export blocker: `escapeMarkdownInline` uses absolute source
  `span.Range.Start` against block-local `text`, then advances by inner-text
  length rather than the token's `Range.End`. Bold/links in later blocks can
  drop surrounding words or duplicate delimiters. Pass the block/item base,
  validate local ranges, and consume exact source ranges. Add export checks
  with several blocks, Chinese prefixes, bold/italic/code/link combinations.
  Report unsupported content in the returned warnings instead of claiming a
  clean export. Reject protocol-relative network hrefs as well as schemes.
- Formats parser resource blocker: `detectUnsupportedAt` lowercases the
  entire remaining string for every scanned character/boundary. Long lines
  with spaces/Unicode create quadratic copying. Compare only the bounded
  phrase prefix; use real UTF-8 decoding for current/previous rune boundaries.
  Also bound cumulative scans for repeated unmatched brackets/delimiters.
  A maximum input size alone does not bound quadratic processing affordably.

- Plugins current-stage review: `InstallFile` compares total error counts,
  then removes the new file, reloads, and indexes the last error after it may
  have disappeared. This can panic on installation failure. Preserve the
  concrete error before cleanup, verify the new ID/source path loaded as
  intended, and test install-limit/collision/load-failure rollback. Prefer
  atomic exclusive publish via `document.WriteNewFile` when available.
- Registry `Manifests`, `Manifest`, and `Match.Manifest` all need extension
  slice copies; manager-only cloning does not protect the exported Registry.

## Application integration review (after first edits)

- `Window.Update` is asynchronous (MyGo content.go documents it returns
  before fn). `OpenMany` must not read `reused` immediately after dispatch or
  mutate the `opening` map from both worker and UI. Use a real ordered open
  queue with acknowledged/continuation dispatch, and test delayed dispatch.
- `addDocument` flushes the former active tab but currently omits releasing
  its renderer caches. Opening N files must obey the same lifecycle as Select.
- Ordinary save should preserve source/preview/split mode and caret; current
  success path configures it back to preview. SaveAs may change the format,
  but retain a compatible mode and report a downgrade if necessary.
- Search replace and result reveal can mutate/change presentation while a
  draft is pending. Apply the same fallible flush barrier; a failed Select
  must not be followed by hidden-tab mode/selection changes.
- Review actual native asynchronous paths, not only `Window == nil` unit
  tests. A synchronous fake would hide the ordered-open regression above.
