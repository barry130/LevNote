# Raster admission fixture-correction checkpoint

2026-11-10. Working checkpoint, not root acceptance.

The previous GLM owner's test fixtures were replaced with checked container
constructors:

- PNG chunks now use IEEE CRC32, IHDR edits rewrite their CRC, and metadata is
  inserted at the first checked target chunk rather than after IEND.
- JPEG insertion walks complete pre-SOS marker segments and inserts a complete
  marker, length, and payload before the target. It never searches entropy.
- EXIF fixtures use the selected endianness for the IFD0 entry count, public PNG
  and JPEG paths exercise orientation, and every short TIFF prefix is checked
  for rejection without panic.
- MCU expectations independently cover 4:4:4, 4:4:0, 4:2:2, and 4:2:0,
  including the 16x16 4:2:0 surface of `256 + 64 + 64 = 384` bytes and odd
  dimensions.

Production corrections include PNG envelope/order checks through IEND, legal
`eXIf` after IDAT, bounded chunk count including IHDR, exact JPEG SOS envelope
checks, grayscale charging one RGBA surface rather than two, and separate padded
luma plus two chroma plane sums. The former 17x17 4:2:2 and 8192x1151 mip
expectations were corrected after actual focused runs: 4:2:2 pads luma to 32x24
and the 8192x1151 chain has 14 levels.

Focused verification has already been recorded under
`native/handoffs/raster-admission-final-evidence-20261110/`; the latest test
passes. Final vet, file hashes, design synchronization, and the frozen result
handoff remain. No product build, worker, resource run, or out-of-scope file was
touched.
