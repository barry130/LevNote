# Session-output `.16` documentation result

Status: FROZEN after narrow correction, 2026-11-10. Completed the bounded
six-document follow-up after
[root package acceptance](session-output-release16-root-acceptance.md), then
corrected only the adapter-validation and publication-failure wording identified
in root review. No source, test, package, evidence, or unrelated documentation
was edited.

## Documentation

The current runnable entry is
`native/bin/levnote-native-worker-20261110.16`, version
`native-worker.20261110.16`. The accepted package is 23 files /
23,690,310 bytes; host 14,066,688 bytes, SHA-256
`76A60252CFCC3394386B4DAF87AF8B845086DE35ED5DCCACC6CC79179E60B03D`;
manifest 4,137 bytes, SHA-256
`53994DB6861F4B9918D33F9FF90330744862730C838417E0C10501ADA050CD90`.
Package inventory identity is
`F9937B33552496EAEDA1FB2A7F68B7F27FB769915A6E4FEFC546E566D10E9CF5`.

The user-facing text describes the practical change: session recovery uses the
standard `json.Encoder` through an adapter into the existing private-state
atomic temporary-file writer. It removes only the additional full returned
copy previously created by `json.Marshal`. At runtime the adapter enforces the
preflight byte length, rejects short writes and overflow, retains one pending
byte, and lets `Finish` require that byte to be Encoder's mandatory final LF.
It does not explicitly validate the initial byte and does not perform an
oracle-hash comparison at runtime. Tests separately compare the complete
result with `json.Marshal` byte for byte, including field order, escaping,
omission rules, and the absence of a terminal LF.

Snapshot validation and the 4 MiB single-draft, 100-tab, and 16 MiB encoded
budgets occur before filesystem effects. Temp-file write, sync/close,
prepublication check, or publication-operation failure occurs before the
atomic commit point and preserves the old recovery file and timestamp.
Publication success is the commit point; the subsequent `syncDirectory` can
still return an error. It is currently a no-op on Windows and can fail on
Unix. In that postpublication case Save reports an error, but the new recovery
file has already been published and is not rolled back. Existing
retry/discard behavior remains.

The documents state that `json.Encoder` still owns a complete internal JSON
buffer and dirty draft strings remain. They do not claim token streaming,
fixed memory, elimination of all allocations, process RSS or working-set
improvement, whole-Save latency improvement, or physical UI/IME verification.
Sticky adapter errors are described as helper-contract hardening, not proof of
a prior production data-loss defect.

Validation records `core/document` 0.894s, `core/session` 1.646s, and `app`
17.571s, followed by scoped vet exit 0 with empty output. The corrected
output-only benchmark reports:

| Encoded bytes | Marshal path | Encoder/adapter path |
|---:|---:|---:|
| 1,048,701 | 1,164,120 B/op, 5 allocs/op | 106,213 B/op, 4 allocs/op |
| 1,049,114 | 1,469,992 B/op, 5 allocs/op | 412,825 B/op, 4 allocs/op |

Pooled standard-library encoder state affects these samples. The documents
limit them to output-boundary observations and include no process-memory,
UI/CPU, whole-Save, or physical-input conclusion.

The delivery documentation records validator and aggregate 12/12 PASS, one
existing package invocation producing four successful GUI builds, and one
copied hidden host with an owned normal close. The clean 413-byte session had
one UTF-8 Markdown tab, no BOM or draft, and a final `}` with no LF/CR. This is
not dirty-recovery or physical-input validation. The 299 scoped source/build
inputs (2,996,205 bytes), six frozen session source/test files, complete `.15`
package (23 files / 23,680,582 bytes), and its direct evidence (109 files /
47,758,090 bytes) were preserved. `.15` remains UTF-8 output history; the
`.11` worker run and `.13` visual evidence remain historical.

Evidence:

- [`.16` delivery](session-output-release16-delivery.md)
- [root package acceptance](session-output-release16-root-acceptance.md)
- [12/12 validator](../artifacts/acceptance/session-output-release16-20261110-001354/validator.json)
- [implementation result](session-output-implementation-result.md)
- [independent reconciliation](session-output-independent-reconciliation.md)

## Files changed

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` | 17,345 | `57FB7138B167A7445AAF213E35871C0A844B1D15B9EE2FAFFD142297EF397C2D` |
| `ROADMAP.md` | 117,366 | `98678AB1FF96FCA0A4CDAB9CD5731B51E8C0EE02944D7329FECEDB139DBE3EC4` |
| `CHANGELOG.md` | 139,013 | `169094E8BF3EF4A0A82ADD7788330DB049133EDF1F49D92B12B8C5586591D2CB` |
| `native/README.md` | 54,563 | `88C20F218C2E69DFE4924180D1045BCCEDA80BC7ADBB8DEB3DE4BF4375717BF6` |
| `native/ARCHITECTURE.md` | 54,367 | `12742C6EB2D77544845A883AA22B0FC3F07E82ECA1B48051D9A37544120BC9FF` |
| `native/VALIDATION.md` | 123,397 | `838F6E74C435463EF0F296D3289BDC12F4B800F9FD3F45AB0675D5721808EB9E` |

The correction pass replaces the inaccurate “首尾字节” contract with the actual
runtime length/pending-LF checks and clearly marks full-byte equality as a test
oracle. It also distinguishes failures before the publish commit point from a
directory-sync error after publication. One read-only local-target scan over
these six documents and this result found zero missing local Markdown targets.
The current host and manifest identities were checked once against the accepted
package during activation. These were documentation-only checks; no tests,
benchmarks, builds, package operations, worker requests, application launches,
or evidence conversions were performed.
