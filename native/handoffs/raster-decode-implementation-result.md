# Raster decode implementation result

FROZEN2026-11-10 for root review. The implementation is limited to
`native/formats/raster/**`, this handoff chain, and
`native/handoffs/raster-decode-evidence-20261110/`. No application, document,
session, vendor, script, product build, host, or resource work was touched.

## API

```go
func Decode(ctx context.Context, r io.ReaderAt, declaredSize int64) (DecodeResult, error)

type DecodeResult struct {
    Kind         Kind
    Width        int   // final width after EXIF orientation
    Height       int   // final height after EXIF orientation
    SourceWidth  int   // codec width before orientation
    SourceHeight int   // codec height before orientation
    Orientation  uint8 // 0 when the file carries no orientation tag
    Pixels       *image.RGBA
}

type DecodeStage uint8 // setup, read, codec, decoded bounds, canceled

type DecodeError struct {
    Stage  DecodeStage
    Detail string
    Err    error // reader, codec, or context error; supports errors.Is/As
}
```

`Decode` always runs `Admit` itself on the caller's reader; no caller-supplied
admission estimate is accepted. Admission rejections pass through unchanged as
`*RejectionError`, so the host can distinguish "outside the admitted subset"
from "admitted but failed to decode". Decode failures are `*DecodeError` with a
stable stage. Cancellation and every failure return a zero result with nil
pixels; nothing is partially published.

## Contract and ownership

The caller supplies one stable, immutable `io.ReaderAt` and its exact declared
size; the host uses `BinarySnapshot`. `Decode`:

- reads only through a context-aware bounded `io.SectionReader` over that same
  reader, with no full-input clone, no `[]byte` input API, no filesystem, no
  network, no background goroutine, and no generic `image.Decode` registry;
- calls the explicit `image/png.Decode` or `image/jpeg.Decode` chosen by the
  admitted kind;
- revalidates the decoded bounds against admission and the hard limits
  (`1..8192` per side, at most `8,388,608` pixels, matching pixel count,
  orientation within `0..8`) before allocating the final surface, and rejects a
  decoded orientation outside that range instead of silently treating it as
  identity;
- writes straight into one `image.NewRGBA` surface with origin `(0,0)`,
  stride `4*width`, and exactly `4*width*height` bytes. Components come from
  `color.Color.RGBA()`, which is already premultiplied, so the high bytes give
  valid premultiplied `image.RGBA` pixels for `ui.NewBitmap` without another
  full-size conversion;
- applies EXIF orientations 1..8 in the same pass, with these forward
  source-to-destination mappings for a `w x h` source:

```text
1 (x, y)          5 (y, x)              output h x w
2 (w-1-x, y)      6 (h-1-y, x)          output h x w
3 (w-1-x, h-1-y)  7 (h-1-y, w-1-x)      output h x w
4 (x, h-1-y)      8 (y, w-1-x)          output h x w
```

No temporary full-size oriented image is allocated. The result retains no
reader, closer, or source bytes; mutating the source afterwards cannot change
the returned pixels.

## Cancellation and failure timing

Context cancellation is checked before every actual read (both during admission
and during the codec), before and after codec execution, at the top of every
conversion row, and once more before publication. A canceled call returns
`*DecodeError{Stage: DecodeStageCanceled}` wrapping `context.Canceled`, with nil
pixels. A reader error during the codec is reported as `DecodeStageRead` with
the original error wrapped; a corrupt or truncated but admitted payload is
`DecodeStageCodec`; geometry problems are `DecodeStageBounds`.

Limit: codec CPU between reads cannot be preempted. Go's PNG/JPEG decoders read
through buffered internal readers, so a cancellation can only take effect at
the next guarded read (or the next conversion row). The call still returns an
explicit cancellation and publishes nothing; it does not bound worst-case
latency of an in-flight codec call.

## Fallback behavior

There is no fallback path inside this package: unsupported subsets are rejected
by admission, and admitted-but-corrupt bytes fail with an explicit decode error
while the caller keeps the original snapshot for byte-level inspection and the
visible error. There is no silent orientation fallback, no silent downscale, no
silent first-frame selection, and no generic codec that could accept a format
outside the admitted PNG/JPEG subset.

## Tests

`decode_test.go` adds 10 top-level tests and 30 subtests; the existing 14
admission tests and 41 subtests are unchanged (file hash matches the accepted
admission freeze). Coverage:

- exact premultiplied pixels for PNG grayscale, truecolor, truecolor-alpha, and
  indexed palettes with alpha, with expected bytes computed from the documented
  NRGBA-to-premultiplied formula rather than from the decoder;
- a hand-written asymmetric 3x2 color grid (`abc/def`) with `eXIf` inserted
  after IDAT for orientations 1..8, including swapped `2x3` dimensions for
  5..8 and the orientation recorded in admission;
- `jpeg.Encode`-produced baseline grayscale and three-component fixtures with
  known quadrant regions and lossy tolerances, plus EXIF orientation 6 on a
  32x16 two-color JPEG (rotated dimensions and region colors);
- PNG with a corrupted IDAT Adler-32 and PNG with a corrupted IDAT CRC that both
  pass header admission and fail the codec;
