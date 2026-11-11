# Native mindmap markers — real app acceptance

Status: **FINAL FOCUSED PASS / FULL MARKER ARTIFACT ACCEPTED**.
The app tests were owned by `/root/glm_marker_app`; after that owner was
interrupted, `/root/astra_marker_view` became sole integrator/final-doc owner.
Product/tests stayed frozen during full integration. The final focused command
is preserved in `artifacts/acceptance/mindmap-markers-20261108-234731-final-run/`:

```powershell
$env:LEVNOTE_MINDMAP_MARKERS_APP_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\artifacts\acceptance\mindmap-markers-20261108-234731-final-run'
native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMindmapMarkers','-count=1','-timeout','180s','-v')
# PASS; ok levnote.local/native/app 2.817s
```

The prior `234000-current-run/focused-run.log` remains FAIL2.623s. Corrections
were test-only: priority4 replaces priority3 rather than appending beside it,
and recovery actions use `AccessByRole(RoleButton,...)` for their full28DIP
bounds instead of `Find` returning the text glyph. Final tests include actual
native split-view clear, resulting icons/SVG, BOM reopen and reachable recovery.
No additional focused vet was needed; full `vet ./...` below covers it.

Root reviewed all four final760×480 images: normal light/dark, split and dark
conflict. Clear and recovery controls have full reachable bounds; all19 glyph
controls are not simultaneously visible. Narrow split squeezes the source
pane, graph labels may clip, and screenshots do not prove realOSIME/GPU.

Root then accepted the full artifact in
`artifacts/acceptance/mindmap-markers-integration-20261109-000341/`:
test9.480s/vet1.829s/build2.541s/smoke3.450s all exit0,20 package summaries,
15/15 evidence checks and154 unchanged frozen file fingerprints. Exact exe:
13,461,504bytes, SHA256
`17E014EB6F61E671F6FCCA439719E8831B7591042A6F29EE5354960C120C0466`,
mtimeUTC`2026-11-08T16:04:08.3209774Z`. Prior34F3F48C…cell exe is preserved.
See `integration.md` and separate `root-acceptance.json`; no resource sampling,
benchmark, product changes, extra focused rerun, commit or push occurred.

## Historical earlier focused checkpoint — superseded by final run above

The original app owner reported the following2.797s run and earlier vet. These
are historical, not the final focused acceptance or current integration status.

## Actual focused verification

All commands used the existing Go1.27.3 wrapper:

```powershell
$env:LEVNOTE_MINDMAP_MARKERS_APP_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\artifacts\acceptance\mindmap-markers-20261108-230124'
.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMindmapMarkers','-count=1','-timeout','180s','-v')
# ok levnote.local/native/app 2.797s ; all five top-level tests PASS, exit0

.\native\scripts\go.ps1 -GoArguments @('vet','./app')
# exit0, no diagnostics
```

The focused command ran both real 1180×780 adapter tests and the optional
760×480 image test in one invocation. It covered:

- real registered `mindmap` adapter and preview mode, with the marker section
  initially collapsed in the existing 224DIP scrolling inspector;
- actual pointer clicks on the stable native controls;
- independent expected sequences: `[1️⃣,自定义,⭐]` → click优先级1 →
  `[自定义,⭐]` → click优先级2 → `[自定义,⭐,2️⃣]` → click红色标记 →
  `[自定义,⭐,2️⃣,🟥]` → click星标标记 → `[自定义,2️⃣,🟥]`;
- exact saved pre-star source restored byte-identically by the shared app
  editor undo command, then redo; this is labeled accurately as the app
  editor command route, not a native canvas keyboard gesture;
- clear writing present-empty `[]`, save and reopen, then a real marker click
  after reopen;
- UTF-8 BOM preserved on open/save and only stripped for direct format/JSON
  assertions;
- rich source preservation after real marker edits: top-level unknown number
  spelling, node unknown object, tag text/style/className/extra ordered raw
  values, and image URL/width/height/extra raw values;
- pending note Flush before marker action; source/split/preview round trip,
  then another real marker transaction;
- readonly accessible button Disabled, visible readonly guidance and immutable
  host source/version/dirty state;
- a same-field source conflict: the real marker pointer action is blocked,
  host source/version/history/dirty state and marker icons stay unchanged,
  `节点草稿尚未写回` and `标记操作问题` are visible, both recovery controls are
  reachable, and the real `恢复为新文档` control opens an exact unsaved copy while
  the authoritative host remains unchanged.

The app layer does not repeat the view owner’s full conflict matrix. It checks
the registered shell integration, visible recovery controls and host authority
only.

## Evidence and images

The first23:01 screenshot attempt failed because `openMarkerPanel` used
1180×780 coordinates inside a760×480 tester. Its original directory was:

`native/artifacts/acceptance/mindmap-markers-20261108-230124/`

The author reused that directory for the passing rerun at23:08, overwriting
its earlier image evidence. Copying those passing bytes to the directory below
did **not** preserve the original failed images. This is a provenance limitation;
the original failed image bytes are not recoverable from those copies:

`native/artifacts/acceptance/mindmap-markers-20261108-231014/`

| File | Bytes | SHA-256 |
|---|---:|---|
| `mindmap-markers-app-light-760x480.png` | 45,071 | `63D675D2763CE04453C3DB17A8E000ECA7866F42AC5C51B66FABDAC7F6866982` |
| `mindmap-markers-app-dark-760x480.png` | 45,152 | `B86AC97D5E61D4254B6B8E0E3E5FEA9BFAB80B1C171240CBB55FDB5D4BAFE60F` |

Both images were visually inspected. The current scroll region shows the marker
palette and clear action legibly in light and dark themes. The earlier note
field and summary remain reachable by scrolling; the clear action is within the
760×480 viewport in the captured state. These are not claims that all 19 marker
controls are simultaneously visible. Graph labels remain clipped under the
existing bounded-canvas behavior.

## Ownership and limits

- Pure `Icons/SetIcons` was delivered by the format owner; its semantics and
  regressions remain in the accepted frozen product.
- Graphview's latest focused checkpoint includes the independently identified
  SVG marker omission correction: complete package4.477s and vet exit0. The
  earlier3.462s figure preceded that correction.
- Full native tests, release build and isolated hidden startup/WM_CLOSE are now
  complete and root-accepted as recorded at the top; this historical app section
  alone is not the full integration evidence.
- Tester screenshots and synthetic input do not prove real OS IME, GPU or
  production-window behavior, and no resource comparison is implied.
