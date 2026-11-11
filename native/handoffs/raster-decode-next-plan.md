# Raster decode continuation

PLANNED2026-11-10 root; not active until admission helper is accepted. Keep
ownership in new formats/raster, no app or vendor changes. This slice gives the
host a bounded codec operation before UI integration; it can be developed in
parallel with terminal Bitmap retirement once its admission contract is stable.

Recommend a small Decode(ctx, io.ReaderAt, declaredSize) result holding metadata
and one owned *image.RGBA. Run admission internally; do not trust a caller-forged
Admission estimate. Reader is the same immutable BinarySnapshot ReaderAt used
through the entire operation. No filesystem and no full-input clone/[]byte API.
Use explicit Go image/png.Decode or image/jpeg.Decode codec primitives after
admission, not generic image.Decode/DecodeBitmap/SVG/remote support. MyGo's
DecodeBitmap would require exposing/copying compressed bytes and retains remake;
instead UI later wraps transferred RGBA with public NewBitmap on the UI thread.

Verify codec output kind/raw dimensions against admitted values before drawing,
and enforce finite/int/bounds requirements independently. Honor all eight parsed
EXIF orientations in project-authored pixel transfer; PNG eXIf included. Combine
RGBA conversion and orientation where simple, avoiding redundant full-size
surface allocation. Admission remains a conservative modeled payload bound;
update its formula only with explicit evidence if actual allocation path changes.
Use clear ownership: result pixels transferred exactly once, caller releases by
dropping; after publication caller cannot mutate while bitmap uses the backing
pixels. No implicit compressed-source retention from a remake closure.

Context-aware reader checks before each actual read and between conversion rows;
decoder CPU cannot be preempted between reads, which must be disclosed. Cancel
cannot publish pixels. Bounded decoder failure/corrupt final pixels returns
explicit error and retains source for byte fallback. No background goroutine
inside pure Decode; host owns one coalesced execution lane and publication.

Meaningful tests with independently expected asymmetric colored grids for
orientations1..8, alpha/palette/gray/truecolor PNG, baseline JPEG grayscale and
sampling, corrupt/truncated codec payload admitted by header, ReaderAt failure,
cancel before read/during conversion, dimension mismatch. PNG exact pixels,
JPEG tolerances appropriate for lossy codec; don't compare decoder to itself.
Focused package tests/vet once final corrections pass. No UI/tester/package or
performance run in this continuation. Product feature stays future until host
and native view are integrated and tested.
