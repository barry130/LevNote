# Native image view checkpoint

2026-11-10, before focused verification.

Implemented the neutral image contract and readonly native image view within
the assigned files:

- `ui/editor/image.go` and additive `Host.BeginImage` in `ui/editor/contract.go`
- `modules/imageview/adapter.go`
- `modules/imageview/geometry.go`
- `modules/imageview/view.go`
- `modules/imageview/ui.go`
- `modules/imageview/input.go`
- `modules/imageview/imageview_test.go`

The contract definitions were sent to root immediately after they were saved.
No app, core, modules.go, raster, vendor, scripts or product-doc file was
edited.

Known fixture corrections are complete. Tests now cover finite fit/aspect/
resize geometry, 100%, pointer-anchored wheel zoom, drag and pan clamps,
one-request activation, decode/error/source states, release and stale
cross-epoch completion, explicit Dispose failure, real EXIF-oriented JPEG
decode, real transparent palette PNG decode, and two optional synthetic
760x480 Tester captures.

Next: run one focused `modules/imageview` test, correct actual failures, then
run the final affected `modules/imageview` plus `ui/editor` suite and vet. All
commands, outputs and exits are to be retained under
`handoffs/native-image-view-evidence-20261110/`.

## Root review correction checkpoint

2026-11-10, after the known review-fixture corrections and before their focused
verification. The original implementation history and evidence remain intact.

- Fit now uses a positive finite display scale independent of the 2% manual
  zoom floor. A 1x8192 image in a 120x120 viewport fits at 72/8192, and drawing
  accepts that sub-floor scale.
- Manual resize translates pan using the old viewport center's image
  coordinate before applying the new viewport clamp. Regressions cover the
  exact 400x300 to 600x400 translation and the actual Tester resize lifecycle,
  including dimensions that become smaller than the image.
- `MaxZoom` now documents eight DIPs per image pixel (800%).
- Binary byte inspection is labeled `查看字节`, with accessible label
  `查看只读字节`; refusal prose uses `只读字节视图`.
- `imageRectangle` checks the converted `float32` X and Y positions as well as
  width and height.

Next: run one focused imageview review-regression test, generate a separately
named corrected capture pair, then run the final affected imageview/editor
suite and vet. No host, product package, worker or resource run is pending.
