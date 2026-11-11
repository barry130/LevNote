# Native image keyboard result

2026-11-10. Second freeze after the combined-suite correction; ready for root
source review. No child delegation was used. The `.23` package and frozen
release evidence remain unchanged.

The prior combined run preserved in
`native-autosave-evidence-20261110/04-final-test.output.txt` passed `./app` but
failed `TestOneRequestAndPublicToolbarInput`: a small image fitted at exactly
100% retained `fitMode:true` after the explicit button action. Vet was not run.
The correction changes only `ui.go` and `imageview_test.go`.

## Implemented behavior

Focused image canvases expose the reviewed keyboard reading controls:

- `=`/Shift+`=` zooms in by 1.25.
- `-` zooms out by 1/1.25.
- Keypad add/subtract perform the same actions with zero modifiers.
- `0` fits and `1` returns to 100%.
- Arrow keys pan by 32 DIPs in the existing reading direction.
- Key takes precedence over Code, so a dual key/code event performs one action.
- Recognized actions are consumed even under clamps; unsupported keys,
  modifiers, text/IME, commands, key-up and pointer-drag keys are untouched.
- Keyboard zoom anchors at viewport center. A clamped arrow no-op preserves fit
  mode, including the sub-floor fit state.
- An explicit `100%` action enters manual mode even when a small image's fitted
  scale and pan already equal 100%, so later resizes no longer resume fitting.
  Relative zoom and pan keep their genuine clamped no-op behavior.

Enabled fit/100%/decrease/increase toolbar actions return focus to the canvas.
Decode completion, tab selection, plugin changes and the byte-view action do
not. Existing control labels and layout remain unchanged; key guidance is in
tooltips.

## Public Tester coverage

- `TestImageKeyboardFocusAndInputContainment` covers click/Tab focus, each key
  form, one-action key/code normalization, drag isolation, modified shortcuts,
  full viewport containment across adjacent text/IME/commands, window blur,
  toolbar Enter/Space focus return, and Tab exit.
- `TestImageKeyboardGeometryThroughPublicEvents` covers center anchoring,
  manual zoom bounds, all pan edges, finite bounded state, fit/100%, and an
  admitted `1x8192` sub-floor-fit no-op in a short valid viewport.
- `TestImageKeyboardReleasedViewHasNoRoutes` covers release, stopped drawing,
  pending focus cancellation, unchanged state, and no decode/recreation.
- `TestExplicit100PercentFitAtOneRemainsManualAcrossResize` covers both the
  public button and `1` key on an admitted `400x200` image fitted at 100%.
  Resizing the public Tester to `300x240` proves the explicit 100% view remains
  cropped and clamped at zoom 1, while public `0` then returns centered fit
  below 100%.
- `TestNativeImageKeyboardByteAndTabBoundaries` covers real app keyboard zoom
  through rendered geometry, the readonly byte editor focus boundary,
  bytes/version/token preservation, Ctrl+Tab authority transfer, ordinary text
  input isolation, Ctrl+Shift+Tab return, and restored held bytes.

Root review notes 1-4 are resolved. The app test no longer uses
`HasText("125%")`; containment compares the complete viewport; the sub-floor
fixture uses admitted geometry; and the evidence manifest records every Go
command, exit and failed attempt.

## Verification and identities

Final corrected focused runs:

- `.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','TestImageKeyboard','-count=1')`
  passed in `14.186s`, exit0.
- `.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestNativeImageKeyboardByteAndTabBoundaries$','-count=1')`
  passed in `2.409s`, exit0.
- `.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','^(TestExplicit100PercentFitAtOneRemainsManualAcrossResize|TestOneRequestAndPublicToolbarInput)$','-count=1')`
  passed in `1.204s`, exit0 after strengthening the public resize regression.

Exact earlier failures and commands are preserved in
`native-image-keyboard-evidence-20261110/command-manifest.md` and its logs.

| File | Bytes | SHA256 |
| --- | ---: | --- |
| `native/modules/imageview/input.go` | 2963 | `76D1AB17A3F9B4DC99A153782305AB490904F547A6C7239F2A548248BB2601DE` |
| `native/modules/imageview/ui.go` | 6456 | `BEF3A43EA6A028791A623728E9899CF481A1BB46BC12D91D61AFD67FB95AA368` |
| `native/modules/imageview/imageview_test.go` | 42807 | `824645F0115D8E4DF7851696E78B33AAC941523C9C2D5929B0EB3B1019094231` |
| `native/app/binary_keyboard_test.go` | 5793 | `77856CCE4BA1FBA6E2FF2E1E6044C0149FC145948320F8E5171BFA3BF137050D` |

The prior freeze identities remain preserved:
`ui.go` was 6227 bytes SHA256
`9BA8160171861C08A3655171BC6DBE8B5084AB022C3993A62473B6336EC9A494`;
`imageview_test.go` was 39849 bytes SHA256
`E019EC67C75A12D16C89EDD6E69E2B839E9D4ED05FB9C2DF29D6B36B031260BE`.
The first correction regression, superseded before final freeze, was 41682
bytes SHA256
`AD52ED575502BC895CCAB65944FCAE1481301273F0F4AE626C28A5FC08101EFA`.

`native/modules/imageview/view.go` remains at frozen SHA256
`AA1EB6A586C38579EFF6F65EF9276B9E4FE35C75FD87BFC0F89821E06943B346`.
No app runtime, contract, geometry, raster, document, session, vendor,
release/evidence, package or product-execution file was changed.

Source is frozen again pending root review. The combined app/imageview suite and
vet remain held for Astra's sole coordinated run; neither was rerun here.
