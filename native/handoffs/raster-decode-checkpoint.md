# Raster decode checkpoint

2026-11-10, before focused verification. Working checkpoint, not acceptance.

`native/formats/raster/decode.go` now implements the direction from
`raster-decode-next-plan.md`:

- public `Decode(ctx, io.ReaderAt, declaredSize) (DecodeResult, error)` that
  always runs `Admit` internally and reads through one context-aware bounded
  section reader on the caller's immutable reader;
- explicit `image/png.Decode` / `image/jpeg.Decode`, no generic registry, no
  filesystem, no full-input clone, no goroutine, no retained reader;
- explicit `*RejectionError` passthrough for admission and `*DecodeError` with
  read/codec/bounds/canceled stages afterwards;
- decoded bounds rechecked against admission and the hard limits before the
  final `*image.RGBA` is allocated;
- all eight EXIF orientations applied source-to-destination straight into the
  final tightly packed origin-zero premultiplied surface, with the output
  dimensions swapped for 5..8;
- cancellation before every read, before/after the codec, between conversion
  rows, and before publication; failures and cancellation return nil pixels.

`decode_test.go` adds authored fixtures/oracles: exact premultiplied PNG
grayscale/truecolor/alpha/palette pixels; a hand-written 3x2 color grid for
orientations 1..8 with PNG `eXIf` after IDAT; `jpeg.Encode` grayscale and
three-component known regions with lossy tolerances; JPEG EXIF orientation 6;
corrupt IDAT, bad CRC, and SOS-truncated payloads that pass header admission;
ReaderAt failure before and during decode; initial, mid-read, between-row,
final-row, and post-codec cancellation; seam bounds rejection (mismatch,
oversize side, pixel cap, empty surface, disagreeing count) with `At` never
called; and structural plus behavioral proof that the result retains no reader.
Existing admission tests are untouched.

Admission result wording was corrected: `8192x1151` passes only the private
`mipBudget` and is rejected by public `Admit` for `MaxPixels`; `minimalJPEG`
fixtures are labeled header-only admission fixtures rather than valid JPEGs.

`native/scripts/go.ps1 -GoArguments @('fmt','./formats/raster')` exited 0
(evidence run 01). Focused decode tests are the next step.
