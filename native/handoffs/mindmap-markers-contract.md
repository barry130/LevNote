# Native mindmap markers — final root contract

Status: FULL INTEGRATION ACCEPTED / PRODUCT FROZEN. Root reviewed focused
format/view/app evidence and four final760x480images, then full20-package
test/vet/build,15/15evidence checks and hidden startup/ownedPID12100 WM_CLOSE.
Final integrator was astra_marker_view after the explicit ownership transfer;
older owner names below describe implementation history only. Accepted binary:
13,461,504 bytes; SHA256
17E014EB6F61E671F6FCCA439719E8831B7591042A6F29EE5354960C120C0466;
UTC2026-11-08T16:04:08.3209774Z. Evidence is
artifacts/acceptance/mindmap-markers-integration-20261109-000341 and matching
startup-smoke-mindmap-markers-20261109-000341. Separate root-acceptance.json
and evidence-notes.json retain the accepted aggregate/validator unchanged.
No resource sampling, physical keyboard/IME or full visible-window acceptance
is inferred. Prior accepted cell binary and actual234000-current-run failure
remain preserved; the overwritten older230124attempt cannot be reconstructed.

Originally activated after root accepted the Markdown single-cell artifact
(13,425,152 bytes, SHA256 34F3F48C98300B053B0BA0E69BE9D9F89CDA15E17759E6A26AD817A1E9C1D14C).
This document supersedes both
marker design proposals. Root plans/reviews only; child agents implement,
test, build and update product documentation. No child delegation.

## Finite scope and ownership

Add native editing of the existing optional `icons` string array. Reuse the
existing selected-node inspector, graph caches, host buffer and undo boundary.
No new editor instance, engine, dependency, format version, network access,
tag/image editing, outline extension, custom-marker input or import feature.

