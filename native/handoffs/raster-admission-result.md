# Raster admission result

FROZEN2026-11-10 for root review. The implementation is limited to
`native/formats/raster/**` and the named raster handoff/evidence files.

## Outcome

`native/formats/raster` is a pure PNG/JPEG header preflight. It accepts an
`io.ReaderAt` and declared size, copies no IDAT payload, performs no filesystem
or network access, validates a bounded subset, parses bounded orientation
metadata, and returns small dimensions plus modeled payload budgets.

```go
func Admit(r io.ReaderAt, size int64) (Admission, error)

type Admission struct {
    Kind               Kind
    Subset             Subset
    Width, Height      int
    Pixels             int64
    Orientation        uint8
    CompressedBytes    int64
    BaseRGBABytes      int64
    MipChainBytes      int64
    MipLevels          int
    DecodePayloadBytes int64
}
```

Failures are explicit `*RejectionError` values with one of `RejectionRead`,
`RejectionTruncated`, `RejectionInvalid`, `RejectionUnsupported`,
`RejectionMetadata`, `RejectionDimensions`, `RejectionCompressed`,
`RejectionMipBudget`, or `RejectionDecodeBudget`.

## Limits and budgets

The exact limits are:

- Declared compressed size: `1..16,777,216` bytes.
- Each side: `1..8192`; raw pixels at most `8,388,608`.
- Base RGBA plus every ceil-halved level through `1x1`:
  `4 * sum(ceil(width/2^i) * ceil(height/2^i)) <= 50,331,648` bytes.
- Modeled decode payload at most `134,217,728` bytes.
- All arithmetic on dimensions, lengths, and payloads is checked `int64`.

The mip count includes level zero and the final `1x1` level. For example,
`8192x1151` is 50,299,036 bytes over 14 levels and passes the private mip
formula, but public `Admit` rejects it because its 9,428,992 raw pixels exceed
the `8,388,608` `MaxPixels` limit; `8192x1152` also exceeds the 48MiB cap. That
example pins only the private mip arithmetic, not an admitted public size.

PNG destination bytes per pixel are one for grayscale and indexed images, and
four for truecolor, grayscale-alpha, truecolor-alpha, and grayscale/truecolor
widened by tRNS. Its modeled payload is:

```text
compressed
+ destinationBytesPerPixel * pixels
+ 2 * (1 + ceil(bitsPerPixel * width / 8))
+ 4 * pixels
+ 4 * pixels when orientation is 2..8
```

JPEG sampling tracks luma and chroma H/V independently. Luma must be maximal;
the two chroma components must be equal; the admitted ratios are 4:4:4, 4:4:0,
4:2:2, and 4:2:0. Exact padded planes are:

```text
cols = ceil(width / (8*lumaH))
rows = ceil(height / (8*lumaV))
Y = (cols*8*lumaH) * (rows*8*lumaV)
Cb = Cr = (cols*8*chromaH) * (rows*8*chromaV)
decoderPlanes = Y                 for grayscale
decoderPlanes = Y + Cb + Cr       for three components
```

One 16x16 4:2:0 MCU therefore costs `256 + 64 + 64 = 384` bytes. A 17x17
4:2:2 image pads Y to 32x24 and charges `768 + 2*384 = 1536`; a 17x17 4:4:0
image also charges 1536 through different padded geometry. JPEG payload is:

```text
compressed + decoderPlanes + 4096 stream scratch
+ 4 * pixels viewer surface
+ another 4 * pixels for a three-component conversion surface
+ 4 * pixels when orientation is 2..8
```

Grayscale charges only the viewer surface. Three-component images retain the
second surface for either Adobe/RGB conversion or the simultaneously reachable
viewer/conversion surfaces documented in the design.

## Bounded parsing

PNG requires the exact signature and a 13-byte IHDR first, with legal at-most-8
bit color/depth pairs, noninterlaced geometry, and methods 0/0. It walks at most
4096 total chunks. PLTE precedes IDAT and is required for indexed color. tRNS
precedes IDAT, appears once, and has the bounded color-specific length. IDAT
chunks must remain consecutive. eXIf is accepted before IDAT or after the IDAT
run, but not between IDAT chunks. APNG `acTL`/`fcTL`/`fdAT`, unknown critical
chunks, invalid chunk names, non-empty IEND, and bytes after IEND are rejected.
Ancillary data other than bounded eXIf remains opaque and unread.

JPEG accepts SOI, a bounded marker walk to SOS, SOF0 only, 8-bit grayscale or
three components, oracle-supported sampling, and legal 0xff fill bytes. The SOS
must name exactly the SOF components with unique selectors, baseline table
selectors 0/1, and spectral parameters `0/63/0/0`. The walk stops at SOS and
never interprets entropy bytes as markers. APP1 EXIF removes exactly the
six-byte `Exif\0\0` prefix; PNG eXIf begins directly with TIFF. IFD0 parsing is
relative to the bounded TIFF slice, checks every offset/count, accepts one
SHORT orientation value 1..8, and rejects duplicates or malformed metadata.

