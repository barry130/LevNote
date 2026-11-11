# Binary core accepted

2026-11-10 root. Accepted source prerequisite for session/host integration after
reading binary.go, atomic.go, substantive regression code, final saved streams
and actual scoped hashes. Root did not rerun tests/builds. The .22 executable is
unchanged and does not yet expose these binary APIs.

The16MiB stable immutable snapshot is separate from text Document. Mutable
BinaryFile stays on its owner goroutine; immutable token/snapshot may cross
workers. Public Reader/ReaderAt/WriteTo do not expose the backing slice.
Exact SaveAs is exclusive, permits a held accepted source after external change
or deletion, verifies before adopting the destination, and reports committed
postpublication errors without deleting the destination or mutating baseline.
Same-path SaveAs verifies with streaming digest and never rewrites.

Final evidence under binary-core-final-evidence-20261110:
- focused-binary-2:17PASS,exit0, plus corrected rename-identity-3 PASS,exit0;
- document-suite-3:full affected suite PASS,1.111s,exit0;
- document-vet-3:empty output,exit0.

Review corrected a manufactured postcommit cancellation and a Windows lazy
os.Stat identity fixture. The final test uses the actual passed context/default
verifier and eager handle-based identities. Earlier misleading fixture results
are retained and explicitly superseded. Production already captures eager
handle identity. No physical UI, concurrency-race detector, resource, general
memory ceiling or copyright guarantee follows from these tests.

Final identities:
- binary.go14495B:E3D56794354B3CE2B1A9CBF93BD68A67E3F333A482CCA3EC5D8753DE18FF759D
- binary_test.go22341B:60637AD62D744E9331F584B6E60F95DBA268C2E2487DAB8D7F2E1B07D0CCFB19
- atomic.go3516B:33D45A65253972D8547A5DC7ACD1AC9FA04C0540EF11DD3DE207676801A232CC

Next core/session task activated separately. Async host copying needs a narrow
publication-receipt/UI-adoption extension before calling it on a worker; do not
race the present mutable SaveAs implementation.
