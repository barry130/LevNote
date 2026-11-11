# Raster decode source acceptance

ACCEPTED2026-11-10 by root, source/evidence read-only review. No tests, builds,
launches or measurements were rerun by root. Current runnable .22 stays unchanged.

Reviewed decode.go, decode_test.go, the frozen result and final recorded full
suite/vet command, output and exit files. SHA256 matches:

- decode.go: 322150A187715FF2A769BE410520B6CB5229E9EE2D03FAA7F0FEFF0500963060
- decode_test.go: 859B9C7633674666D4F5400675CC942EBC63B130389ED3F0FAFC0F5A18C4CEFD
- admission.go/admission_test.go still match accepted admission identities.

Accepted behavior: mandatory bounded admission on the same immutable ReaderAt,
explicit PNG/JPEG codec, independent bounds check before final allocation, one
owned origin-zero premultiplied RGBA output, EXIF1..8 direct conversion, no source
retention and nil pixels on failure/cancellation. The asymmetric-grid and known
region fixtures exercise actual codecs; seam tests cover geometry and conversion
cancellation without pretending to be codec integration tests.

Final affected suite records24 top-level tests/71 subtests PASS, exit0; vet exit0
with empty output. Admission source stayed frozen. Earlier focused passes are
retained; no repeat requested. Admission-result prose corrections are accurate.

Limits remain explicit: codec CPU between guarded reads is not preemptible;
payload accounting is not a measured process/GPU ceiling; supported format subset
is narrower than all PNG/JPEG. No UI/upload/driver/host or packaged-image support
is established by this acceptance. Next owner may implement the neutral native
image view, while app wiring waits on binary copy/session contracts.
