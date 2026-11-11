# Image view focused source review

2026-11-10 root read-only review of the first frozen implementation. Before
acceptance, correct two geometry behaviors and their regressions. No app or
vendor scope change; no host/package/resource runs.

1. Fit currently clamps to MinZoom=.02, so an admitted tall8192 image in the
   view's120DIP minimum canvas becomes163.84DIP tall and is cropped despite
   适合窗口. A120x120 viewport with a1x8192 image proves this; MaxPixels/mip
   admission allow it. Make fit show the entire image with padding under all
   positive practical canvas sizes, allowing a smaller finite fit scale than
   the manual zoom floor if necessary. Update downstream validity/zoom-button
   behavior accordingly; don't silently label a cropped result as fit.
2. On resize in manual mode, drawViewport only clamps the old pan; it does not
   preserve the image coordinate at the old viewport center. Example zoom1,
   image1000x800, old viewport400x300 pan(-300,-250) centers image(500,400).
   Resize600x400 should become(-200,-200) before clamping, not retain old pan.
   Preserve the same center image coordinate, then clamp. Test both expandable
   and newly smaller-than-viewport dimensions through actual resize lifecycle.

Also correct MaxZoom's reversed comment: zoom8 means eight DIP per image pixel,
not eight image pixels per DIP. Button text 查看只读源码 is misleading for binary;
use 查看字节 (keep accessible label clear), and warnings should say 字节视图.

Validate finite positions after float32 conversion, not only sizes, if geometry
helpers are claimed to reject all finite-overflow cases. Keep normal host limits
and trust boundaries explicit; no need another giant generic validation layer.

Run only substantive new geometry/regression tests, then one final affected
imageview/editor suite/vet after corrections. Retain original result/evidence
history and add numbered logs. Existing captures need updating only if the
toolbar text/layout changed; two final captures maximum, don't delete originals.
