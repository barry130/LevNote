# Raster admission design

FROZEN2026-11-10 after implementation and focused verification. This records
the behavior of `native/formats/raster`; root acceptance remains pending.

## Scope

Add a pure, local preflight helper in `native/formats/raster`. It reads only a
caller-supplied `io.ReaderAt` and declared size, never the filesystem, and never
copies the whole input. It admits only bounded PNG and JPEG headers and derives
payload budgets. It does not decode pixels, prove RSS use, replace codec
validation, integrate a viewer, or re-encode data. A later codec must validate
the same immutable bytes; any decode failure must remain visible and allow byte
inspection.

## Public API

```go
type Admission struct {
    Kind                 Kind
    Subset               Subset
    Width, Height        int
    Pixels               int64
    Orientation          uint8 // 0 means absent; otherwise 1..8
    CompressedBytes      int64
    BaseRGBABytes        int64
    MipChainBytes        int64
    MipLevels            int
    DecodePayloadBytes   int64
}

func Admit(r io.ReaderAt, size int64) (Admission, error)
```

Rejections use `*RejectionError`, whose typed `Reason` distinguishes read
failure, truncation, unsupported format/mode/metadata, dimensions, and each
payload limit. Constants expose the exact limits below.

## Hard limits and formulas

- Compressed input: `size` in `1..16MiB` (`16 * 1024 * 1024`).
- Raw dimensions: each side in `1..8192`.
- Raw pixels: at most `8,388,608`.
- Base RGBA plus every ceil-halved mip level through `1x1`:
  `4 * sum(ceil(w/2^i) * ceil(h/2^i))`, at most `48MiB`.
- Modeled decode payload: at most `128MiB`, using checked `int64` arithmetic.

The mip loop adds level zero, then repeatedly ceil-halves both dimensions and
stops after the level whose dimensions are both one.

### PNG payload

For the admitted noninterlaced, at-most-8-bit subset:

- Decoder destination: 1 byte/pixel for grayscale and indexed; 4 bytes/pixel
  for truecolor, grayscale-alpha, truecolor-alpha, and any `tRNS` that widens
  grayscale output.
- Filter rows: `2 * (1 + ceil(bitsPerPixel * width / 8))`.
- Viewer RGBA conversion: `4 * rawPixels`.
- Orientation temporary: another `4 * rawPixels` only when parsed orientation is
  `2..8`.
- Plus the compressed input size.

This follows the pinned Go PNG decoder's concrete destination selection and its
two `cr`/`pr` row allocations.

### JPEG payload

For 8-bit baseline SOF0, grayscale or standard three-component JPEG:

- Sampling factors are limited to 1 or 2. Three-component files require equal
  Cb/Cr factors, require both luma factors to be maximal, and require one of
  the exact ratios 4:4:4, 4:4:0, 4:2:2, or 4:2:0. Luma H and V are tracked
  separately.
- MCU-padded decoder surface: grayscale is one byte per padded pixel; 4:4:4 is
  three; 4:4:0 and 4:2:2 are two; 4:2:0 is `ceil(paddedPixels * 3 / 2)`.
- Fixed stream scratch: 4096 bytes, matching the pinned decoder's byte buffer.
- Three-component conversion allowance: two `4 * rawPixels` surfaces. One
  covers either the viewer's RGBA destination or Go's Adobe/RGB conversion;
  the other covers the other simultaneously reachable surface in the Go
  decoder/future conversion pipeline.
- Grayscale conversion allowance: one `4 * rawPixels` viewer surface. The Go
  grayscale decoder itself is already covered by its padded one-byte plane.
- Orientation temporary: another `4 * rawPixels` only when parsed orientation is
  `2..8`.
- Plus the compressed input size.

These are modeled payload bounds, not RSS, driver, GPU, or physical-memory
guarantees. Fixed small codec tables are intentionally outside this variable
payload model. If a mode is not modeled, admission rejects it instead of guessing.

## Bounded parsing

- PNG checks the exact signature, requires the first chunk to be a 13-byte IHDR,
  enforces legal color/depth pairs, rejects compression/filter/interlace values,
  and walks at most 4096 total chunks through a terminal empty IEND. PLTE must
  precede IDAT and is required for indexed color; tRNS must precede IDAT, have
  the color-type-specific bounded length, and appear at most once. A non-IDAT
  chunk ends the consecutive IDAT run, so another IDAT is rejected. `eXIf` is
  accepted before IDAT or after the IDAT run, but not between IDAT chunks.
  `acTL`, `fcTL`, and `fdAT` are rejected explicitly. Unknown critical chunks,
  invalid chunk names, bytes after IEND, and non-empty IEND are rejected.
  Unknown ancillary chunks remain opaque bytes in the source and are skipped by
  checked offsets. No CRC or pixel validation occurs here.
- JPEG requires SOI, performs a strict bounded marker walk, validates the small
  SOS envelope, and stops at the scan payload so entropy bytes are never
  interpreted. SOS must contain exactly the SOF components with known unique
  selectors, baseline table selectors 0 or 1, and spectral parameters
  `0/63/0/0`. It accepts SOF0 only and rejects progressive, lossless/other SOF
  modes, four-component files, unsupported sampling, excess total sampling, and
  unknown markers with typed reasons.
- EXIF is parsed from PNG `eXIf` chunks, which begin with the TIFF header, or
  from JPEG APP1 payloads at most 64KiB after validating and removing exactly
  `Exif\0\0`. The bounded TIFF walk reads only IFD0, accepts a single
  orientation SHORT with value 1..8, and rejects malformed or ambiguous
  orientation metadata. Absence reports orientation zero.

## Files

- `native/formats/raster/admission.go`
- `native/formats/raster/png.go`
- `native/formats/raster/jpeg.go`
- `native/formats/raster/budget.go`
- `native/formats/raster/exif.go`
- `native/formats/raster/admission_test.go`

Focused package tests and vet completed on 2026-11-10. Exact commands, separate
stdout/stderr, exits, source hashes, and retained failed correction cycles are in
`native/handoffs/raster-admission-final-evidence-20261110/`. Final behavior and
limitations are recorded in `native/handoffs/raster-admission-result.md`.
