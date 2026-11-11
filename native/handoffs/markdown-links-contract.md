# Native Markdown links: implementation contract

ACTIVE 2026-11-08 15:50 +08:00 after root accepted the task-checkbox release.
All implementation and test execution remain delegated. The accepted task
artifact remains available while this new finite stage is developed.

## Scope and ownership after activation

- Astra: modules/markdownview/** native inline interactions, identity validation,
  focused tests, and final feature documentation. Prove native input feasibility
  first; preserve the existing task and draft interactions.
- DeepSeek: app/** existing local-reference route and real registered-adapter
  acceptance. First demonstrate any stale callback gap, then make the narrow
  membership/current-tab correction if needed. Final integration after review.
- GLM: independent read-only review of the implemented view and app boundary;
  clarify existing parser behavior. Parser and other product code stay frozen
  unless root assigns a concrete defect. No subdelegation or new dependency.

## Native input behavior

Use parsed InlineLink values and native RichText children. Do not use ui.Link,
which can invoke the OS browser. No automatic external opening, WebView, script,
shell execution or custom document editor dependency.

Ordinary pointer selection starting inside a link must still select paragraph
text, and ordinary double-click must retain the existing block-editing action.
Ctrl/Cmd+primary click activates; Tab then Enter is the keyboard path. Space
keeps ordinary scrolling behavior. A readonly view may navigate links.

Read markdown-links-feasibility.md: a focusable clickable inline child steals
selection in the current native input dispatcher. Probe the public modifier-
sensitive HitTest seam before committing to an implementation. Test actual
input, including wrapped link fragments and a modifier changed while pressed.
Do not patch vendor without proving and reporting a specific required change.

Capture modifier and numeric source identity at the activation gesture, never
infer destination from label, ordinal, generated display text or current focus.
After gesture cancellation, no navigation. A completed gesture opens once.
Publish the exact bounded RoleLink accessible name and unavailable state to
the app owner as soon as the input probe succeeds.

## Identity and lifecycle

Capture Buffer pointer/version, original enclosing block/list-item UTF-16
range, exact inline syntax UTF-16 range and bounded full href. Convert from row
range plus local source offsets; do not scan the complete source prefix for
each visible link on every frame. Captured closures should not retain an old
whole parser document through small source substrings.

On activation Flush the current independent composition/draft first. A failed
Flush retains draft/source/view and visibly refuses. Then verify Buffer identity,
map both untouched ranges through the existing journal, check current bounded
parser context and require the exact mapped enclosing and inline ranges and
full href. An unrelated insertion above may map successfully. Target overlap,
deletion, Restore, expired journal, replacement Buffer, changed fence/grammar
context, Partial or oversized document refuses. Do not fall back to labels or
list indices. Temporary validation must not mutate the active List snapshot.

Call Host.OpenReference with the authoritative full href once after validation.
No new source transaction or Changed notification belongs to navigation. A
preceding successful Flush can edit the document: refusal wording must not
claim the source was unchanged in that case. Without a pending draft, link
navigation/export/open leaves bytes, undo history and dirty state unchanged.

## Resource and destination policy

Maximum copied full href per control:4096 UTF-8 bytes. Larger values remain
visible/source-preserved with an explicit unavailable reason; never truncate
to a different destination. Accessible display text:128 UTF-8 bytes plus a
rune-safe ellipsis. Tooltip detail:512 bytes plus a rune-safe ellipsis. Clone
only bounded retained metadata; the existing32KiB shaping guard remains.
Do not truncate a visible authored label just to satisfy the metadata limit.
An empty parsed label needs a small visible link placeholder if navigable.

The view does not decode, trim or classify targets. Host.OpenReference owns
policy: literal relative filenames from the saved source directory, full local
Windows paths, decoded-once local file: URLs and decoded same-document anchors.
Do not add relative percent decoding, file+anchor splitting or URL title syntax
as incidental scope. Unsupported URLs, UNC paths, host file URLs, unsaved
relative references and unresolvable anchors remain visibly refused.

Existing outline navigation contract applies: preview scrolls without focusing
hidden source; split reveals source and preview. Duplicate/Unicode headings
resolve through the shared outline provider, not local label search.

### Dense paragraph fallback approved after measurement

The current MyGo decoration path decodes the paragraph per underlined glyph
run. A700-link row allocated about2.15GB over five unchanged tester frames;
the practical8/16/32/64-link curve confirmed superlinear allocation. Therefore
rows containing more than16 parsed links use the existing bounded readable
Span preview for the entire row. Link labels keep the theme-readable color but
have no underline, RoleLink, focus target, retained href or navigation handler.
Show a nearby visible explanation that this paragraph has many links and that
targets remain available in source. Preserve ordinary selection/copy and
double-click editing; readonly may still read/copy/source-view. Do not make an
unexplained first16-active subset, trim destinations, or alter authored bytes.

This is a tested present UI limit, not a parser/format/source limit. Verify the
16/17 boundary and700-link fallback. Follow-up measurements found the30KiB
label at265px bounded, so no new visible-text length limit is approved. Mixed
style rows also amplify decoration work: root additionally approves at most64
total parsed InlineSpan entries for interactive linked rows. Above64, use the
same whole-row styled/nonunderlined fallback and a visible explanation of dense
inline formatting. Verify64/65, actual rendered guidance for both causes, and
selection/edit access. Source/parser budgets remain unchanged.

## Acceptance boundary

Focused view checks: real plain drag/double click, Ctrl+click, Tab/Enter, wrapped
link hit fragments, modifier changes/cancellation, task label coexistence,
readonly, same/other-block Compose, failed Flush, exact duplicate identity,
mapped unrelated edits versus changed target/context, oversized metadata and
virtualization/settling. Use compact fixtures; do not duplicate parser matrices.

Focused app checks: local target opening once through ordered route, saved base
directory, same-document Unicode/duplicate anchor in preview/split, rejection
without navigation, stale source callback, failed Flush retention, no-op source
bytes/dirty/history and save/reopen. Capture one narrow wrapped-link image for
root review; keep task-checkbox controls visible in at least one composition
case. No full release build until focused gates and root review are complete.

Native tester input is not an OS input-method candidate-window or full
accessibility acceptance claim. Explain the supported Markdown subset and the
literal relative-path policy in the capability documentation.
