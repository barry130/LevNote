# UTF-8 save allocation review — planning only

2026-11-09. Active for GLM
`read_native_handoffs_release10_docs_task_md`. Read this task if the payload is
opaque. No children. Root coordinates and accepts. Keep context below200K.
Own only `native/handoffs/utf8-save-resource-plan.md`; no product, tests, docs,
scripts, builds, launches or measurements in this planning task.

The next stage should address the user's memory goal with a small, measurable
change after the Find keyboard .14 release. Review actual document Save/SaveAs,
encoding, atomic write, disk verification and Snapshot.WriteTo paths. Currently
UTF-8 save materializes Buffer.String and a full encoded byte slice. Decide a
bounded UTF-8/BOM-only streaming save slice; legacy/UTF-16 may keep existing
encoding for now. Do not broaden to whole application memory benchmarking or
change file-safety behavior to obtain lower allocations.

At most two pages: identify confirmed full-buffer allocations vs unmeasured
process-memory hypotheses; propose the smallest shared atomic temp writer seam
and expected encoded-hash verification while preserving final conflict checks,
exclusive creation, permissions, untouched no-op bytes/timestamp, dirty/undo/
selection/baseline, composition refusal, readonly SaveAs and existing encoding
errors. Outline meaningful exact-byte/error/failure tests, including BOM/CRLF/
Unicode/chunk boundaries and existing-file rejection. Describe one bounded
before/after function benchmark with identical independently generated modest
fixtures (e.g.1MiB and8MiB) and honest B/op scope; no RSS/app-speed inference.
Specify ownership and acceptance checks without inventing new frameworks.

Read only the relevant files/current constraints, not huge histories or vendor.
Deliver the plan and stop; no implementation before root activates next phase.
