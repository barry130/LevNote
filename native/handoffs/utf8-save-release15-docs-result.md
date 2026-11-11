# UTF-8 streaming save: .15 documentation result

Status: FROZEN, 2026-11-09. Completed the bounded six-document `.15` follow-up
after [root package acceptance](utf8-save-release15-root-acceptance.md). No
source, test, package, evidence, or unrelated documentation was edited.

## Documentation

The current runnable entry is now
`native/bin/levnote-native-worker-20261109.15`, version
`native-worker.20261109.15`. The accepted package is 23 files /
23,680,582 bytes; host 14,056,960 bytes, SHA-256
`3C5B07646C2E50A82304B88A02B59C2EB170F067599B7FFAB044DCCEBF3DD804`;
manifest 4,137 bytes, SHA-256
`3A451E08D9394F829CEE0E85D7C7F14F1FFB735C47049DB6CEF502EE22E35D32`.
The actual executable and manifest were inspected once during activation.

User-facing text describes the practical change: dirty UTF-8/BOM `Save` and
`SaveAs` stream the existing snapshot through a 32 KiB buffer instead of first
assembling another whole encoded-text copy. Optional BOM, exact bytes and line
endings, no-op behavior, conflict checks, readonly/composition refusal, atomic
publication, and exclusive `SaveAs` remain unchanged. UTF-16, GBK, and GB18030
retain the full-buffer path.

The architecture contract records that the committed readback digest must
equal the exact intended output digest before advancing the saved snapshot,
disk token, baseline revision, or dirty cache. Post-publication failure has no
rollback promise, and `SaveAs` retains the `.14` Path/IsNew/ReadOnly timing.

Validation records the final affected suite (`core/document` 0.818s,
`core/session` 1.040s, `app` 16.036s) and vet exit 0 with no output. The
fixed-10x output-only benchmark is reported as 2,105,398 to 33,131 B/op and
16,785,478 to 33,139 B/op, with allocation count increasing from 4 to 8. The
documentation explicitly limits this to encoded-output allocations and excludes
snapshot capture/baseline retention, whole-save latency, peak/process
memory, RSS, working set, private usage, UI/CPU/GC, and physical-input claims.

Validation also records the 12/12 package validator, four successful GUI builds,
one hidden exact-copy startup with normal owned `WM_CLOSE`, preserved
source/session/bundle, and no dirty Save or worker request in that smoke. It
preserves `.14` and `.13` as history; the `.11` managed worker requests remain
historical evidence.

## Files changed

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` | 16,584 | `311704B15E1AD37B667F7402ABDAA1AF612EFC568821A04DC9FD90EDD89F019C` |
| `ROADMAP.md` | 116,046 | `278257F02D77E41B9762C6B67FC56C8D1BE5DA40D957A77ACDC73A1BADB7A1FA` |
| `CHANGELOG.md` | 136,873 | `0454D6ED87AFA1E98013BAACCBC86E1C5BECFEE50AFAF3762F41FCE22E38BC0F` |
| `native/README.md` | 52,435 | `633A89163A0434A9A7D041918F6598B0C1D176D5EE5F3B342B1A9EAF814483E2` |
| `native/ARCHITECTURE.md` | 52,085 | `BD766D54FB473CC4A47F400A3B5B95B9B76FE085C30CA36F9D19C711B0BD1087` |
| `native/VALIDATION.md` | 118,374 | `87A0FD69EC6A95D30F50917DE4F0936AED6B08B06C4018D446340D38CE496C31` |

The documents link to the accepted source result,
[implementation result](utf8-save-implementation-result.md),
[independent reconciliation](utf8-save-independent-reconciliation.md), and
[`.15` delivery](utf8-save-release15-delivery.md), plus the retained raw
validator and focused evidence. One local Markdown target-path scan over the six
documents and this result checked 379 local targets with 0 missing.

## Freeze and limits

This was documentation-only work. No test, benchmark, build, package, worker,
application launch, evidence conversion, or source/evidence modification was
performed. The accepted `.15` binary and evidence remain immutable. The
benchmark is not an application performance or resource measurement, and the
hidden startup does not prove a dirty save, UI input, or worker behavior.
