# Image-view resize assertion correction frozen

2026-11-10 for the imageview/DeepSeek owner. This correction is limited to
`modules/imageview/imageview_test.go`; product geometry, UI, app code, callbacks,
bounds, and release behavior are unchanged.

`TestManualResizePreservesCenterThroughTester` now uses a local
`geometryTolerance = 1e-6` DIP for the computed 100% zoom, both preserved image
center coordinates after expansion, and the two post-resize centered pan values.
The new Tester viewport leaves ordinary `target/current.zoom` multiplication
residue (`0.9999999999999999`); one millionth of a DIP is far below one-pixel
accuracy and does not conceal a visible geometry change. Fit/manual mode,
dimension thresholds, callback errors, and all other exact assertions remain
unchanged. The accepted product source remains
`ui.go` 5729 bytes SHA-256
`2B168A93E3D3B0140D3114CD282113B02CE4CA1944820A7FA37F9F853071FBAC`.

Test file identities:

- before: 27719 bytes SHA-256
  `E4FE9CA693ABFDCC0535E629A1845965DA64CEECB7F251FEE7088F8854A36592`
- after: 28115 bytes SHA-256
  `CD2506B146E45D6AD663D9EC6DCA8BBF5DDDC39A42AB1EA37254C908459AB021`

`native/scripts/go.ps1 -GoArguments @('fmt','./modules/imageview')` completed
with stdout/stderr empty and exit 0. Per the narrow task, no standalone test was
run. The preserved integrated `final-suite1` remains the failure record:
unchanged app/modules/plugins/session/document/raster/editor packages passed,
while only this exact floating-point assertion failed.

Astra should run the full imageview package once and then the already planned
combined vet. No captures are required for this test-only correction.
Preparation resumed after this source freeze; no product execution occurred.