- glm_formats_finish: formats/mindmap/** additive Icons/SetIcons API and
  meaningful focused regressions; publish the exact seam/results promptly.
  After finishing its files it becomes read-only view/app reviewer.
- astra_marker_view: modules/graphview/** marker UI/actions/lifecycle,
  focused native tests and component images. No Markdown edits.
- glm_marker_app: finalize accepted cell release docs, then app/** real
  adapter marker acceptance and final product docs. Shell product code changes
  only for a demonstrated defect. Sole full integration after root gate.

Other formats, core, editor, vendor and scripts stay frozen. No resource
sampling or repeated benchmark. GLM/DeepSeek compact before200K context.

## Pure format seam

`func (d *Document) Icons(id string) ([]string, bool, error)` returns an exact
defensive ordered slice copy plus presence. Absent returns false; present-empty
returns a nonnil empty slice and true. Read never changes dirty/serialization.

`func (d *Document) SetIcons(id string, values []string) error` stores present
icons. Nil input means present `[]`, distinct from absent. Validate every input
string as UTF8 before assignment; missing node/invalid input rejects atomically.
Clone input, preserve order/duplicates/arbitrary glyphs. Same ordered values on
a present field are no-op, preserving old pointer/dirty state/original bytes.
On a real change, rollback the exact old pointer and dirty state if document
validation fails. Do not expose a partial candidate. No glyph vocabulary,
sorting, deduplication or new count/byte limit belongs in the format layer.

The existing []string canonical writer already serializes nil inner slices as
`[]`; Node.Clone is not broken and requires no unsolicited rewrite. Regression
must cover present-empty SetIcons/Clone/AddNode round trips. No remove-field
API: explicit clearing of actual markers stores present-empty.

## Palette and native UI

Use one collapsible `标记` section in the existing224DIP scrolling inspector,
after the current field/editor/error controls. It must not replace or discard
the topic/note/link editor. Start collapsed to avoid clutter; show bounded
current count/summary and a discoverable toggle. When open, constant native
buttons grouped into priority1..9, color6 and symbol4, with Chinese accessible
names/tooltips and visible selected state. Use both text and shape/state, not
color alone. Pointer, Tab+Enter and Space must work without triggering twice.

Fixed glyphs: `1️⃣ 2️⃣ 3️⃣ 4️⃣ 5️⃣ 6️⃣ 7️⃣ 8️⃣ 9️⃣`,
`🟥 🟧 🟨 🟩 🟦 🟪`, and `⭐ ✅ ❗ ❓`.
Priority/color are exclusive only on adding a different member. If the chosen
glyph exists, remove all exact occurrences of it; otherwise remove members
of its exclusive group and append the chosen glyph once. Other entries keep
their exact order/duplicates/values. Symbols are independent. Existing multiple
same-group glyphs stay visible until an explicit user action changes them.

`清空全部标记` explicitly removes every existing glyph, including custom ones,
and writes `[]`. Explain this in its tooltip. It is disabled/no-op when the
field is absent or already empty; opening/viewing/closing never creates `[]`.
Known toggles preserve unknown entries. No separate custom-glyph deletion UI.
The bounded summary may ellipsize and must say how to inspect full source.

Read-only may reveal/read the section but cannot mutate. Missing selected node,
parse failure, foreign transaction, preview limit or unavailable buffer must
never become a silent fallback target. Errors should identify the blocked
operation, not claim a successfully flushed previous field is still unsaved.

## Bounded state and exact action identity

UI editing budget: at most64icons and4096UTF8bytes in aggregate, both for the
existing list and candidate. Check count first, then bounded length sum before
copying. Existing over-limit data remains fully preserved and displayed through
existing bounded canvas/summary; actions including clear are unavailable with
source-edit guidance. The pure format must still accept larger valid arrays.
Tooltip/summary caps512UTF8bytes rune-safely. No per-frame full-source or icon
list clone, candidate construction, parse, or serialization. Collapsed section
must not build all palette controls. The19 controls and metadata are constant.

At the actual pointer/key gesture capture the specific action, Buffer identity,
selected node ID, exact icons presence and bounded ordered glyph list. Do this
BEFORE Flush. Do not wait until Clicked to read a potentially different current
selection/list from a rebuilt fixed-key control. Retain bounded snapshots only
through the gesture; cancel/scroll/collapse/Release clears gesture state.

On activation:
1. Flush the existing independent field draft. Failed Flush keeps its exact
   text/selection/history/error/recovery state and blocks the marker action.
2. Check captured Buffer and selected node still match, readonly/group/preview
   bounds, then refresh/reparse current authoritative source using existing
   guards before materializing. No current-node fallback on mismatch.
3. Require that captured node still exists and current icons presence/list
   equal the captured baseline. Any icons change, reorder, deletion, or field
   presence change refuses visibly. Unrelated topic/note/other-node/source
   edits may merge only while this identity holds, preserving those edits.
4. Derive candidate under count/byte limits. A semantic no-op adds no history,
   Changed callback, serialization or dirty state. No-op clear on absent/empty
   is silent because the user-visible intention is already true.
5. SetIcons on the current parsed candidate, serialize/reparse through existing
   checks including candidate document byte limit, and commit exactly one host
   transaction for the marker action. A separately pending field Flush may
   legitimately be its own earlier transaction; do not promise one undo for
   both actions. Preserve numerical source selection where legal using the
   existing selection-preserving boundary, and give marker-specific wording if
   clamping is needed. Do not reuse the outline-only '展开节点后' warning.
6. Keep selected node and field; update parsed/render caches normally. Unknown
   top/node fields, rich tags (raw/style/className/extra), images and number
   spellings survive. Real edits follow existing graph canonical serialization;
   only untouched/no-op documents promise byte-identical preservation.

No source offsets/pointers into an obsolete parsed Node may cross the action.
Button identity/state may not cause an old press to toggle a newly selected
node. Failed actions have no marker transaction or partial mutated cache.

## Focused acceptance and evidence

Format: read presence/copy; absent-to-empty; same-list no-op with exact original
bytes; caller/returned-slice alias safety; Unicode/multi-rune glyph/order and
duplicates; invalidUTF8/missing node; validation rollback from clean AND dirty;
present-empty Clone/AddNode; rich metadata preservation after real edit;65icons
accepted by pure API (UI cap not parser cap).

View: actual native pointer and Tab/Enter/Space, collapse/discovery, selected
states; exclusivity/toggle/targeted duplicates/custom glyphs; clear[]/no-op;
captured Buffer/node/list before held gesture with redraw; unrelated source and
pending field Flush merge; same-list change/presence change/node deletion,
Buffer replacement/group/readonly/oversize refusal; failed field draft retained
and recoverable. Test64/65 and4096/4097 plus prospective overflow. Successful
marker action one Changed/undo; undo/redo and legal source selection preserved;
no-op unchanged bytes. Release clears gesture and caches. Existing note/link,
drawio and bounded-label/no-idle regression tests remain in package validation.

Use fresh independent before/after expected arrays and rich fixtures, not the
same toggle helper to manufacture the expected result. App tests target the
actual registered adapter and native controls, exact save/reopen/BOM handling,
undo/redo, SVG metadata, source/preview/split and readonly/pending/conflict
barriers. Avoid repeating all view helper tests at app layer.

Capture final component and integrated760x480light/dark states with marker
section and recovery controls reachable through the existing inspector scroll.
Do not claim every row/control is simultaneously visible. Run owned package
tests/vet once stable; root reads actual source/results/images, then sole app
integrator runs full native test/vet/release/isolated hidden startup+WM_CLOSE.
No new OS IME/GPU/resource proof is implied by tester or hidden startup.
