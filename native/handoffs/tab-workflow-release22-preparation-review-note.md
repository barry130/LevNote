# .22 preparation draft correction

Root read the saved draft05:28. Before static parse/final handoff:

- Preparation requires COMPLETE reusable helpers. It only prohibits running
  them now. run-host-smoke.ps1 cannot be an unconditional throw, and
  run-package.ps1 must not throw after a successful build. Retain the normal
  ExecutionAuthorized guard and prepare actual code from accepted .20 helpers.
- Do not rewrite the harness from scratch. Copy the accepted .20 files into
  this new directory, then apply bounded .22/path/protected-.21 changes and
  accepted tab-gate parameters. Existing .20 startup/WM_CLOSE/session code is
  the requested implementation; no new UI automation or schema guesses.
- initialize-context has invalid `if (-not [Test-Path ...])` syntax; use actual
  copied PowerShell syntax. TabSourceIdentity is a hash string, not a path;
  checking Split-Path/Test-Path on it will reject every valid hash. Define and
  verify precisely what that identity represents, or replace it with accepted
  source identity JSON path and actual required fields after review.
- source-scope references native/portable/NOTICE-PREAMBLE.md; verify the file
  exists, otherwise use the actual copied accepted scope/portable asset list.

No package/host/test execution yet. One static parse after complete code. Do
not report placeholders as ready and do not invent another preparation phase.

Root continuation read after static parse2026-11-10: syntactic success alone
does not validate copied context references. Current initialize-context writes
ReviewedDiagramNotes as an object, but baseline.ps1 and run-host-smoke.ps1 read
nonexistent ReviewedDiagramNotesBytes/SHA256. They will always refuse before
launch. Likewise current context uses PortableSourceReadmeFingerprint, while
aggregate.ps1 and possibly run-package.ps1 still read old flat README fields.
Use one actual context schema consistently across the copied helpers. Check
every referenced helper function exists too: run-package uses
Invoke-CapturedPowerShellScript, absent in the current saved delivery-lib.ps1.

run-stage.ps1 and invoke-stage.ps1 still put open aggregate logs into the same
evidence root. The task explicitly requires aggregate capture logs in a
separate sibling directory; do that for this stage and reference them in the
result. Excluding known names is less clear than the requested separate path.
After aggregate exits, a final closed-file inventory may cover all evidence.
Never rerun a passed package/host step to fix metadata.

Finish these bounded copied-helper corrections, preserve initial parse record,
and parse the final changed set once. No product tests/build/host yet. Native
portable/NOTICE-PREAMBLE.md exists (492B), so that source scope is valid.

Follow-through check of saved corrections:
- The host's installed plugin is a COPY. Test-FingerprintEqual compares Path,
  so source-vs-copy must compare bytes/hash (and optional preserved timestamp),
  not require the two paths equal.
- Get-PEInfo is also called by run-package; ensure its actual accepted
  implementation/import is present, not only Invoke-CapturedPowerShellScript.
- Avoid Join-Path ($expr containing bare if); assign captureRoot with a normal
  if/else before Join-Path. Use a stage-specific sibling capture name so it
  cannot overwrite an unrelated generic aggregate-capture directory.
- Get-FileFingerprint for scoped individual files has Path but no RelativePath;
  Get-InventoryIdentity/Assert-InventoriesEqual currently only read RelativePath.
  Add a Path fallback for these scoped records so identity includes the path.
  Keep the accepted directory-inventory algorithm unchanged for .21 protection.
- Preserve prior parse records before final validation. Initial failure was
  already overwritten; disclose that factual evidence gap, do not fabricate it.

Read the full helpers together once before final freeze, not just single text
replacements. Root will review the finished consistent result; no launch yet.
