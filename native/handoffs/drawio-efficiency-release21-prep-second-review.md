# .21 preparation review: JSON property casing

2026-11-10. Root read-only draft review; no product execution.

`Get-JsonArrayProperty` now deliberately compares property names with `-ceq`.
The Go record has lowercase `warnings`, while `run-managed-smoke.ps1` writes
the normalized result field as uppercase `Warnings`. `aggregate.ps1` currently
calls the helper with lowercase `warnings` for BOTH files. The second call will
throw after the successful actual worker run, because the normalized result
does not contain that exact property name. Use the actual written name for
the normalized result; preserve strict raw Go JSON validation.

Before preparation freezes, read both write/read paths together and exercise
the allowed tiny no-product JSON sanity check for null, empty array, uppercase
normalized result and rejected nonempty/scalar/object values. Retain the small
sanity output and actual exit; do not invoke a worker/host or initializer.
Rerun static parsing only if scripts changed since the recorded parse, retain
the earlier record instead of overwriting it. Execution remains inactive until
the accepted source gate is explicitly released by root.
