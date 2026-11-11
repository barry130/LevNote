# Outline contract — approved and published

Root approved this contract; implementation is enabled. The additive types are
published in `ui/editor/contract.go`. This supersedes the earlier consumer
drafts. Astra implements both providers/views while DeepSeek integrates app.
Documentation checkpoint is complete in `docs.md`.

## Concrete additive API

```go
type OutlineItem struct {
    ID, AnchorID, Text string
    Level, Offset int
}

type OutlineSnapshot struct {
    Version uint64
    Items []OutlineItem
    Truncated bool
}

type OutlineSource interface {
    OutlineSnapshot() (OutlineSnapshot, error)
    ResolveOutlineItem(id string) (OutlineItem, error)
    ResolveOutlineAnchor(anchor string) (OutlineItem, error)
    Release()
}

// Optional field on existing Adapter:
Outline func(Host) OutlineSource

// Optional interfaces on the mounted View:
type OutlineNavigator interface {
    NavigateOutlineItem(id string) error
}
type CurrentOutlineProvider interface {
    CurrentOutlineItem() (id string, ok bool)
}
```

Existing `OutlineProvider{Outline() []OutlineItem;
GoToOutlineItem(string) bool}` stays source-compatible. New app uses the
adapter's `Outline` factory; nil means unsupported. This stage does not need a
second snapshot interface on View or a combined controller that constructs a
canvas. `NavigateOutlineItem` deliberately differs from the old bool name.

Collection failure returns error; valid no-headings is an empty snapshot;
unsupported is nil factory. `Truncated` means a parser limit, entry cap or
label/anchor clipping made it partial. App shows one concise partial notice
and keeps existing format Warnings for detailed parser issues; no new unbounded
problem list or whole-label tooltip copy. Both readonly and writable buffers
can be inspected without changing them.

`Version` is exactly the represented `Host.Buffer.Version()`. Caller also
guards tab identity and actual Buffer pointer; a matching number alone is not
a generation token. Snapshot Items are owned by the caller and bounded;
string values contain only bounded labels/IDs, not retained source substrings.
App requests a new copy only on source/tab/format changes, not every UI frame.

`Level` is **1-based**: Markdown1..6, mindmap root1/children2/...; visual indent
can be clamped independently. `Offset` is UTF-16 at the start of the Markdown
heading syntax, never a UTF-8 byte count. Mindmap Offset=-1 because its parser
does not expose a trustworthy source position. No other meaning for zero.

## Separate identity from link anchors

`ID` is an opaque navigation identity; `AnchorID` is a current-snapshot link
slug, not identity. Markdown uses recognized parser `BlockHeading` only. No
second heading regex or new parser-budget policy.

Markdown IDs encode the Buffer identity, source version and full heading
UTF-16 syntax range. A source and a preview controller with the same Buffer
understand the same ID without independently assigned counters. An old ID
resolves by `MapRangeSince` and then requires an exact recognized current
heading range. Insertion at the heading start maps before it, end insertion
maps after it; overlapping edit, Restore, expired64-edit journal, Buffer
replacement or a no-longer-heading target fails visibly. A current item may
have a new ID; Resolve returns that new current item, never searches by label
or slug. Selection preservation can use Resolve rather than raw equality.

This explicitly prevents duplicate insertion retargeting: headings `A`/`A`
have anchors `a`/`a-2`, but inserting another A before them cannot redirect a
previously captured outline click to the new first A. If the clicked heading
itself changes during Compose/Flush, conservative failure is allowed; the user
can select its new row. Unrelated edits within journal history continue to map.
IDs are in-process only and never stored as document fields or session data.

Markdown AnchorID algorithm is shared by outline extraction and HTML export:
Unicode lowercasing; keep letters/numbers/combining marks; whitespace/hyphen
runs become one hyphen; drop other punctuation; trim hyphens; empty => section.
Bound base slugs at256 UTF-8 bytes on rune boundaries. Duplicates choose the
first globally unused suffix starting at -2, so `a`, a duplicate `a`, and
literal `a-2` cannot collide. Parser-order makes links deterministic for the
current source. Anchor resolution intentionally resolves the *current* slug;
item resolution never uses it. HTML h1..h6 IDs use this exact helper and proper
HTML attribute escaping. Unsupported/Partial export keeps the current refusal.

Mindmap IDs are its stable file-level node IDs. AnchorID is empty; anchor
resolution returns actionable unsupported error. Resolve checks the current
parsed tree contains that ID; the app's Buffer guard prevents reload crossing.
Graph has no additional invented source coordinates or copied metadata fields.

## Bounded observational source

Factory receives the current Host but stores only the Buffer needed for
inspection. It does not call normal Adapter.Create, View.Draw,
graph.ensureSynced/layout, Flush, Host.Changed or selection operations.

