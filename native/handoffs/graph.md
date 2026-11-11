# Graph safety handoff

Ownership confirmed by root 2026-11-08 after GLM acknowledgement: `modules/graphview/**` is now Astra-owned. Read `graph_safety_task.txt`; stage A only, no optional graph features, no subagents/commits/pushes.

Plan accepted: explicit pending field drafts keyed by stable node/page identity, compare current authoritative field before mutation, retain failures/conflicts across parse refresh and lifecycle, Flush barriers, separate unsaved-copy recovery through Host.RecoverDraft, linear node traversal, focused native UI/failure tests. App will retain lightweight View pointers after Release for inactive tabs/source mode; parsed/layout caches are released.

## Stage A completed — 2026-11-08

- Graph view implements optional `editor.Flusher`. Pending node text is held in an independent small `editor.Editor`, with authoritative buffer identity, original field value, node ID, and stable draw.io page ID. Existing `ui.TextInput` was replaced with this engine after root approval because its deferred `Changed`/preedit publication could not supply a reliable save barrier. The field has its own accessible `节点文本` label, 13px text, 76px bounded inspector region, and no line numbers. `Editor.Label`/`MinHeight` are configurable; ordinary editor defaults remain unchanged. Final native commit now publishes `OnChange` even when committed text equals existing preedit.
- `Flush` finalizes native composition, compares the edited field against the current source, then applies only that mutation. Unrelated source edits and unknown metadata survive. Group ownership, parser, serialization, read-only and same-field conflict failures retain the exact draft and show persistent retry/recover actions. Draw.io page reorder follows unique stable page IDs; ambiguous idless multi-page sources reject after source changes. A removed target page is shown as unavailable, never as another page with the same cell ID.
- Export, structural commands, node/page selection and undo/redo all flush first. Source refresh does not replace pending input. `RecoverDraft` uses `Host.RecoverDraft` to open a separate `.txt` document with field/node/page context; failure retains the draft and success never writes the authoritative source. Successful recovery also releases the old graph caches.
- `Release` clears parsed models/layout/geometry and forces same-version rebuild while retaining selection/page/zoom/pan. A defensive direct release keeps a failed draft and releases its editor geometry. The host must keep the lightweight View pointer and stop a switch/release on failed `Flush`, as coordinated by root. Clean label editors and their undo buffers are discarded on release.
- Stale unused note/link/tag/icon setter scaffolding was removed; these metadata remain parsed, displayed where already supported, and preserved. No optional metadata editing UI was added. Iterative document traversal replaces recursive slice concatenation. Source refresh cancels an in-progress drag/connect/reparent operation with a report; page switches clear gestures, preventing stale node/page writes.
- Root's export findings fixed: multiline label splitting uses linear `strings.Split` slices; mindmap SVG background uses the exact numeric viewBox origin/size including negative coordinates; native and SVG mindmap paints share a length-bounded literal color parser (hex, standard named colors, transparent/none). Paint-server URLs, expressions and unsupported CSS values use a visible counted fallback while raw metadata remain untouched. Internal SVG arrow marker references remain local. Root styles now apply consistently to canvas/export.

## Validation

Passing final commands:

```powershell
native/scripts/go.ps1 test ./modules/graphview ./ui/editor ./core/text
native/scripts/go.ps1 vet ./modules/graphview ./ui/editor ./core/text
native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-run','^$','-bench','BenchmarkGraphSplitLongLabel','-benchmem','-count=1')
$env:LEVNOTE_GRAPH_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\build-cache\graph-acceptance'
native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-run','TestGraphDraftScreenshot','-count=1')
```

- Focused suite: graphview 1.221s; editor/text cached; vet exit 0. Screenshot test 1.314s. Regression coverage includes actual native `Tester.Compose`/`Type` followed immediately by flush/save barrier, selection, export or release; exact single-step undo; foreign Buffer group, invalid XML text, parser failure, deleted node/page, page reorder, same-field conflict, recovery callback failure/success, readonly documents/compressed pages, unchanged bytes through lifecycle/export, unknown image/style metadata, stale native drag, external SVG paints, negative background bounds and long Unicode multiline export.
- Split benchmark (Windows/amd64, Ryzen 7 6800H): 64 KiB line **1,770 ns/op, 32 B/op, 1 alloc/op**; 1 MiB line **28,540 ns/op, 32 B/op, 1 alloc/op**. These measure splitting only, not full graph rendering or process memory.
- Inspected `native/build-cache/graph-acceptance/graph-draft-conflict.png` (960×640). Authoritative `外部修改` stays on canvas while local `正在输入🙂` remains in the inspector; conflict text, retry and separate recovery actions are visible without clipping/overlap.

Limits: graph PNG export remains explicitly unsupported; graph images are preserved placeholders and no file/network image load occurs; compressed draw.io pages remain read-only. Only existing node text/structure editing is provided. CSS rgb()/hsl()/border shorthands are preserved with an explicit fallback, not interpreted. App-level complete save/tab/mode integration is the app owner's acceptance responsibility; tests here verify the actual native input path and optional flush contract. No commits/pushes or edits to other owners' format engines/app/session.

## Narrow-window acceptance follow-up

Root reported cropped zoom controls and vertically crushed page names at the app's 760px minimum (graph content width 532px). Toolbar now wraps at natural action widths, keeps zoom controls together, and keeps previous/name/next together with a single-line ellipsis for page names. No global/body font reduction. `TestNarrowGraphToolbarRemainsVisible` at 532×380 checks all zoom controls lie within bounds, page title height stays one line, and actual zoom click works. Generated/inspected `build-cache/graph-acceptance/mindmap-narrow.png` and `drawio-narrow.png`: toolbar occupies two rows, inspector is 224px, remaining canvas stays usable. DeepSeek notified to recapture full app 760×480 images. Current final graph/Markdown/editor/text/app suite and vet pass; see `markdown-view.md` for exact final times.