## Verification

All Go commands used `native/scripts/go.ps1 -GoArguments @(...)`. Exact argument
text, separate stdout/stderr, and exits are under
`native/handoffs/raster-admission-final-evidence-20261110/`:

| Run | Command | Exit | Result |
|---|---|---:|---|
| 01 | `fmt ./formats/raster` | 0 | formatted |
| 02 | `test ./formats/raster -count=1 -v` | 1 | build failed on two implementation name errors |
| 03 | `fmt ./formats/raster` | 0 | formatted |
| 04 | `test ./formats/raster -count=1 -v` | 1 | four independent fixture/expectation failures |
| 05 | `fmt ./formats/raster` | 0 | formatted |
| 06 | `test ./formats/raster -count=1 -v` | 0 | intermediate pass, superseded by EXIF-order coverage |
| 07 | `fmt ./formats/raster` | 0 | formatted |
| 08 | `test ./formats/raster -count=1 -v` | 0 | intermediate pass, superseded by final coverage additions |
| 09 | `vet ./formats/raster` | 0 | intermediate vet |
| 10 | `fmt ./formats/raster` | 0 | formatted final candidate |
| 11 | `test ./formats/raster -count=1 -v` | 0 | final: 14 top-level tests, 41 subtests, `ok ... 0.811s` |
| 12 | `vet ./formats/raster` | 0 | final vet, empty output |

Retained correction evidence:

- Run 02 corrected a shadowed JPEG frame variable and the PNG IDAT state name.
- Run 04 corrected the little-endian EXIF fixture count, an illegal-PNG case
  that accidentally selected indexed color, the odd-size 4:2:2 padded-surface
  expectation, and the 8192x1151 mip level count.
- Run 08 preceded the final post-IDAT eXIf, `jpeg.Encode`-produced entropy-valid
  JPEG pair, duplicate eXIf, and metadata-reader-failure coverage. Run 11 is
  the final test authority.

The final suite uses standard Go PNG/JPEG encoders and adversarial constructed
containers. It covers valid PNG depth/color variants; position-sensitive PNG
metadata and tRNS; exact orientation through public PNG and JPEG admission;
endianness and every short TIFF boundary; duplicate orientation/eXIf; APNG;
unsupported PNG/JPEG modes; exact SOS envelopes; legal and excessive fill bytes;
all four JPEG sampling ratios with odd dimensions; compressed/dimension/pixel
limits; checked overflow; chunk/marker truncation and traversal caps; bytes
after IEND; and ReaderAt failure/short-read behavior without panic.

Fixtures assembled by `minimalJPEG` are header-only admission fixtures: SOI plus
synthesized SOF0/SOS segments with no entropy-coded scan data. They are not
entropy-valid or decodable JPEGs, and no test treats them as such. Only fixtures
produced by `jpeg.Encode` are complete encodings.

## Frozen files

SHA256 and byte size, also saved as evidence file `13-source-sha256.txt`:

```text
608B178F47CDFB7A85779E4BD000C84FE130B9825B4D131BD7AFDC2269BB16DC  admission.go       5308
125A01145EB3A322746579921D39899C3F8224458A1A166A72C22DEE73AD64B3  png.go             9183
60DF8720E322CAD91618B76F5A15D5A566F77246D63B8FB878D500CCDCF686ED  jpeg.go           10819
14180CA7F300815E30FA21DE64334F3F07170B704B5608B5A60A1286623F9C50  budget.go          1900
23FB7FD054868F18150C1636ABA28558AF31C7D8717EFC502D1FE96B16DF8709  exif.go            2574
AC9788F1261ED42919A958E42204C46F466F778D35A58BA8F7840D7540C14C80  admission_test.go 27400
```

`native/` is untracked, so an empty Git diff cannot establish isolation. Scoped
inventory shows only `native/formats/raster/` plus raster task/design/review,
checkpoint, final checkpoint, evidence, and this result handoff. No application,
document, session, vendor, script, product documentation, package, or build file
was changed. No product build, worker, resource matrix, or decoder integration
was run.

## Limits

This is preflight, not CRC validation, decompression, entropy validation, or an
image decode. A later decoder must validate the same immutable bytes, enforce
the same limits, and surface every decode failure while retaining byte
inspection. PNG CRCs and IDAT content are intentionally unverified; JPEG entropy,
DQT/DHT contents, and terminal EOI are intentionally outside this phase. The
payload numbers are modeled allocations from the pinned Go codec paths, not RSS,
driver, GPU, or whole-process memory guarantees. Fixed codec tables are outside
the variable payload model. EXIF interpretation is intentionally limited to one
IFD0 orientation; other metadata remains preserved but uninterpreted. The next
bounded decoder task was not started.
