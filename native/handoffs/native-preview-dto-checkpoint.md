# Neutral scene codec source checkpoint

2026-11-09. Active scope is only new `core/enginescene` source/tests and scoped
evidence. Public DTO/constants and exact required wire keys follow the corrected
contract; no catalog/IPC/runtime/producer/consumer change belongs to this gate.

Proposed files: `scene.go` (DTO, constants, shared validation), `decode.go`
(fixed-schema shallow reader), `encode.go` (shared count/fill emitter), and
independent literal/boundary/admission tests. Errors expose two stable sentinels,
`ErrInvalidScene` for syntax/schema/semantic refusal and `ErrLimitExceeded` for
payload/string/element/numeric budget refusal; callers may use `errors.Is`.
All Encode failures return nil bytes; Decode failures return zero Scene.

Decode rejects payload length before scanning. It never builds a generic JSON
tree: exact object key masks detect missing/unknown/duplicate/case-altered keys;
fixed node/edge/rect/point readers bound shape depth. Strings are scanned with
incremental decoded UTF-8 byte checks before copying, including JSON escapes,
surrogate pairs and raw UTF-8. Arrays enforce hard caps before parsing the first
excess element; late count fields are checked against actual retained lengths.
JSON numbers accept exponent/fraction forms, with exact integrality checks for
schema/counts and finite bounded float values. Standard outer whitespace is
accepted; trailing data, null and unexpected shapes are refused.

Shared validation checks field limits and exact normalized paints, then unique
opaque IDs, one root, references and inbound degree. An iterative traversal
checks connected rooted-tree structure in O(nodes+edges), without recursion or
source-format imports. All untrusted scene strings are owned copies on Decode.

Encode validates first, then uses one deterministic JSON emitter twice: bounded
count and exact-size fill. Strings are escaped directly without constructing
complete escaped temporary strings; success returns len==cap output. It does
not mutate caller slices or normalize/repair source values.

Tests will use authored literal JSON and independent expected DTO/bytes, every
layout, arbitrary key order/late counts, escapes/Unicode/surrogates, exact numeric
and string boundaries, malformed graph/schema and caller-byte mutation. Private
reader offsets/retained counts give direct admission oracles for overlong strings
and excess elements; payload refusal is checked before parsing. Full new-package
test and scoped vet will retain actual streams/commands and a small before/after
source/module/wrapper inventory. No existing package/process/UI/package replay.
