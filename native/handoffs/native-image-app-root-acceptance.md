# Native image integration source accepted

2026-11-10 root read the final app implementation, binary lane/actions/state
and focused regressions, independent review, final command/output/exit records
and the corrected integrated capture. All15 app files match the frozen source
inventory; root ran no tests/builds/host processes.

Accepted behavior: exclusive binary/text authority; pre-text signature/suffix
routing; active verified snapshots, readonly image and bounded byte views;
original-byte exclusive copy and explicit transactional reload; plugin and text
capability guards; sequential mixed opens and binary-aware mixed-session restore.
Late duplicates reuse existing tabs through Flush. Clean legacy image records
route before text loading; dirty image-suffix text drafts remain recoverable text.

Root notes1–7 are resolved. Ready-result delivery checks owner/token/generation/
mode/plugin; one execution lane prevents overlapping codec operations, awaiting
results block further full work. Dialogs defer publication; close during a dialog
does not consume a ready copy. Pending explicit operations report their actual
outcome before a later close. Dispatch timeout drops payloads and frees the
reservation on the next UI frame/command; Shutdown joins/clears. Same-tab mode
or plugin changes retain accepted source while disposing the bitmap; actual tab
deactivation drops both. Failed Flush/reload retains the previous authority.

Verification accepted as combined evidence, not an invented all-green run:
- final-suite1 overall exit1 is preserved. App PASS19.490s, and modules,
  graphview/markdownview/sourceview, plugins/session/document/raster/editor pass.
  Only imageview's exact computed zoom assertion failed from floating residue.
- Root reviewed the test-only correction: local1e-6 tolerance for computed zoom/
  pan/center values, all behavioral/mode/dimension checks retained. Runtime view
  geometry unchanged. Final imageview_test.go28115B SHA256
  CD2506B146E45D6AD663D9EC6DCA8BBF5DDDC39A42AB1EA37254C908459AB021.
- imageview-final1 full package PASS1.203s/exit0; planned combined vet exit0,
  empty output. No repeated passing package tests or captures.
- Synthetic760x480 corrected image capture30620B SHA256
  9CC430CDE23FF506F73B9B11A58CA917E53FC4DD2DD45490F2AAEC86239BD8DD
  visually inspected by root: fit/zoom/byte controls complete. Prior image and
  dark byte captures preserved. Narrow532/300 pane evidence accepted separately.

Frozen app inventory2524B SHA256
B744BB2621FBFA66820E7539244BD85617CBCA40E74AEE4BD2CF499DCEE78E81.
Final result9853B SHA256
FFA6955A15A944E1F76EDFA03F30879C4A6379DE1E2A5A24B645CFA018FEB315.
Evidence: native-image-app-evidence-20261110. Raw prior capture/launcher failures
and incorrect early exit metadata are disclosed and preserved.

Limits: readonly supported PNG/JPEG subset,16MiB source,8192side,8,388,608pixels;
oversize byte prefix64KiB without this path's copy. Startup restoration performs
sequential synchronous reads. Steady state one active source/bitmap; explicit
open/reload and a native copy dialog can temporarily retain old accepted source
alongside replacement work. Canceled codecs may finish between guarded reads.
These are ownership/model limits, not RSS/VRAM/driver or physical-input evidence.
Licensed toolkit/codecs remain dependencies, with no copyright-immunity claim.

Source is frozen for .23. Delivery preparation may bind this acceptance;
actual packaging and two hidden host runs require separate root activation.
Current accepted runnable package remains immutable .22 until that completes.
