# .9 delivery preparation checkpoint

2026-11-09. Source-only harness preparation is complete. No package, Go
compile/test, product build or application launch has been run.

Evidence directory:

`native/artifacts/acceptance/fixed-engine-package-delivery-20261109-103801/`

Prepared files:

- `context.json` and `commands.json` freeze the absent
  `native-worker.20261109.9` target, immutable `.8` input, planned counts and
  execution scope.
- `baseline.ps1` will fingerprint the scoped app/core/formats/modules/cmd/editor
  and portable package inputs plus the complete 23-file `.8` before packaging.
  `native/formats` is included; every referenced input, including
  `portable/NOTICE-PREAMBLE.md`, is required to exist as a regular file.
- `run-package.ps1` invokes the existing
  `scripts/package-markdown-worker.ps1` exactly once. That invocation performs
  the four existing GUI builds and then verifies and copies the exact 23-file
  `.9` release into evidence.
- `run-managed-real.ps1` copies `.9` into isolation, compiles the runtime package
  test binary through `scripts/go.ps1 -GoArguments`, then executes exactly one
  selected test from that binary:

  `-test.run '^TestManagedRealOldReleaseThenBundledResetSmoke$' -test.v -test.count=1`

  This bounds runtime execution to one named test. Its production operations
  are exactly two worker exports: activate/export immutable `.8` Markdown HTML,
  reset, then export bundled `.9` Markdown HTML.
- `run-host-smoke.ps1` starts one exact copied `.9` host with
  `-WindowStyle Hidden`, one ordinary unchanged Markdown CLI source and isolated
  `APPDATA`/`LOCALAPPDATA`, then sends `WM_CLOSE` to its exact owned window and
  verifies normal exit, no draft, an unchanged source and an unchanged binary
  bundle. It makes no tab-close engine-lane assertion; full app shutdown owns
  that lane.
- `aggregate.ps1` requires one package invocation/four builds, exact candidate
  and copy inventories, four GUI PEs, three engine manifests, unchanged `.8`
  and scoped inputs, the two managed exports, and the one hidden host
  start/close. Total bytes use an explicit `[Int64]` loop.

All four `acceptedApp*` paths remain null until root accepts GLM's final app
correction. The source-inventory field accepts either JSON records or GLM's
existing exact text format of `SHA256`, two spaces and an absolute path. Point
it at the final post-correction inventory; converting an existing final report
does not require another test run. Every execution script is
`executionAuthorized: false`; aggregate additionally refuses placeholders. After
activation, the accepted result, evidence directory, source-inventory path and
four tester captures must be filled and their source hashes must still match
current scoped sources.

Preparation made no physical UI, IME, GPU, memory, CPU or resource claim.
