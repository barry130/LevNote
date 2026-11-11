# Explicit 100% full-suite correction result

2026-11-10. Ready for root review; image keyboard source is frozen again.

The prior combined run in
`native-autosave-evidence-20261110/04-final-test.output.txt` passed `./app` and
failed `TestOneRequestAndPublicToolbarInput`: a small fitted image stayed
`fitMode:true` after explicit 100% because `zoomToCenter` returned when zoom and
pan were already equal. Vet was not run.

The narrow correction clears fit mode on an equal-geometry explicit
`zoomToCenter` target. Both the toolbar button and `1` key route through that
path. `zoomAt` and keyboard pan no-op checks remain unchanged, so genuinely
clamped relative zoom and pan still preserve fit.

`TestExplicit100PercentFitAtOneRemainsManualAcrossResize` covers the public
button and `1` key on an admitted `400x200` image fitted at exactly 100% in a
`760x480` public Tester. It resizes to `300x240` and verifies both paths remain
in manual mode at zoom1 with a genuinely cropped image and clamped pans. Public
`0` then enters fit mode with zoom below100% and centered pan. The same focused
invocation retained the existing `TestOneRequestAndPublicToolbarInput` toolbar
assertions.

Exact command:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','^(TestExplicit100PercentFitAtOneRemainsManualAcrossResize|TestOneRequestAndPublicToolbarInput)$','-count=1')
```

Result: PASS, `1.204s`, exit0. Output and exit are preserved as
`native-image-keyboard-evidence-20261110/focused-full-suite-correction-attempt2.log`
and `.exit.txt`. The first correction attempt also passed in `1.071s`, but its
small-image resize did not distinguish fit from manual geometry; that output is
preserved as `focused-full-suite-correction-attempt1.log` and the regression was
strengthened before this final freeze.

Final owned source identities:

| File | Bytes | SHA256 |
| --- | ---: | --- |
| `native/modules/imageview/input.go` | 2963 | `76D1AB17A3F9B4DC99A153782305AB490904F547A6C7239F2A548248BB2601DE` |
| `native/modules/imageview/ui.go` | 6456 | `BEF3A43EA6A028791A623728E9899CF481A1BB46BC12D91D61AFD67FB95AA368` |
| `native/modules/imageview/imageview_test.go` | 42807 | `824645F0115D8E4DF7851696E78B33AAC941523C9C2D5929B0EB3B1019094231` |
| `native/app/binary_keyboard_test.go` | 5793 | `77856CCE4BA1FBA6E2FF2E1E6044C0149FC145948320F8E5171BFA3BF137050D` |

The normal and keyboard app files are unchanged, including the binary keyboard
test. No app, geometry, admission, raster, document, session, vendor, release or
frozen evidence source was changed.

No full suite, vet, package, host, worker, capture, performance or resource run
was performed. Astra remains the sole combined runner.
