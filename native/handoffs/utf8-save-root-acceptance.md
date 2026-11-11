# UTF-8 streaming save: root source acceptance

2026-11-09. Root accepts this bounded source change for the next `.15` package.
The current accepted runnable package is still `.14` until binary acceptance.

Reviewed the actual shared atomic writer, UTF-8 routing, verifier, focused test
source, benchmark source and final raw test/benchmark/vet logs. Independently
hashed the five frozen source/test files and owner result; all match the
published freeze. The independent reconciliation reports no material defect
and has reconciled all five final evidence identities as well.

- Owner result: `utf8-save-implementation-result.md`, SHA256
  `142EEE3090FA6A3D2CC950915D35CCAE49C34EE565186B87F8DC0BE7A90209A8`.
- Final read-only reconciliation: `utf8-save-independent-reconciliation.md`,
  SHA256 `045CDA71B9945A7D486E5A703F4B4280AEB459368843E092B571AD77FCE9F9B1`.

Dirty UTF-8/BOM Save and SaveAs now write snapshot chunks through a 32 KiB
buffer, hash the actual output and preserve the shared atomic publication
lifecycle. Readback must equal that intended digest before saved snapshot,
disk token, baseline revision or dirty-cache advancement. Partial-writer and
pre-publication failures retain the target and clean up the temporary file.
Post-publication failures cannot promise rollback. SaveAs retains `.14`'s
Path/IsNew/ReadOnly update timing after publication and before verification.

Untouched saves, readonly/composition refusal, exclusive SaveAs, legacy/UTF-16
encoding, undo/redo and selection retain their established behavior. The final
document/session/app suite passed (0.818s/1.040s/16.036s) and vet exited 0.
Focused exact-byte/cleanup/state/digest/short-write/sentinel cases passed.
Intermediate failures remain in the evidence directory.

The 10x benchmark compares encoded-output generation from the same preset
snapshot to io.Discard, including exact output hashing. Setup/snapshot capture
are excluded for both. 1 MiB: 2,105,398 versus 33,131 B/op; 8 MiB:
16,785,478 versus 33,139 B/op. Streaming has 8 allocations/op versus 4 on the
full-buffer path; the measured benefit is allocated bytes, not allocation count.
It is not a whole-Save, process-memory, UI-speed or GC measurement. Snapshot
descriptors and saved backing chunks still exist.

Product source is now frozen for `.15` delivery. Delivery and a later six-doc
follow-up are delegated separately. No tests/builds/launches were run by root.
