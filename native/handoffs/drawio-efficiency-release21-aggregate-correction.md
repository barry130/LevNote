# .21 aggregate evidence correction only

2026-11-10. Root read actual package, managed worker and host records. Product
operations passed. First aggregate15/16 false host gate plus locked-log error
are harness defects, not grounds to replay any product operation.

1. aggregate.ps1 uses `$hostResult.SourcePreserved`, but actual run-host-smoke
   writes `FixturePreserved` (true), verified by exact before/after hashes.
   Replace this property in the COPIED evidence aggregate script only.
2. Aggregate inventories its own outer/stage redirected log files while they're
   open. Get-FileHash fails for outer-aggregate.stderr.log. Exclude only the
   actively open aggregate invocation streams/records from in-process aggregate
   Evidence. After that process exits, record final complete evidence inventory
   from the parent as a separate external handoff record, avoiding self hashing.
   Don't force access to live files or recursively inventory unrelated folders.

Preserve first failed validator, scripts and all invocation/log files under
distinct first-attempt names before corrected execution. Run ONLY aggregate
again with distinct second-attempt outer/stage log names, or direct bounded
captured invocation whose redirection lives outside the scanned evidence root.
No package build, worker compile/run, host launch or source tests repeated.
No changes to product/gates/prepared harness/.20. Report this narrow metadata
retry with actual exits and final checks. Frozen final evidence must include
the first failure; do not turn the failed validator into a historical PASS.
