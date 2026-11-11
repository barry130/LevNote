# Source-retention evidence labels

2026-11-10. Root has read the valid pre-change benchmark and fixture-validation
records. Parse/edit still matched the accepted pre-change identities at review.

The evidence-local `run-go.ps1` currently computes its `sourceSha256Before`
dictionary AFTER the invoked process has exited. Do not overwrite/reconstruct
the existing baseline records or repeat a passing benchmark solely for this
label. Disclose that the baseline fingerprint was captured after that invocation
and before production edits, supported by the accepted known parse/edit hashes.

For subsequent invocations, capture real before and after source identities
at the corresponding times and name them accurately. Preserve previous harness
identity or note the small harness correction separately; benchmark Go code and
fixture must remain byte-identical. Temporary fixture-identity test source was
not present when root read the tree; record whether it was an evidence-only
temporary probe and retain its exact source or equivalent deterministic identity
calculation in the handoff so the recorded dimensions/hash remain inspectable.

This is evidence hygiene, not a product blocker or instruction to repeat gates.
