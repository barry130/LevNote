# Raster admission checkpoint

2026-11-10 10:25 local. This is a working checkpoint, not acceptance.

## Source state

Implemented `native/formats/raster` as a pure `io.ReaderAt` PNG/JPEG admission
package with typed rejections, checked budget arithmetic, PNG and JPEG subset
walkers, bounded EXIF orientation parsing, and focused table tests. The design
handoff exists at `native/handoffs/raster-admission-design.md`.

Corrections incorporated from root reviews:

- declared-size signature gate;
- PNG walk through IEND, explicit `acTL`/`fcTL`/`fdAT` rejection, and a 4096
  chunk cap;
- TIFF-relative bounded EXIF parsing, separate PNG/JPEG signatures, duplicate
  and short-boundary checks;
- PNG orientation temporary and JPEG simultaneous conversion/orientation
  surfaces;
- JPEG exact six-byte `Exif\0\0` APP1 signature, separate luma H/V sampling,
  admitted 4:4:4/4:4:0/4:2:2/4:2:0 ratios, bounded segment/fill work, and
  small SOS envelope validation;
- 4:2:0 decoder surface corrected to padded luma plus both chroma planes
  (`256 + 64 + 64 = 384` bytes for one 16x16 MCU), replacing the half-plane
  arithmetic bug.

## Actual verification so far

All commands were run through `native/scripts/go.ps1`.

1. `go fmt ./formats/raster` — exit 0.
2. `go test ./formats/raster -count=1 -v` — exit 1, build failed:
   missing `io` imports, a test short declaration, invalid append, and
   mismatched `mipBudget` return count.
3. After fixes, `go fmt ./formats/raster` — exit 0.
4. `go test ./formats/raster -count=1 -v` — exit 1, build failed:
   unused local and repeated short declaration.
5. After fixes, `go fmt ./formats/raster` — exit 0.
6. `go test ./formats/raster -count=1 -v` — exit 1, build failed:
   a `Fatalf` argument-count mistake.
7. After the fix, `go fmt ./formats/raster` — exit 0.
8. `go test ./formats/raster -count=1 -v` — exit 1, runtime/test failures:
   incorrect expected mip level count, a malformed PNG test fixture caused a
   test slice panic, and several incorrect fixture expectations.
9. After fixture/budget corrections and the first review revisions,
   `go fmt ./formats/raster` — exit 0.
10. `go test ./formats/raster -count=1 -v` — exit 1: malformed appended
    metadata was outside the parsed PNG/JPEG region, EXIF big-endian fixture
    was malformed, some IHDR edits targeted the wrong field, 4:2:0 expectation
    was based on the known missing-Y-plane bug, and the alleged mip boundary was
    above 48MiB.

## Remaining before final freeze

- Replace append-style fixture helpers with placement before PNG IDAT/IEND and
  JPEG SOS, preserving CRC and container endings.
- Correct IHDR offsets, big-endian IFD count, minimal SOS payload, and mip
  boundary fixtures.
- Add independent decoder-surface assertions for 4:4:4, 4:4:0, 4:2:2, and
  4:2:0 including the 384-byte 16x16 MCU case.
- Run one final focused test and vet, save actual streams/exits and file
  identities in `raster-admission-result.md`, then freeze.

No product build, worker, matrix wrapper, or broad suite was run.