- a JPEG truncated after a complete SOS header that passes admission and fails
  the codec/read stage;
- ReaderAt failure before admission (`*RejectionError`) and during the codec
  (`DecodeStageRead` with the injected error wrapped), plus admission running
  inside `Decode` (bad signature, nil reader, oversized declared size, nil
  context);
- cancellation before any read, mid-read, at a conversion-row boundary,
  during the final row (publication check), and immediately after the codec,
  with pixel counts proving the row-boundary and publication checks stop the
  work where intended;
- forged/seam geometry rejection before any `At` call: dimension mismatch,
  oversize side, pixel cap, empty surface, disagreeing pixel count, and an
  out-of-range orientation;
- structural (`reflect`: no reader/closer field) and behavioral (source bytes
  overwritten after return) proof that no input is retained.

Minimal `minimalJPEG` fixtures remain header-only admission fixtures, as now
stated in `raster-admission-result.md`; only `jpeg.Encode` output is treated as
a complete JPEG. The `8192x1151` wording there now says it passes only the
private `mipBudget` and is rejected by public `Admit` for `MaxPixels`.

## Verification

All Go commands used `native/scripts/go.ps1 -GoArguments @(...)` from the repo
root. Exact command text, separate stdout/stderr, and exits are preserved in
`native/handoffs/raster-decode-evidence-20261110/`:

| Run | Command | Exit | Result |
|---|---|---:|---|
| 01 | `fmt ./formats/raster` | 0 | rewrote `decode_test.go` formatting only |
| 02 | `test ./formats/raster -count=1 -v -run TestDecode` | 0 | focused pass, 10 top-level/30 subtests |
| 03 | `fmt ./formats/raster` | 0 | final formatting, no output |
| 04 | `test ./formats/raster -count=1 -v -run TestDecode` | 0 | focused pass after orientation hard-limit addition |
| 05 | `test ./formats/raster -count=1 -v` | 0 | full affected suite: 24 top-level/71 subtests, `ok ... 0.854s` |
| 06 | `vet ./formats/raster` | 0 | empty output |

No verification failure occurred. Run 01 is the only run that changed a file
(gofmt on `decode_test.go`); run 03 confirms a clean tree afterwards.

Admission frozen hashes in `07-source-sha256.txt` still match the accepted
values (`admission.go` `608B178F...`, `png.go` `125A0114...`, `jpeg.go`
`60DF8720...`, `budget.go` `14180CA7...`, `exif.go` `23FB7FD0...`,
`admission_test.go` `AC9788F1...`), so the frozen preflight was not modified.

## Frozen files

SHA256 and byte size, also saved as evidence file `07-source-sha256.txt`:

```text
608B178F47CDFB7A85779E4BD000C84FE130B9825B4D131BD7AFDC2269BB16DC  admission.go       5308
125A01145EB3A322746579921D39899C3F8224458A1A166A72C22DEE73AD64B3  png.go             9183
60DF8720E322CAD91618B76F5A15D5A566F77246D63B8FB878D500CCDCF686ED  jpeg.go           10819
14180CA7F300815E30FA21DE64334F3F07170B704B5608B5A60A1286623F9C50  budget.go          1900
23FB7FD054868F18150C1636ABA28558AF31C7D8717EFC502D1FE96B16DF8709  exif.go            2574
AC9788F1261ED42919A958E42204C46F466F778D35A58BA8F7840D7540C14C80  admission_test.go 27400
322150A187715FF2A769BE410520B6CB5229E9EE2D03FAA7F0FEFF0500963060  decode.go         11068
859B9C7633674666D4F5400675CC942EBC63B130389ED3F0FAFC0F5A18C4CEFD  decode_test.go    22663
```

`native/` is untracked, so an empty Git diff cannot establish isolation. The
files this task created or changed are `native/formats/raster/decode.go`,
`native/formats/raster/decode_test.go`, the corrected
`raster-admission-result.md`, `raster-decode-checkpoint.md`, this result, and
the decode evidence folder; the other raster sources and admission tests are
byte-identical to the accepted freeze.

## Limits

- This is a pure decode slice. No host, UI, bitmap upload, worker, package,
  resource measurement, or product build was run; the next owner wires
  `DecodeResult.Pixels` into `ui.NewBitmap`.
- Codec CPU between guarded reads is not preemptible; cancellation latency is
  bounded only by the codec call itself.
- Output is 8-bit premultiplied RGBA. The admitted subset is noninterlaced
  at-most-8-bit PNG and 8-bit baseline SOF0 JPEG with one or three components;
  16-bit, interlaced, APNG, progressive/lossless JPEG, and CMYK remain
  admission rejections.
- Decode allocates the final surface plus the codec's own intermediate surface;
  admission's modeled payload bound is unchanged and is not a measured RSS
  guarantee.
- EXIF handling remains one IFD0 orientation tag; other metadata stays opaque
  and preserved in the source bytes.
- Exact pixel expectations are for lossless PNG; JPEG checks use region means
  with tolerances appropriate to Go's baseline encoder, not bit-exact output.
- No new performance or memory benchmark was run, and no physical-driver or
  GPU claim is implied.
