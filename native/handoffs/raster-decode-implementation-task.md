# Implement bounded PNG/JPEG decoding

ACTIVE2026-11-10 for DeepSeek read_native_handoffs_raster_admission_finish_task_md.
Root accepted admission in raster-admission-root-acceptance.md. Continue now,
no design-only pause. Own formats/raster/** and new evidence/handoffs; no
app/document/session/vendor/scripts/product build/host/resource work. No child.

Implement the actual Decode direction in raster-decode-next-plan.md: explicit
Go PNG/JPEG codec on the same immutable ReaderAt+declared size, admission inside
Decode, context-aware bounded section reader, no full-input clone or filesystem,
no background goroutine. Return small metadata and one owned tightly packed
origin-zero premultiplied image.RGBA. Check decoded raw bounds against admission
before output allocation and never return pixels on cancellation or failure.
The caller must supply a stable immutable reader (host uses BinarySnapshot).
The output must be valid for ui.NewBitmap without another full-size conversion.

Project-authored orientation1..8 and conversion should write directly into the
final surface, using standard color conversion rules; keep transparent colors
premultiplied and swapped final dimensions correct. No DecodeBitmap/remake or
silent orientation fallback. Honor ctx before each read and between conversion
rows, including final check; codec CPU loops between reads cannot be preempted.
Keep conservative admission formulas unless source evidence justifies changes.
Avoid general pluggable codec framework; a small private per-call decode seam
is fine only if needed to test final bounds/cancellation against actual logic.

Tests need independent asymmetric color-grid orientation oracles1..8, PNG exact
alpha/palette/grayscale/truecolor pixels and baseline JPEG known regions with
appropriate lossy tolerances. Include metadata after IDAT, corrupt CRC/IDAT or
truncated JPEG that passed preflight, ReaderAt failure, canceled initial/read/
conversion paths, forged/changed dimensions rejected before output publication,
and no retained input reader after return. Do not compare output solely to the
same implementation. Existing admission tests stay intact.

While here correct result wording:8192x1151 only passes private mipBudget; public
Admit rejects it for MaxPixels. Also label header-only constructed JPEGs as such.
No new broad performance runs. Use wrapper, preserve exact commands/output/exits
and failures in native/handoffs/raster-decode-evidence-20261110. One focused
decode run then one final affected raster suite/vet after actual fixes. Publish
raster-decode-implementation-result.md with API, ownership, limitations and
frozen source identities. Root then connects the separately owned host/view.
