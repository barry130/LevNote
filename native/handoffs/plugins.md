# Plugins/App handoff

## ACTIVE — oversize rich-preview app acceptance test plan (2026-11-08)

Root activated `oversize_preview_stage_plan.txt`. Astra owns the view/parser
guards, exported default constants, cache cleanup and view-level allocation
tests; DeepSeek owns app lifecycle coverage and composition tests only. No app
shell behavior will be guessed before Astra publishes the exact guard
checkpoint. Current schema2 source-plugin checkpoint remains accepted/frozen.

Planned focused app tests:

1. **Oversize refusal preserves source authority**
   - For Markdown (>8 MiB), mindmap (>16 MiB) and drawio (>32 MiB), cross each
     default by the smallest practical amount.
   - Exercise source -> preview/split at app level; verify source mode remains,
     the exact buffer pointer/version/selection and undo history survive, the
     editor remains editable when the document is writable, and readonly text
     can still reveal/navigate without mutation.

2. **No stale preview/export publication**
   - Begin with a small valid preview, replace/edit the same buffer above its
     parser bound, force the relevant view refresh, and confirm old parsed rows,
     graph canvas/descriptors and `ViewWarnings` cannot be exported or displayed
     as current content.
   - Export refuses with no output file publication and a visible actionable
     source route.

3. **Draft/Flush safety**
   - With an actual Markdown draft or graph dirty field, make an independent
     oversized edit or trigger an oversize refresh; a failed `Flush` must keep
     tab/view/draft/selection/controller identity unchanged.
   - A successful flush followed by oversize refusal may release the rich view,
     but must not discard pending independent source/draft recovery.

4. **Lifecycle recovery**
   - Repeated unchanged frames after refusal perform bounded/no repeated parse
     attempts.
   - Shrinking below the parser bound refreshes and mounts a valid view again.
   - Reload into a within-bound new buffer recovers normally; changing buffer
     identity/version cannot reuse a stale refusal or stale view.
   - Mode/tab/plugin/refresh changes retain the existing Flush-before-lifecycle
     ordering.

5. **App boundaries**
   - Installed source-only schema2 remains source-only and never reaches these
     rich guards.
   - Existing builtin precedence and source fallback remain unchanged.
   - Assertions use parser-owned exported constants/defaults rather than copied
     magic byte counts.

Acceptance sequence after Astra's API checkpoint: focused `./app` tests/vet,
record exact API and failures in this handoff, then root source review. One
full suite/vet/build and isolated smoke only after review and only if product
code changes. App will not import concrete parser internals or edit
`modules/markdownview/**` or `modules/graphview/**`.

## FINAL FROZEN — schema2 installed source outlines (2026-11-08 14:43 +08:00)

Root authorized final integration after source review and the baseline-highlight
compatibility correction. Product source is frozen. The accepted 14:03 artifact
below belongs to the preceding outline stage and is historical; it is not the
current schema2 binary.

Implemented in the owned app/modules boundary:

- `Tab.OutlineFactory` is separate from rich `Tab.Adapter`; installed schema2
  formats keep `Adapter == nil` and expose source-only editing plus an optional
  neutral outline factory.
- `modules.InstalledSourceOutlineFactory(pluginID, syntax)` is the only
  plugin-DTO to `sourceoutline.Config`/`sourceview.NewOutline` composition seam.
  `app` stores a generic `func(editor.Host) editor.OutlineSource` and does not
  import either concrete source-outline package.
- configure/plugin refresh/disable/enable/reload invalidates factory and
  controller generations. Source navigation flushes and revalidates tab,
  adapter path/mode/controller/factory generation/buffer before resolving an
  old ID; stale controllers are never reused.
- preview, split and rich export reject installed source-only tabs before
  touching `Adapter`.
- source capability text is captured as a compact badge/detail on `Tab` during
  configure. The editor frame no longer calls `PluginManager.Syntax` merely to
  render the header, avoiding repeated deep copies of keywords/extensions/rules.
- every valid declarative schema keeps the baseline quoted-string/number
  highlighter already provided by `code.HighlightDeclarative`; configured
  line-comment/block/keyword rules are additive. schema2 without outline rules
  remains outline-free but still retains baseline highlighting.
- capability text distinguishes baseline highlighting from configured rules.
  schema2 rules say source outline plus line-local highlighting and disclose
  that multiline/comment/string context is not recognized.
- the source partial notice now covers content or label clipping instead of
  claiming an entry limit. It names the finite `2 MiB / 65,536 lines / 2,048
  items / 512 bytes per label` bounds and directs the user to the full source.

Focused tests added/passing in `app/source_plugin_outline_test.go` cover real
`InstallFile`, exact BOM/CRLF source after open/unmodified save, edited save,
Unicode and duplicate headings, pending Compose -> Flush -> old-ID resolve,
readonly navigation without mutation, disable/re-enable release/recreation,
changed rules invalidating old IDs, failed Flush retaining source/caret/view,
factory replacement during Flush rejecting the captured controller, schema1
and empty-schema2 fallback, enabled builtin precedence, source-only rich-mode
rejection, 2,000-row virtualized keyboard activation, zero-item partial
guidance, and 760x480 geometry.

Exact final commands and results:

