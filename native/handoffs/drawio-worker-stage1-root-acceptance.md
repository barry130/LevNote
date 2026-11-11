# Fourth worker lower layers: root source acceptance

ACCEPTED 2026-11-10 for later native-host integration, with separately frozen
warning correction. This is source acceptance, not a packaged release.
Current runnable package remains immutable `.18` with three workers.

Root read the original stage1 result, actual protocol/strict JSON/response
decoder, runtime admission/run/types, package state/migration, catalog/script,
parser scope and decisive fixtures; read final raw suite/vet/script-parse logs.
Independent GLM review in `drawio-worker-stage1-independent-review.md` found no
remaining blocker after the concrete warning-allocation correction.

Root independently checked all19 original non-worker entries against the frozen
manifest (hashes and lengths matched), both corrected worker hashes, and original
result identity. No tests/builds/launches were run by root or repeated for review.

Original stage1 result SHA256:
C476EF99BBAE3F8E7145274BEE0F1F9A7AC0AAA1FAF321D9C5E453F7308BF3BA.
Original manifest SHA256:
072828726C48B3BBE390B80FE5BFD0707955307B1CDFA72038CFA22FC2E2AEBE.
Correction result SHA256:
AF39CBEF0CAE6DDDBBF8AD1DB1BDDBD944810097D77BCFC69AFC61A15AF3603C.
Final worker main.go SHA256:
1B8EA397373D1CD3A2CC6F09D4FA6B9EA27E81B6F2FC0241722E7D7D4B85BE20.
Final worker main_test.go SHA256:
06007F93A9CCF105A16943B0F780322D5103266D84808145F2BF28FDE03E88E5.

Accepted: fixed fourth descriptor, wire v1/page strict bounds, old3 exact request
bytes, admission-owned page copy, exact index+ID without fallback, independent
second-page SVG oracle, scoped/ordered256warning bound with bounded conversion
storage, nil failed payload and nonnil empty warning array, schema1 old3 no-write
load plus bundled fourth, explicit schema2 exact4 persistence. Manifest schema1,
cleanup/poison/retention and atomic state semantics retained. Response zero-length
payload has nil slice; wire/error validation still preserves old successful
zero-length semantics. No unsupported guarantee attached to slice identity.

Original affected8package tests/vet exited0; some test outputs cached, runtime
38.381s, old worker tests also passed. After warning correction only changed worker
test0.728s and vet0 were rerun;19other files stayed identical. Packaging script
AST had0 errors; it was not executed. Historical failures remain in original logs.

Remaining stage gates: corrected view contract acceptance, app capture/recheck/
source-mode integration, portable assets, actual .19 host+4worker builds, one
managed installed drawio request, independent and root actual-binary acceptance.
No app/source-BOM/history, physical Windows UI/IME, sandbox, peak process memory,
performance or arbitrary executable-plugin claim from this lower-layer evidence.