Source cache key is Buffer identity+version. It retains at most2048 bounded
item descriptors; display Text is at most512 UTF-8 bytes with explicit
ellipsis, ID/slug work has finite limits. Huge mindmap IDs are not clipped
into another identity: skip unsupported over-budget IDs and disclose partial.
No full node topic/note/metadata string is copied into the outline. Returned
tooltip/a11y uses the same bounded Text; source contains full text.

Markdown uses its bounded existing Parse and shared heading metadata helper;
temporary parsed data is discarded after collection. Mindmap parses with
existing byte/depth/node guards, walks pre-order including folded children,
then drops the parsed tree without layout. No parser budget expansion.
Repeated snapshots reuse cache without parse; Release drops cache and source
parse temporaries, never flushes or mutates. Only the active tab needs one
source controller; tab/reload/plugin invalidation releases it.

## Host activation lifecycle

1. Capture active tab, Buffer pointer and requested item ID from current rows.
2. Run existing fallible `flushTab` before any navigation. Failure keeps tab,
   mode, selection, view and draft; return its existing report.
3. Revalidate captured tab/current tab/Buffer. Refresh source snapshot if
   version changed; call `ResolveOutlineItem(oldID)` to obtain current target.
   Do not use old item coordinates, row index, label or AnchorID.
4. Revalidate Buffer/version once more after resolving, before action.
5. Source mode with Offset>=0: use `Editor.Reveal(offset,offset)` only; do not
   mount/call preview. Source mode with Offset<0: report
   “此格式没有源码定位坐标，请切到阅读/分屏定位”; do not change mode, graph folds,
   graph selection or source caret.
6. Preview: call mounted View's `OutlineNavigator.NavigateOutlineItem(id)`.
   Split: call Navigate first; on success reveal source only if Offset>=0.
   Negative graph offset still navigates its existing pane, keeps source caret
   and reports “已在图形中定位；此格式没有源码定位坐标”.
7. View navigation itself defensively Flushes and re-resolves the incoming ID,
   so direct calls remain safe. After successful Navigate, if Buffer version
   changed (e.g. uncollapse), refresh snapshot before updating list selection.

Anchor links use an explicit route: host Flushes, then source controller's
`ResolveOutlineAnchor(decodedFragment)` obtains the current ID and the same
mode-aware action. Existing `openReference` must stop passing a slug blindly
to `GoToOutlineItem`; no ambiguity between old item ID and anchor slug.

## View behavior and UI limits

Markdown preview finishes a successfully synchronized draft, rebuilds its
virtual-row descriptors if required, finds the resolved heading and calls
existing ListState.ScrollTo. It does not build all offscreen widgets. Partial
preview that cannot expose the heading returns “请切到源码定位”; source mode
can still use truthful offsets among returned partial entries.

Mindmap preview resolves stable node ID. To reveal a descendant, collect only
its collapsed ancestors and clear those flags in one existing undoable model
transaction, preserving unknown fields. Include a collapsed root if it hides
children; selecting the root itself needs no expansion. Readonly hidden-node
navigation returns a limitation before changing selection/pan/folds/content.
Visible readonly targets can be selected and centered. Selection does not
rename/edit the node or create a second active input. Keep existing zoom;
adjust pan only enough to locate the target. Undo restores collapsed metadata.

CurrentOutlineItem is optional and observational: Markdown finds the closest
preceding heading from the host caret where meaningful, graph uses selected
node. It returns false when no truthful current item exists. No synthetic
selection fallback from an arbitrary graph root.

DeepSeek owns one existing-sidebar toggle and virtualized `ui.List`; keyboard
Home/End/arrows/Enter, far-row scroll and stable selection use these bounded
items. Unsupported/empty/error/partial states remain explicit. At760x480 and
dark theme, sidebar and editor stay within the existing layout. No new window,
image/network operation, table/metadata controls or extra format scope.

## Focused acceptance

Parser-recognized headings only; Unicode/duplicate/literal-suffix slug collision;
shared HTML ids; byte→UTF16 after emoji; source insertion at boundaries then
far-item activation; duplicate insertion during Flush; edited target and expired
journal rejection; independent source/view ID resolution; no inspection side
effects or graph layout; label/item caps and release; actual native Compose and
conflict retention; source/preview/split mode behavior; readonly collapsed root/
descendant and one-step undo/unknown metadata; app virtual rows/narrow/dark/tab/
plugin/reload. Focused tests precede one final integration pass.

## Astra implementation checkpoint — focused tests/vet passed

Concrete API is in `ui/editor/contract.go`; Markdown/mindmap adapter factories and
mounted navigation are implemented. Markdown shares `formats/markdown/outline.go`
with HTML heading IDs, keeps independent range identities, and maps old IDs after
Flush through the 64-edit journal. Unicode/duplicate/literal-suffix anchors have
one bounded generator with per-base suffix cursors; byte-to-UTF16 advances once.
Source caches retain bounded copied descriptors only. Mindmap source parses but
never mounts/layouts; folded descendants are included. Both provider caches are
keyed by Buffer identity+version, Release drops them, and byte limits are checked
before Buffer.String. Snapshot labels <=512 UTF8 bytes, entries<=2048, huge file
IDs skipped rather than clipped; partial/error/unsupported remain distinguishable.

