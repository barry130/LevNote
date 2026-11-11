# Root review of initial raster draft

2026-11-10. This is early concrete feedback on WIP, not final acceptance.
Continue implementation within existing task. No tests until matrix ends.

- exifOrientation currently slices payload[6:8] before proving len>=8 (len6/7
  panic), uses TIFF-relative IFD offsets against whole APP1 bytes, and assumes
  PNG eXIf has the JPEG Exif\0\0 prefix. Normalize to a bounded TIFF sub-slice:
  JPEG validates/removes exactly six-byte signature; PNG eXIf starts directly
  with TIFF. Check every offset/count against that slice before slicing. Test
  both endiannesses and each short boundary, pointer offset at end, duplicates.
- PNG draft passes orientation as a numeric payload part rather than charging
  another4*pixels for orientations2..8. Document the actual allocation formula
  and test non-square orientation before accepting budget result.
- JPEG Go Adobe/RGB conversion can retain the original YCbCr surface plus a
  newly allocated RGBA; viewer RGBA and orientation allocation may be separate
  again. Inspect exact pinned codec and future MyGo conversion path. Model all
  simultaneously reachable payloads or reject RGB/Adobe mode in first slice;
  do not assume one4*pixels term covers two allocations.
- PNG stopping at first IDAT cannot reject a trailing acTL or read eXIf after
  IDAT. The task promises APNG rejection and bounded metadata discovery: walk
  container chunks through IEND, skipping IDAT without reading its payload,
  preserve no-CRC/full-decode caveat, reject malformed animation markers/order
  rather than silently accepting only a first frame. Bound traversal count/work
  (16MiB input alone can contain over one million tiny chunks).
- Signature read should honor declared size before asking ReaderAt for8 bytes;
  do not read outside declared envelope. Reject invalid full-count+non-nil
  error deliberately/consistently. Exact rejection/cause can remain simple.
- mipBudget checks the cap before the last1x1 addition; enforce final total as
  well. Public admission dimensions protect arithmetic but private helpers must
  not loop forever on zero/negative dimensions if directly used/tested.

Do not overbuild metadata parsing. Single bounded IFD0 and explicit unsupported
cases are sufficient. Keep API/formulas and source consistent, use real codec
fixtures/adversarial independent expectations, preserve all failed checks.

Second WIP read10:05:
- JPEG compares a six-byte payload slice to five-byte string "Exif\x00";
  valid EXIF is six bytes Exif\x00\x00 and currently is never recognized. Test
  through real APP1 container, not only private TIFF helper.
- maxSampling=max(h,v) loses independent lumaH/lumaV. Example luma2x1,
  chroma2x1 has actual ratios1x1 but current formula derives1x2 and charges2
  instead of3bytes per padded pixel; sufficiently tall files are underestimated.
  Also luma2x1/chroma1x2 is admitted by current maxima but unsupported.
  Keep H/V separately and validate actual Go allowed ratios, or deliberately use
  3bytes per padded pixel as a proven conservative surface bound with explicit
  API/documented estimate. Do not claim4:4:0/4:2:2 factor1.75; planar Y+2chroma
  factor is2, while4:2:0 is1.5. Model exact padded allocations or bound honestly.
- PNG maxChunkCount=size/12 is the inherent file bound and still permits over
  one million allocation-heavy header reads. Use a concrete small cap (e.g.4096)
  and explicit rejection; apply bounded segment/fill-byte traversal for JPEG.
  Reject fcTL/fdAT as unsupported animation markers too.
- JPEG legal repeated0xff fill bytes should be consumed with bounded progress,
  not interpreted as a segment type/length. Add real encoder fixtures plus
  crafted sampling/fill cases. SOS can stop entropy parsing, but verify small
  SOS envelope/expected scan before claiming supported baseline subset.

Third read10:23 uncovered a new production arithmetic bug: jpegAdmission sets
bytesPerMCUPixel=0 for4:2:0 and adds only half the MCU pixels; the Y plane is
missing. A16x16 padded4:2:0 MCU needs256+64+64=384bytes, not128. Use explicit
padded luma and both chroma plane sums with checked arithmetic. Independently
assert this and 4:4:4/4:2:2/4:4:0 cases (including odd sizes/common factors).

Current tests are still invalid rather than a validation basis: PNG helper
appends metadata after IEND and omits CRC; EXIF fixture drops IEND; IHDR16bit/
interlace edits target width/height offsets; JPEG helper appends metadata after
SOS/entropy, so admission cannot see it; minimal SOS payload{0} is malformed;
big-endian IFD count writes high byte (256entries); alleged mip boundary8192x2047
already exceeds48MiB. Build real chunk/marker placement helpers with checks,
use valid known fixtures, change the specific intended field, and assert that
the tested branch was reached. Do not relax production to pass incorrect tests.
Save actual tests/failures/commands/exits and final file identities as instructed.
