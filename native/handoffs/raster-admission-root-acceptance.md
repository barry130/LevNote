# Raster admission source accepted

2026-11-10 root source/evidence review. Accepted for the pure decoder continuation,
not as a working image viewer or process-memory guarantee. Root read all six
implementation/test files, actual pinned PNG/JPEG allocation paths, corrected
fixture constructors, final design/result and final test/vet streams. Root did
not rerun checks. All six actual SHA256 values match the frozen result.

Final authority is evidence11-final-test:14 top-level tests/41subtests PASS,
exit0;12-final-vet empty stdout/stderr,exit0. Earlier failures remain recorded.
The previous missing historical logs are disclosed rather than recreated.

The public ReaderAt preflight limits compressed input to16MiB, each side8192,
pixels8388608, modeled base+ceil-halved mips48MiB and modeled decoder payload
128MiB. Header admission never authorizes final pixels: the explicit PNG/JPEG
codec must validate the same immutable snapshot, with final dimension checks,
cancellation and visible fallback on error. No CRC/entropy/EOI validation is
claimed by Admit; no filesystem, UI, product build or real resource evidence.

One result wording correction to include in the decoder continuation: the
8192x1151 example passes the PRIVATE mipBudget helper, not public Admit (it
exceeds MaxPixels). Avoid saying that image size is generally admitted. The
public rejection logic is already correct; no production correction required.

Accepted scoped identities:
- admission.go608B178F47CDFB7A85779E4BD000C84FE130B9825B4D131BD7AFDC2269BB16DC
- png.go125A01145EB3A322746579921D39899C3F8224458A1A166A72C22DEE73AD64B3
- jpeg.go60DF8720E322CAD91618B76F5A15D5A566F77246D63B8FB878D500CCDCF686ED
- budget.go14180CA7F300815E30FA21DE64334F3F07170B704B5608B5A60A1286623F9C50
- exif.go23FB7FD054868F18150C1636ABA28558AF31C7D8717EFC502D1FE96B16DF8709
- admission_test.goAC9788F1261ED42919A958E42204C46F466F778D35A58BA8F7840D7540C14C80

Next task: raster-decode-implementation-task.md. The current .22 package remains
unchanged and does not contain this feature. No repeated admission-only suite
needed before the actual decode changes.