```powershell
native\tools\go\bin\gofmt.exe -w native\app
native\scripts\go.ps1 test ./... -count=1 -timeout=300s
native\scripts\go.ps1 vet ./...
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

Full test PASS: `app` 4.270s, `core/plugins` 2.963s,
`formats/sourceoutline` 2.655s, `modules/sourceview` 2.741s; every other
native package passed. Full vet exit0 with no diagnostics. The single release
GUI build exit0.

Final executable:

- Path: `F:\AIproject\LevNote\native\bin\levnote-native.exe`
- Bytes: `13,201,920`
- SHA-256:
  `4F9145A3E09AA2731F0FD365502CADE53E755467A376AC45BCAE610542848B8F`
- UTC mtime: `2026-11-08T06:41:35.3481194Z`
- Local mtime: `2026-11-08T14:41:35.3481194+08:00`
- `go1.27.3`, Windows amd64/v1, CGO enabled

Exact visual artifact directory:
`native\artifacts\acceptance\source-plugin-20261108-144035\`

It contains:

- `outline-installed-ini-narrow-dark-760x480.png` (32,212 bytes);
- `outline-installed-ini-far-unicode-crlf-760x480.png` (54,194 bytes);
- `plugin-manager-schema2-outline.png` (78,669 bytes);
- `result.json` with final test/vet/build/smoke fields and screenshot hashes.

The images were regenerated from final frozen source with
`LEVNOTE_SOURCE_PLUGIN_SCREENSHOT_DIR=...` and
`test ./app -run '^TestAcceptanceInstalledSourceOutlineScreenshots$' -count=1`
(PASS 1.608s), then inspected. The narrow dark tab filename/close control,
compact capability badge, editor and status remain separate and unclipped. The
far image visibly navigates to `末尾目标 🙂`. The panel shows the INI schema2
outline+configured-highlight entry and TOML schema1 configured-highlight entry;
the empty-schema2/source-only case is test-only and is not claimed as part of
that PNG.

Isolated startup/graceful-close passed in:
`native\artifacts\acceptance\startup-smoke-source-plugin-20261108-144240\`.

- window title `轻笺 · 原生版`;
- `GracefulCloseRequested: true`;
- exit code `0`;
- updater-shaped file/directory sentinels survived with exact contents;
- copied executable SHA-256 matches the release binary.

No material failure remains. Product source and final evidence are frozen.

## ACTIVE — schema2 installed source outlines (DeepSeek app/modules)

2026-11-08: I acknowledge and accept the activated source-plugin stage and the
exact neutral `sourceoutline.Scan` / `Config` /
`sourceview.NewOutline(editor.Host, Config)` contract in
`handoffs/source-plugin-contract.md`. My earlier freeze/transfer of
`core/plugins/**` and `examples/plugins/**` remains in force; GLM owns those
files for schema2. Astra owns `formats/sourceoutline/**` and
`modules/sourceview/**`.

I will change only `native/app/**`, `native/modules/modules.go` and the agreed
app/modules composition seam: a separate `Tab.OutlineFactory` (rich
`Adapter` remains nil for installed source formats), DTO mapping at the
composition boundary, generation-safe configure/plugin/reload invalidation,
and source-only UI/routing tests. Preview, split, export and rich-view paths
must reject installed source formats before dereferencing `Adapter.Create`;
schema2 without rules has no outline capability. Navigation will reuse the
accepted Flush/revalidate/resolve source flow and source reveal.

The accepted14:03 outline source/build remains frozen while this stage is
implemented. I will publish focused test results here and leave the full
test/vet/build/smoke for root's freeze review.

## FINAL frozen outline checkpoint — 2026-11-08 14:03 +08:00

Plugin/app/outline implementation and screenshots are source-frozen. This
checkpoint supersedes all earlier outline-in-progress status below. The
accepted plugin/resource implementation remains frozen; the 12:55 binary and
resource samples remain historical evidence for that earlier source.

Final source checks passed exactly once after the formatter/freeze:

```powershell
native\tools\go\bin\gofmt.exe -w native\app
native\scripts\go.ps1 test ./... -count=1 -timeout=300s
native\scripts\go.ps1 vet ./...
```

Full-suite output:

```text
app 4.751s
core/document 1.007s
core/plugins 4.841s
core/search 2.943s
core/session 3.177s
core/text 2.379s
dependency-patches 1.206s
formats/code 2.434s
formats/drawio 2.158s
formats/markdown 2.504s
formats/mindmap 2.199s
modules/graphview 3.517s
modules/markdownview 3.218s
ui/editor 3.277s
```

All packages passed. Full vet emitted no diagnostics. The dependency patch
suite is included in `./...` and passed; no separate repeat was run.

Final narrow-readonly correction:

- app tab strip and format mode row now use `Shrink(0)`, preserving the full
  `planning.mindmap` tab, its 26px close control, and all mode controls at
  760×480;
- a geometry regression checks the tab/close/mode rectangles, a >100px graph
  canvas, and the visible readonly inspector;
- Astra removed the duplicate generic host-readonly graph warning while
  retaining inspector `只读`, page-specific readonly reasons, and real
  parse/compatibility warnings;
- `outline-mindmap-readonly-760x480.png` was regenerated alone and inspected:
  one readonly notice, full tab/controls, usable canvas/inspector, no clipping.

The outline toggle is runtime-only (`ShowOutline`); it is not persisted in
session state. No persistence claim is made.

Final executable:

- Path: `F:\AIproject\LevNote\native\bin\levnote-native.exe`
- Bytes: `13,143,040`
- SHA-256:
  `D77BC3F45114957416BE83B361873AC3648708E23B308C3395A35D099FF15925`
- UTC mtime: `2026-11-08T06:02:54.6755956Z`
- Local mtime: `2026-11-08T14:02:54.6755956+08:00`
- Toolchain/configuration: `go1.27.3`, `GOOS=windows`, `GOARCH=amd64`,
  `GOAMD64=v1`, `CGO_ENABLED=1`, MyGo `v0.8.3`
- PE: machine `0x8664`, PE32+ optional magic `0x020B`, subsystem `2`
  (Windows GUI)

Exact build command:

```powershell
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

Two stale prior smoke processes held the old executable at the first link
attempt. They were verified by executable path and stopped; no unrelated
process was touched. The rebuild then succeeded.

Isolated startup/graceful-close smoke passed in:
`native\artifacts\acceptance\startup-smoke-outline-20261108-140315`.

- native window title: `轻笺 · 原生版`;
- `GracefulCloseRequested: true`;
- exit code: `0`;
- unrelated `.unrelated-data.old-999999` survived with contents
  `preserve-file`;
- unrelated `.unrelated-directory.old-999999\keep.txt` survived with contents
  `preserve-directory`;
- the smoke's `result.json` records the exact evidence.

Narrow visual/source freeze acknowledgement:

- tab and mode rows cannot flex-shrink;
- readonly graph has no app/view duplicate generic notice;
- fixture images now activate real outline targets before capture, so far and
  per-mode screenshots show the navigated content and status;
- real split hidden mindmap navigation preserves a nonzero reversed source
  selection through expansion, Undo, and Redo;
- `CurrentOutlineItem` remains a snapshot/navigation hint, not continuous
  caret tracking.

`core/plugins/**`, `app/**`, `cmd/levnote/**`, `modules/modules.go`,
`modules/codeview/**`, `formats/code/**`, `examples/plugins/**`,
`vendor/**`, and `dependency-patches/**` are frozen at this checkpoint. The
next source-plugin stage is design-only until root accepts this checkpoint and
the Astra/GLM/DeepSeek DTO and factory contracts are agreed; no next-stage
product edit has been made.

## Current outline app checkpoint — 2026-11-08 13:57 +08:00

The provider gate is green:

```powershell
native\scripts\go.ps1 test ./modules/graphview ./modules/markdownview -count=1
```

Result after Astra's hidden-navigation selection fix:
`modules/graphview` passed in 1.790s and `modules/markdownview` passed in
1.458s. This clears the app for its focused outline suite and the single final
acceptance pass after source freeze.

Completed in owned app code:

- The existing sidebar now exposes a bounded, virtualized current-tab outline
  with a runtime-only file/outline toggle, keyboard/far-row activation, and stable resolver-based
  selection, and explicit unsupported, collection-error, retry, empty,
  truncated, and loaded states.
- Markdown source collection never calls `Adapter.Create`; preview navigation
  stays on the mounted view; source and split reveal the UTF-16 block start;
  split graph navigation uses the existing graph pane and leaves the source
  caret alone.
- Captured targets are passed through `ResolveOutlineItem` after the fallible
  Flush, including old Markdown IDs. Anchor links route through
  `ResolveOutlineAnchor`. Failed/stale/readonly targets remain visible and do
  not mutate tabs, modes, source, folds, pan, zoom, or selection.
- Activation captures tab ID, adapter pointer, document path, mode, buffer
  pointer, and a monotonically increasing outline controller generation. All
  six are revalidated after Flush, snapshot refresh, resolver, and navigation;
  replacing any lifecycle field rejects the stale controller.
- Outline snapshots are bounded to 2048 items and 512-byte labels, keyed by
  tab, buffer identity, source version, and controller generation, reused
  across frames, and released on mode/tab/plugin/reload/presentation
  disposal/shutdown. Source mindmap collection parses only the bounded outline
  and does not construct a canvas layout.
- `CurrentOutlineItem` is sampled during snapshot refresh or when there is no
  stable selected row. It is a snapshot/navigation hint, not continuous
  caret-following.

Real builtin-provider regressions cover Markdown duplicate/Unicode headings,
astral text, CRLF, anchors, Compose before activation, old-ID resolution,
source/preview/split reveals, lifecycle invalidation, a 2048-item bound, and a
2050-heading partial state. Real mindmap regressions cover a hidden writable
descendant expanding root/ancestors in one undoable document transaction,
unknown metadata preservation, readonly refusal without mutation, source-only
negative-offset guidance, and split navigation preserving the existing view
and source caret. The split test now activates a real hidden descendant with a
nonzero reversed source range and verifies the exact range survives expansion,
Undo, and Redo.

Exact focused commands and results:

```powershell
native\scripts\go.ps1 test ./app -run 'Outline|Builtin(Markdown|Mindmap|Outline)' -count=1
native\scripts\go.ps1 vet ./app
native\scripts\go.ps1 test ./app -run 'TestBuiltinMindmapSplitHiddenNavigationPreservesSourceSelectionAcrossUndoRedo|TestOutlineVirtualListReachesFarRowAndActivatesByKeyboard|TestAcceptanceScreenshots' -count=1 -timeout=120s
```

The latest focused app run passed in 1.710s. Earlier focused app/vet runs were
green. App-wide test/vet, full-suite/vet/build, new binary hash, and the
isolated startup smoke are still pending for this outline source state.

Persistent acceptance fixtures and PNGs were generated with:

```powershell
$env:LEVNOTE_ACCEPTANCE_RUNTIME_DIR='F:\AIproject\LevNote\native\artifacts\acceptance\runtime-fixture'
$env:LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR='F:\AIproject\LevNote\native\artifacts\acceptance'
native\scripts\go.ps1 test .\app -run '^TestAcceptance(RuntimeFixture|Screenshots)$' -count=1 -timeout=120s
```

Result: `ok levnote.local/native/app 2.617s`. The per-mode/far screenshot setup
now activates the selected outline item before capture, so the saved images
show the target in the content pane and the success status. Inspected images
under
`native\artifacts\acceptance\`:

- `outline-markdown-preview-1180x780.png`
- `outline-markdown-source-1180x780.png`
- `outline-markdown-split-1180x780.png`
- `outline-markdown-preview-dark-1180x780.png`
- `outline-markdown-split-dark-760x480.png`
- `outline-far-heading-760x480.png`
- `outline-partial-760x480.png`
- `outline-mindmap-hidden-1180x780.png`
- `outline-mindmap-hidden-dark-1180x780.png`
- `outline-mindmap-readonly-760x480.png`
- `outline-drawio-unsupported-760x480.png`

They were regenerated after the provider's unknown-metadata warning fixture
was cleaned and inspected for clipping, selected-row visibility, status-row
overlap, dark contrast, truncation disclosure, readonly refusal, and narrow
toolbar behavior.

The accepted `13,084,160`-byte SHA-256
`8ADB375D3AE400B57AA07C73E441D4A13DACE7ADC7124F5F0F38DD457FDC98B9`
binary remains the frozen 12:55 resource checkpoint. It does not include this
outline stage and its runtime resource samples must not be relabeled as
measurements of the next binary.

## Current outline implementation — 2026-11-08 13:35 +08:00

Astra's exact, gofmt-clean contract is now present in
`ui/editor/contract.go`, including `OutlineSnapshot()` (not `Snapshot()`),
the item and anchor resolvers, `Release`, `Adapter.Outline`, optional
`NavigateOutlineItem` / `CurrentOutlineItem`, the three budget constants and
the stale/unsupported/readonly-hidden sentinels.

App product work now starts against those saved signatures. The accepted
plugin/resource implementation remains frozen. The provider implementations
are still pending from Astra, so focused app work uses purpose-built source and
navigator fakes where the app contract, lifecycle ordering and virtual list can
be tested independently of provider availability.

## Approved OutlineSource contract — 2026-11-08 13:22 +08:00

Root approved Astra's smaller additive contract and recorded the acceptance in
`handoffs/stages.md`. App implementation starts only after the exact signatures
are saved in `ui/editor/**` and `handoffs/editor.md`.

Approved shape:

- `editor.Adapter.Outline` is a source-neutral factory returning an outline
  source with observational/fallible
  `OutlineSnapshot() (OutlineSnapshot, error)`,
  `ResolveOutlineItem(id) (OutlineItem, error)`,
  `ResolveOutlineAnchor(anchor) (OutlineItem, error)`, and `Release()`.
- The mounted `View` may additionally implement
  `NavigateOutlineItem(id string) error` plus optional observational
  `CurrentOutlineItem() (id string, ok bool)`.
- `OutlineItem.Offset` is the Markdown heading block start in UTF-16,
  `Level` is 1-based, `AnchorID` is optional/empty for no anchor, and an empty
  outline `ID` is invalid.
- Every view/controller cache key includes buffer pointer identity as well as
  numeric version.
- Markdown `ID` includes its original version and source range. After Flush,
  app must call `ResolveOutlineItem(capturedOldID)` and use its returned current
  item/ID. App must not expect the old ID to equal an ID in a newly generated
  snapshot; the resolver maps through the source coordinate journal.
- Existing Markdown `#anchor` handling uses `ResolveOutlineAnchor(anchor)` and
  then the same Flush/revalidation/navigation path. The prior direct
  `OutlineProvider.GoToOutlineItem` path is compatibility only.
- Collection error, valid empty `OutlineSnapshot` (no headings), nil
  `Adapter.Outline` (unsupported), `Truncated` partial state, and readonly
  hidden-node limitations stay visibly distinguishable. `Truncated` covers
  parser limit, entry cap, and label/anchor clipping; no unbounded problem list
  is added.

App route after source lands:

1. Capture selected old item ID.
2. Call fallible `flushTab`; on failure retain tab/mode/view/draft/list state.
3. Revalidate current tab and buffer pointer/version.
4. Resolve the old item ID (or anchor), then refresh the bounded snapshot and
   cache the returned current ID/offset.
5. Navigate by the approved mode rules: Markdown source reveals the UTF-16
   block start; preview uses the view navigator; split navigates the provider
   then reveals only with a source offset. Source-only graph negative offset
   fails before navigation with "switch to reading/split" guidance. Split
   graph navigation proceeds with the source caret left alone.

Earlier status: waiting for the exact `ui/editor` source publication. That gate
is now satisfied; the current implementation status is at the top of this file.

## Latest outline contract revision — 2026-11-08 13:09 +08:00

Root resumed and accepted the resource checkpoint as frozen. The outline stage
remains contract-only in app-owned code until Astra publishes the additive
interfaces.

Blocking refinements now recorded in `handoffs/outline-contract.md`:

- `OutlineItem.ID` is a version-safe identity for one current outline item;
  `AnchorID` is the distinct deterministic Unicode Markdown slug used by
  anchor links. Duplicate headings are disambiguated only for `AnchorID`.
  Outline navigation never resolves an old clicked item by its label, ordinal,
  or anchor slug. After Flush and a version change, app recollects the bounded
  snapshot and resolves the captured identity again; absence fails visibly.
- App needs a source/preview-neutral optional controller factory on
  `editor.Adapter` (Astra owns final naming). A factory can collect/navigate
  from the current `editor.Host` without mounting a preview `View` or creating
  a heavy graph layout. Mounted `OutlineProvider` remains a compatibility
  fallback, with an explicit source-mode limitation if no light controller is
  available.
- Source/split navigation calls `Editor.Reveal(offset, offset)` for
  `Offset >= 0`; a negative offset is an actionable failure when source reveal
  is required, never coerced to zero. Preview scrolling stays with the
  controller/view. App never forces a mode switch.
- Every visible activation first calls fallible `flushTab`. Failure retains the
  tab, mode, live view/draft and list selection. Success revalidates tab
  identity, current tab and buffer version, recollects after any version
  change, and revalidates the stable target again before navigation.
- Collection stays observational and bounded. It cannot mutate the buffer,
  commit a draft, alter selection/page/pan/zoom, release caches, or build a
  heavy canvas. `Truncated` discloses omitted entries or clipped labels; full
  text remains available to tooltip/accessibility when only display text was
  clipped.

App will implement only against Astra's published final contract, then run
focused outline tests before the single final integration suite/build/screenshot
pass. The earlier accepted plugin/resource code remains frozen apart from a
concrete regression found by outline integration.

Root refinement at 13:17 is now in `outline-contract.md`:

- Bounded row text is also the bounded tooltip/accessibility text. App keeps no
  second full-label copy that would defeat retention limits; clipped labels
  show an explicit "use source/provider for full text" notice.
- Negative `Offset` does not reject graph navigation in reading or split.
  Split navigates the existing graph pane, leaves the source caret untouched,
  and states that the item has no source position. Source-only graph activation
  returns an actionable "switch to reading/split" failure with no mode, caret,
  fold, selection, or document mutation; app never forces the mode.
- Snapshot collection errors are distinct from valid empty outlines. The
  proposed controller returns `(OutlineSnapshot, error)`, and a partial
  snapshot can carry bounded `Problems`; app shows a visible problem/partial
  state instead of substituting empty/unsupported.

## Outline stage acknowledgement and contract request — 2026-11-08

Read `native/outline_stage_plan.txt`. Design and contract alignment may begin;
app implementation waits for Astra to publish the optional editor contract.
No product feature has been added in this stage.

Owned app route:

- Add one restrained current-tab outline area in the existing sidebar, with a
  visibility toggle, bounded label rendering, virtual rows, stable selection,
  keyboard/far-item activation, and an explicit empty/unsupported/partial
  state. Other formats must not appear as fabricated outlines.
- Discover capability only through optional `ui/editor` interfaces. App will
  not import Markdown/mindmap implementations or inspect concrete view types.
- Outline collection is observational: it must not mutate the buffer, commit
  a draft, release caches, or change mode.
- Activation starts with the existing fallible `flushTab` barrier. A failure
  retains the tab, mode, view and draft and reports the existing error.
- Before navigation, revalidate the tab ID/path, document buffer, provider and
  captured outline version. If the flush changed the version, recollect and
  resolve the stable item ID again. A missing/changed item fails visibly.
- For `Offset >= 0`, reveal the source range with the existing
  `Editor.Reveal(start, end)` API in source and split modes. For preview,
  delegate scrolling to the provider. Never force a mode change. `Offset < 0`
  means the format has no source position.
- Mindmap readonly hidden-node failure must report that navigation cannot
  expand folded ancestors without modifying the file. Writable navigation may
  uncollapse ancestors only through the view's normal undoable document edit.
- Tab, mode, plugin, document-version, restore/reload and sidebar changes
  invalidate the displayed outline/selection. Release only the app-side list
  state when the provider disappears; do not retain extra parsed documents.

Contract requested from Astra, preferably as additive optional interfaces:

```go
type OutlineSnapshot struct {
    Version   uint64
    Items     []OutlineItem
    Truncated bool
}

type OutlineSnapshotProvider interface {
    OutlineSnapshot() OutlineSnapshot
}

type OutlineNavigator interface {
    NavigateOutlineItem(id string) error
}

type CurrentOutlineProvider interface {
    CurrentOutlineItem() (id string, ok bool)
}
```

`Version` should be the host `text.Buffer.Version()` represented by the
cached snapshot. `OutlineItem.Offset` remains UTF-16 where meaningful and
negative where no source offset exists. `Truncated` covers bounded count or
label clipping. The existing `OutlineProvider` can remain as a compatibility
fallback, but the snapshot/navigation variants give app enough information to
report partial outlines, revalidate after Flush, and show a precise
navigation/readonly limitation without inspecting implementation types.
The navigator method needs a distinct name because a Go type cannot implement
both `GoToOutlineItem(id string) bool` and
`GoToOutlineItem(id string) error`.

For source-mode list collection, Astra's provider should avoid canvas/layout
construction; parsing required to enumerate a bounded outline is acceptable.
For Markdown, deterministic duplicate/Unicode anchor IDs and UTF-16 offsets
must be documented and shared with existing anchor-link navigation. For
mindmap, use format node IDs.

Requested from Astra:

1. Confirm or refine the three optional interfaces above.
2. Publish exact lifecycle/caching semantics in `handoffs/editor.md`.
3. Report whether current-item tracking is available and whether
   `OutlineNavigator` can return actionable readonly/missing-node errors.
4. Confirm source-mode `OutlineSnapshot` is bounded and does not construct the
   full graph canvas/layout.

App will not begin implementation until that checkpoint is visible.

## Current final app/plugin checkpoint — 2026-11-08 12:55 +08:00

Source is frozen after acceptance. No plugin, Unicode lexer, or app-owned
failure remains pending. Root accepted the corrected narrow Markdown conflict
screenshot, and the final full-suite/vet/build/startup/resource gates passed.

Completed in this ownership:

- Plugin install/export uses atomic `document.WriteNewFile` when available,
  captures the exact new failure before rollback, verifies the installed
  ID/source path directly, and leaves no stale error-index dependency.
  Registry `Manifests`/`Manifest`/`Match` results deep-copy extension slices.
- Builtin syntax plugins can be enabled and disabled safely; all valid
  installed syntax plugins sort after specific builtins but before the broad
  code fallback, independent of directory entry count.
- Plugin directory enumeration is bounded with an extra-entry probe, accurate
  truncation disclosure, and coverage at 1,025 entries.
- The Unicode lexer advances by source unit on every path and cannot loop on
  zero-width or invalid input.
- App lifecycle routes through fallible flush before save/mode/tab/export/
  close/search reveals and retains tabs/views when flush fails. The async
  ordered-open queue waits for acknowledged UI application. Presentation
  release keeps lightweight view identity/state while dropping heavy caches.
- Close/discard/recovery semantics persist clean and saved reopen records,
  remove explicitly discarded drafts durably, retain state if persistence
  fails, and preserve `OriginalPath` without turning a recovered copy into a
  disk-backed save. Ordinary Save retains mode and caret.
- Readonly Save As updates the new tab's writable state; encoding choice and
  unknown/binary labels are honest; bounded tree/search lists preserve
  keyboard activation, far-row selection, row virtualization, expansion
  identity, accurate 256-per-directory / 3000-total disclosure, and bounded
  full issue details.

Evidence:

- Full test/vet/dependency commands, final binary bytes/hash/time, build
  command, isolated startup/graceful-close smoke, screenshot command, accepted
  images, and limitations are in `handoffs/integration.md`.
- Main-process runtime working-set/private-byte/CPU measurements are in
  `handoffs/resources.md`.
- The transferred bounded search/disk checkpoint is in
  `handoffs/search.md`.

Final binary:
`13,084,160` bytes, SHA-256
`8ADB375D3AE400B57AA07C73E441D4A13DACE7ADC7124F5F0F38DD457FDC98B9`,
timestamp `2026-11-08T04:55:06.1079484Z`
(`2026-11-08T12:55:06.1079484+08:00`).

Limitations: the native tester path proves Compose/preedit transport but not a
real OS IME candidate window or physical language switching; no runtime
networking feature was exercised with packet capture; race testing is
unavailable on the bundled Windows toolchain (`0xc0000139`) and is not
claimed; resource figures are OS measurements for the main process, not Go
allocation counters or a Tauri-equivalent benchmark.

## Current resource-stage status — 2026-11-08 12:26 +08:00

No new plugin/lexer/app-owned failure is pending. The app-focused resource
checkpoint remains green, and the current acceptance fixture now has an
opt-in persistent four-document workspace generator shared with screenshot
evidence. Actual Markdown, mindmap, and draw.io Compose-to-Save/mode/tab routes
all pass.

The current app source screenshots were regenerated after the final far-row
selection edit and visually inspected. The 11:39 binary/full-suite checkpoint
below is historical: current full-suite, vet-all, dependency checks, build,
hash, isolated startup, and runtime resource sampling are intentionally
deferred until Astra's final view checkpoint and GLM's actual bounded-search
checkpoint are green. Details and exact commands are in `integration.md` and
`resources.md`.

## Final app checkpoint — 2026-11-08 11:39 +08:00

Completed and verified:

- Full suite passes:
  `native\scripts\go.ps1 test ./... -count=1`.
- Full vet is clean: `native\scripts\go.ps1 vet ./...`.
- Final GUI build succeeded with the same vendored/pinned dependency graph.
- Final executable is 12,973,568 bytes, SHA-256
  `CB388D46451195437CCC611932C627017B698D58070F7B9C23E4E3F7B50D9DE8`.
  It is x64 Windows GUI (`machine 0x8664`, subsystem 2), measured at
  `2026-11-08T03:38:32.7613215Z`.
- The final binary passed the isolated executable-directory smoke: native
  window title `轻笺 · 原生版`, unrelated `.*.old-*` file/directory fixtures
  survived, graceful close was accepted, and exit code was 0.
- Local-reference Stage A passes pure and real `Host.OpenReference` tests:
  saved relative Unicode/spaces, unsaved rejection, external URL rejection,
  remote file URL rejection, encoded UNC rejection, one-pass file URL decode,
  drive-relative rejection, and anchor routing after flush.
- New real-adapter app tests pass for pending native
  `Compose("中文🙂", 3)` through Save, mode switch and tab switch in both the
  Markdown heading editor and graph node editor. Save is read back from disk;
  ordinary Save keeps preview mode.
- Acceptance screenshots regenerate from current source and were inspected at
  1180×780 and 760×480. Current source state's narrow graph/Settings/encoding/
  plugin layouts and dark tab/source syntax are accepted visually.

Exact focused command:

```powershell
native\scripts\go.ps1 test ./app -run 'TestResolveLocalReferenceClassification|TestEditorHostOpenReference|TestEditorHostAnchor|TestRealMarkdownPendingIMECompositionAcrossSaveModeAndTab|TestRealGraphPendingIMECompositionAcrossSaveModeAndTab' -count=1 -v
```

Current external follow-up: GLM should document `native/vendor/**` and
`native/dependency-patches/**` in native notices/licensing material. Race
testing is unavailable (`0xc0000139`) and is not claimed.

## 2026-11-08 — acknowledgement

- Owner: DeepSeek plugins/app agent.
- Ownership accepted: `native/core/plugins/**`, `native/app/**`,
  `native/cmd/levnote/**`, `native/modules/modules.go`,
  `native/modules/codeview/**`, `native/formats/code/**`,
  `native/examples/plugins/**`.
- Read `native/AGENTS.md`, `native/plugins_finish_task.txt`, and
  `native/handoffs/plan.md`.
- No child agents, commits, pushes, executable plugins, scripts, dynamic
  libraries, or runtime networking. Will compact before 200K context.
- Working order: (1) safe plugin management and Unicode lexer with targeted
  tests; (2) application flush/save/close/recovery/encoding/readonly/search
  integration; (3) acceptance/build/screenshots when contracts are available.

## API/contract requests

- Editor: optional `Flusher`, `FlushView(View)`, `Host.RecoverDraft`, and
  `OutlineProvider` as specified in `native/handoffs/plan.md`.
- Formats/session: `document.WriteNewFile(path, data)` for exclusive atomic
  export publish; fallible draft capture that rejects the 4 MiB limit before
  materializing oversized contents.
- Formats/graph: adapter IDs `mindmap` and `drawio`, with `NewDocument`
  templates exposed by those adapters.

Progress, API mismatches, and final test evidence will be appended here.

## Progress

- Targeted plugin/lexer tests pass after fixing registry slice copies, exact
  post-install ID/path verification, and Unicode lexer progress.
  Exact command:
  `native\scripts\go.ps1 test ./core/plugins ./formats/code`.
- App lifecycle/readonly/encoding/ordered-open work is in progress.
- Current cross-owner build blocker: `modules/graphview/edit.go` uses
  `strings` without importing it (observed during formats-agent edits).
- Remaining formats APIs requested: `document.WriteNewFile` and fallible,
  pre-materialization `session.Capture`; session preferences should include the
  line-number flag. The temporary separate UI preference file was removed at
  root request.

## 2026-11-08 — contract migration and current blocker

- App now consumes `document.WriteNewFile`, `session.CaptureBounded`, and
  `session.Preferences.LineNumbers *bool` (nil keeps the first-run default).
- Blocking contract defect in GLM-owned `core/session/session.go`:
  `CaptureBounded` calls `validateTab(t)` before assigning `t.Draft` for a dirty
  new document. Because a new document has `Path == ""`, validation reports
  "tab has neither a file nor a draft" and recovery of every unsaved new tab
  fails. Validate after attaching the draft, or validate path/selection fields
  separately before materializing it.
- Current app failures caused by that defect:
  `TestSessionRestoresUnsavedAsCopy` and
  `TestCloseDoesNotDiscardTabWhenRemainingDraftIsOversized`.
- `TestOpenManyWaitsForDelayedAsyncUIDispatch` had a harness race: the public
  async `OpenMany` returns before its ordered worker completes. App tests will
  drive an internal completion channel while production keeps the same async
  public surface.

## 2026-11-08 — Stage A checkpoint

Completed in owned files:

- Plugin install uses `document.WriteNewFile`, validates the exact installed
  ID/source path/syntax, preserves the concrete failure before cleanup, and
  reports a rollback-removal failure instead of swallowing it. Install-limit,
  collision, preexisting-error, exclusive destination, builtin disable/enable,
  failed-settings persistence, manifest bounds, symlink/type checks, and all
  returned extension/syntax slice copy tests pass.
- Declarative and native lexers decode Unicode before classification. Every
  branch advances, zero-length spans are discarded, and an explicit
  newline-comment-marker regression test proves forward progress and lossless
  span reconstruction in addition to Unicode identifier and malformed UTF-8
  coverage.
- App now uses `session.CaptureBounded` and persisted
  `Preferences.LineNumbers *bool` (nil keeps the first-run true default).
- `OpenMany`, file drops, and CLI arguments use ordered batches. Opening waits
  for each asynchronous `Window.Update` callback; a deliberately delayed
  dispatch test covers `[first, duplicate-first, missing, second]`.
- `addDocument`, tab selection, reveal, save, reload, export, mode changes,
  plugin changes, close, window close, local find/replace, and workspace
  result reveal use the fallible rich-view flush barrier. Failed flushes retain
  the view, document, mode and selection. Replacing text releases a stale rich
  presentation only after an actual mutation.
- Ordinary save preserves mode, presentation, adapter and selection. Opening
  another document releases the prior inactive presentation. Plain readonly
  text keeps preview support and Save As. The byte inspector remains inert and
  cannot write its hex presentation over the source.
- Unknown encodings open an explicit chooser for UTF-8, UTF-16LE, UTF-16BE,
  GBK and GB18030; reopening changes the same tab only after the selected
  decoder succeeds.
- Close/window-close gathers decisions before removal. It builds and persists
  the prospective remaining-tab snapshot before removing anything. A draft
  over 4 MiB or a recovery write failure cancels close, leaves every tab in
  memory, and leaves the previous recovery snapshot untouched. Successful
  close removes explicitly discarded drafts from the persisted snapshot, with
  tests for both one-tab discard and a retained second tab.
- Export flushes first, never releases the live view, and publishes through
  exclusive atomic `document.WriteNewFile`.

Exact passing commands:

```powershell
native\scripts\go.ps1 test ./app ./core/plugins ./formats/code -count=1
native\scripts\go.ps1 vet ./app ./core/plugins ./formats/code ./cmd/levnote
```

Result: all three packages passed; vet produced no diagnostics. The focused
test file includes real delayed dispatch rather than the former `Window == nil`
shortcut.

Race runner is unavailable in this bundled Windows toolchain: even the
`./core/plugins` test binary exits during initialization with
`0xc0000139`; this is not recorded as a passing race check.

Current full-suite failure outside this ownership:

```text
modules/markdownview:
TestDraftFlushSavesIMEAndReleaseRetainsConflict
TestDraftConflictRecoveryAndDiscardAreExplicit/recover
TestDraftConflictRecoveryAndDiscardAreExplicit/discard
TestFlushReportsSplitEditWithoutSilentlyApplying
```

All four stop at the shared UI helper message `double click did not begin
block draft`, before their lifecycle assertions. App, document, session,
search, text, code, drawio, markdown parser, mindmap parser, graphview and
editor packages passed on the same run. Remaining work is full suite/vet/build
and source/Markdown/mindmap/drawio light/dark screenshots after the
markdownview checkpoint turns green.

## Acknowledgement — 2026-11-08 11:13:12 +08:00
DeepSeek plugin/app agent resumed. I will consume the published editor/session contracts, validate the latest parser state without editing GLM-owned files, rerun focused tests/vet, regenerate and inspect acceptance screenshots, then complete the full test/vet/build record once Astra and GLM checkpoints are green. I will keep changes and handoff writes within my ownership.

## Resumed acceptance checkpoint — 2026-11-08 11:23 +08:00

Latest owned fixes:

- Dark active tab now uses `Theme().Text`, giving a readable light label on
  the selected blue surface.
- The plugin manager lays out the tab strip first and replaces only the
  document canvas while open. The active filename remains visible; closing the
  panel restores the editor canvas. Minimum-width tool and extension metadata
  controls are bounded and tested.
- The unknown/binary byte inspector reports
  `原始编码未知（字节视图）` or `二进制（字节视图）`; its main warning states
  the user consequence, and decoder diagnostics live in the Issues details.
- The settings panel uses a separate header/action row and wrapped preference
  controls, eliminating the clipped `关闭设置` action at 760×480.
- Acceptance fixtures now use `t.TempDir()` and do not pollute the screenshot
  directory or `go test ./...`.

Passing commands:

```powershell
native\scripts\go.ps1 test ./app ./core/plugins ./formats/code -count=1
native\scripts\go.ps1 vet ./app ./core/plugins ./formats/code ./cmd/levnote
native\scripts\go.ps1 test ./app -run 'IME|Compose|Flush|Mode|Tab' -count=1
native\scripts\go.ps1 test ./modules/markdownview -count=1
[Environment]::SetEnvironmentVariable('LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR','F:\AIproject\LevNote\native\artifacts\acceptance','Process')
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestAcceptanceScreenshots','-count=1')
```

The current PNGs were generated after these fixes and visually inspected.
Current blockers are outside this ownership: the first full-suite run after
Astra's checkpoint hit GLM's session file while its bounded-encoding helper
edit was incomplete. Full `go test ./...`, full vet, build and final evidence
will be repeated after that file stabilizes. The MyGo startup cleanup audit is
recorded in `integration.md`; no module-cache mutation was made.