Markdown navigation finishes only a successfully synchronized and resolved draft,
then ScrollTo uses the existing virtual rows. A stale/conflicting target keeps the
draft. Mindmap navigation reveals collapsed ancestors in one undo transaction,
including a hidden collapsed root; visible root selection does not expand it.
Readonly hidden target failure occurs before ensureSynced/selection/pan/layout.
Zoom is retained even when navigation precedes the first paint. Source graph
mode policy stays in the host: no fake source coordinate and no forced mode.

A new focused regression found LayoutBoth ignored root.Collapsed while the other
three layouts honored it. Root expressly extended ownership for the small guard
in `formats/mindmap/layout.go` plus cross-layout tests; no other layout redesign.
All four layouts now test root selection, descendant reveal and byte-exact undo.

Exact focused validation:
- `native/scripts/go.ps1 -GoArguments @('test','./ui/editor','./formats/markdown','./formats/mindmap','./modules/markdownview','./modules/graphview','-count=1','-timeout=180s')`
  PASS: editor2.604s; markdown0.612s; mindmap0.641s; markdownview1.611s; graphview2.590s.
- `native/scripts/go.ps1 -GoArguments @('vet','./ui/editor','./formats/markdown','./formats/mindmap','./modules/markdownview','./modules/graphview')` clean.
- Owned `git diff --check` clean.

Substantive new tests cover shared HTML/Unicode/colliding slugs; emoji UTF16;
old ID after source insertion/Undo/view Release/recreation; duplicate insertion
DURING native Compose Flush; edited target/expiry/Restore/replaced buffer rejection;
full native Compose→navigate and conflict retention in both views; partial parser
source offsets vs preview refusal; readonly hidden/root behavior, unknown metadata,
one undo; bounded labels/count/ID/parse failure and cache release; far row1999
visible at532x420 with <40 built preview rows. Tester transports Compose/preedit;
this does not claim a physical OS IME candidate-window exercise.

Headless benchmarks, Ryzen7 6800H, windows/amd64, 5 iterations, `-benchmem`:
`test ./formats/markdown ./modules/markdownview ./modules/graphview -run '^$' -bench 'Benchmark(HeadingAnchorsRepeated|MarkdownOutline|MindmapOutline)$' -benchtime=5x -benchmem -count=1`
- 8192 repeated slugs:6.292ms/op,1,364,187B/op,24,559allocs/op.
- 2000 Markdown headings cold:6.329ms/op,4,227,992B/op,17,964allocs/op.
- 2000 Markdown snapshot cached:0.0192ms/op,131,072B/op,1alloc/op.
- 2000 mindmap nodes cold (parse/no layout):14.872ms/op,7,860,992B/op,138,026allocs/op.
- 2000 mindmap snapshot cached:0.0915ms/op,131,094B/op,1alloc/op.
Cached snapshot deliberately returns a caller-owned descriptor slice; host should
request only on tab/version/format changes, not every frame. These are bounded
headless allocations/timings, not Windows process memory or Tauri comparisons.

Outstanding: DeepSeek app source/preview/split/sidebar keyboard/far-row/narrow/dark
and final integrated evidence. CurrentOutlineProvider is observational; app
sampling frequency determines tracking, so no continuous caret-follow promise.
Astra's implementation is stable for integration; final feature documentation
will be updated after the app acceptance result.

## Post-review selection and narrow readonly corrections

Root identified that hidden-node expansion's whole-buffer serialization moved the
shared source selection to EOF. `commitBufferSelection` now restores the original
numeric UTF16 anchor/head before EndGroup for outline uncollapse only; Undo/Redo
include the correct selection. Forward/reversed nonzero range tests pass. If the
new serialization is shorter, endpoints clamp to EOF; an endpoint inside a new
surrogate pair moves to its preceding boundary. Both cases are tested and show
an explicit graph warning. No semantic same-JSON-field source mapping is claimed.
DeepSeek added the corresponding real split hidden-node reverse-selection test.

Graph full package after this fix: `test ./modules/graphview -count=1 -timeout=180s`
PASS1.500s; `vet ./modules/graphview` clean. At DeepSeek/root request, removed the
generic host-readonly warning duplicated by app and view; inspector readonly and
page/format warnings remain. Focused `test ./modules/graphview -run
'ReadOnly|Readonly|Outline' -count=1 -timeout=90s` PASS1.035s. Product code frozen
again for final app acceptance. Root's app screenshot tab/mode sizing fix belongs
to DeepSeek; no unrelated view layout work added.
