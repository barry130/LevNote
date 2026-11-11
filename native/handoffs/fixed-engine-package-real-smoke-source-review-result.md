# Real managed smoke source — corrected and frozen

2026-11-09. GLM applied the full review list to its owned file
`core/engineruntime/managed_bundle_test.go`. No test, vet, worker, app, build,
or process execution was performed.

## Corrections

- Removed the unused `Inventory` field; the helper no longer reads a release
  inventory into unused metadata.
- Added an immediate `t.Cleanup` close fallback after each successful
  `NewManaged`. Explicit normal `Close` remains present for each runner, and
  the fallback does not overwrite the captured normal-close result. The
  original `RunnerCloseError` is retained and appended when the second close
  also fails, rather than overwritten.
- The test now logs exactly one structured JSON record.
- It checks actual `EngineInfo.ID`, receipt engine/version, source bytes after
  both requests, exact installed/bundled output parity, different
  source-release and host directories using case-insensitive Windows cleanup,
  and distinct old/new versions before proceeding.
- Metadata now reads each release inventory and fixed Markdown manifest through
  held bounded file handles, not unbounded reads, and checks requested
  identity, inventory/manifest version, executable size/hash, manifest size
  and platform/protocol identity. No extra executable is executed or hashed;
  the real Manager/Runner remains the authoritative verifier.
- The opt-in wording in the checkpoint is corrected: the flag alone is not
  sufficient. If the source-release path is absent after `LEVNOTE_MANAGED_REAL
_SMOKE=1`, the test fails explicitly instead of skipping.

## Frozen source

```text
core/engineruntime/managed_bundle_test.go
56b647217653bc6f046a7720b7d219b57472309f6ef38d6ac7e7c7a174c417d4
```

The test remains default `SKIP`. Compilation may be gated later only when root
authorizes it and coordinates with Astra. This result records source
preparation, not proof of joined runs.
