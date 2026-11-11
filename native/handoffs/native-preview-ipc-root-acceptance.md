# Preview transport root acceptance

2026-11-09. Root read current catalog and IPC production source, literal test
oracles, retained full scoped stdout/stderr and run metadata. The two production
additions are the fixed preview descriptor and its IPC alias. Markdown-only
ReadRequest remains unchanged. Source/response limits are16MiB/32MiB.

Accepted source assertions include independent fixed descriptor/frame bytes,
fragmented reads, source-limit classification with unread metadata/source sentinel,
foreign/wrong-format/wrong-version rejection before payload allocation, and legacy
ReadRequest rejection of preview before its source read. The corrected warning
oracle uses an independent7-byte source, not the unrelated response length.

Actual selected gate in `handoffs/evidence/preview-ipc-20261109`:

- test-full.stdout.txt: catalog0.508s and engineipc0.598s, exit0, empty stderr.
- vet.stdout.txt/vet.stderr.txt: empty, exit0.
- run-metadata.txt records full scoped test -count=1 and scoped vet arguments,
  start/end times. No new runtime/worker/UI/package execution at this gate.

Evidence limits: the original inventory.json lists files, not before/after
digests. No pre-run fingerprint inventory was retained; captured-after hashes
cannot prove pre-run equality. The older test-full.log contains a separate
0.544s/0.635s pass and is not the selected final gate. The owner has corrected
the handoff to distinguish those records. Root does not require a redundant
rerun to conceal this limit. Current-source review and actual package pass are
sufficient for this minimal fixed-descriptor addition.

App consumer implementation is active separately. This acceptance is not a
claim that the new preview is yet present in an accepted runnable bundle.
