# Drawio source ownership accepted

Accepted2026-11-10. Root read final parse/edit ownership paths and callers,
external-mutation/deep-rollback tests, unchanged warning/oracle identities,
pre/post raw benchmarks and invocation records, final affected test/vet, and
GLM independent review. Root performed no implementation/tests/builds/launches.

Final repaired result5653bytes SHA256
FA44D4942A34ECA1F5A29743D30F534F594D29F9209E94D7127E712B82A1F8DA;
independent review with addendum4746bytes SHA256
3CB8135E3654D2BD5238B548558509E3E3EAE162055DC81A7DAC30BC12B6DBC8.
Five source/test identities match actual files; types.go unchanged. Author's
earlier handoff drafts contained corrupt prose and unsupported SVG numbers.
GLM repaired the result using existing verified facts, not new test evidence.
Addendum phrase 'removed unsupported numbers' describes the corrected report;
the corrupted draft had added those numbers. No product change from this repair.

Fresh parse now holds2distinct source arrays (one private original/current,
one mutable public copy), after edit3(pristine/current/public). Cap check before
caller clone, private fresh-buffer transfer, defensive returns and assignment
only after successful parse are correct. Actual all-byte external mutation and
deep failure state checks followed by later edits/reparse support isolation.
Unknown XML/CRLF/compressed readonly source survives; public API/BOM/warnings
unchanged. Existing canonical-warning tests, SVG oracle and worker fixture hashes
matched the previous accepted warning slice.

Fixed33985byte8page169node fixture SHA
04E533B0DA533713F51E106F820A7F4A62D0E0F06CA910EB6FEA9857A3D3E9F;
benchmark code unchanged pre/post,20x/count3. Median cumulative B/op:
Parse764244->722520(-5.46%), UpdateSerialize1733705->1487174(-14.22%).
Median ns/op1082480->1244525(+14.97%),2291545->2478320(+8.15%).
Short samples do not establish speed or whole-process/live-retained memory.
KeepAlive isn't a heap measurement. Baseline post-process fingerprint label
disclosed; known accepted pre-change identities and root observation corroborate
it preceded source edits. Later records capture real before/after identities.

Final affected test exit0:drawio0.587s,graphview9.423s,worker0.661s; vet exit0 with
empty streams. Initial fmt binding failure and incorrect cap-test allocation
expectation retained and disclosed. No additional reruns needed. .20 runtime
package remains historical; accepted warning+retention now may be delivered in
.21 using the separately accepted preparation and execution task.

2026-11-10 transcription clarification: the fixture SHA printed above is one
hex digit short. The raw baseline-fixture-identity-1.stdout.log records the
actual 64-digit value
`04E533B0DA533713F51E106F820A7F4A62D0E0F06CA910EB6FEAA9857A3D3E9F`.
This corrects prose only; source, measurements, frozen delivery evidence and
all previously checked file identities are unchanged. No rerun was needed.
