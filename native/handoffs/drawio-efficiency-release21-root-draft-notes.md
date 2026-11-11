# .21 preparation draft review notes

2026-11-10 before prep freeze/execution, read-only root review. No product run.

Current managed smoke record appends zero warnings to a nil slice, so JSON emits
`warnings:null`. Existing run-managed-smoke uses `@($structured.warnings)`, which
can retain one null element, and `$warnings | ConvertTo-Json` on an empty list
emits nothing. Hashing that output via UTF8.GetBytes can fail or hash the wrong
representation. Before frozen prep, explicitly require warnings property and
accept only null or an actual zero-length array for this exact zero-warning
smoke; reject scalars/objects/nonempty arrays. Normalize to a retained empty
array. Serialize with `ConvertTo-Json -InputObject @()` rather than empty
pipeline for stable `[]`, then hash bytes. Final aggregate must match the same
semantics for raw record and normalized managed result. Keep test/protocol
production unchanged; this is evidence harness interpretation only.

Current initialize-context validates benchmark/source gates before execution;
that's appropriate only once they exist. Do not invoke it during preparation.
Static parse doesn't exercise these data semantics, so explicitly inspect the
zero-warning path and include that limitation in prep record. Task allows a
tiny no-product JSON-expression sanity check if needed, not managed/host runs.
