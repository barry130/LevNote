# Native image presentation source accepted

ACCEPTED 2026-11-10 by root after the requested fit/resize/wording corrections.
Root read the actual geometry, UI, input, lifecycle, neutral contract and
regression tests, matched all eight reviewed source identities in the result,
and viewed the corrected light/dark 760x480 captures. Root ran no checks.

Final affected suite records modules/imageview PASS 1.309s and ui/editor PASS
2.344s, exit 0; affected vet exits 0 with empty streams. Earlier failures and
original captures remain preserved. Corrected captures match the result's
c00aa014... (light) and 37ea85ba... (dark) identities and show usable toolbar,
metadata and transparency backdrop. These are software Tester captures only.

Accepted: one host-decoded owned bitmap; mandatory result geometry validation;
fit to the full image, including positive scales below the manual 2% floor;
100% as one pixel per DIP; bounded pointer zoom and pan; center-preserving manual
resize before clamping; visible loading/failure/byte-inspection action; generation
checks preventing stale completion from creating a bitmap; synchronous UI-thread
Release cancellation, reference clearing and public Dispose. The provider owns
binary input and decode scheduling; the view owns drawing and terminal disposal.

The supported codec subset comes from accepted raster admission/decode. This
acceptance is for the unregistered view and neutral Host.BeginImage contract,
not the unfinished host route or a portable image release. There is no physical
driver/IME test or process RSS/VRAM conclusion. The accepted bitmap retirement
limits still apply. Source is frozen for the app/composition integration stage.
