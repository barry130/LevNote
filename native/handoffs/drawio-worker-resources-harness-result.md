# Drawio worker resource harness result

2026-11-10. Root accepted source checkpoint
`6b546efc1f19a38dec603ed9040c183d0ed6f4f69f3b14b3d470fedc05cc69a6`
(40,272 bytes) for bounded preparation validation. Validation is complete and
frozen for root review. No actual draw.io worker request, host request, GUI
launch, product rebuild, benchmark, matrix run, app suite or vet was executed.

## Frozen source

- `native/core/engineruntime/drawio_worker_resources_test.go`
  SHA256 `6b546efc1f19a38dec603ed9040c183d0ed6f4f69f3b14b3d470fedc05cc69a6`,
  40,272 bytes. The source was not changed during validation.

The future managed path remains separately opt-in behind
`LEVNOTE_DRAWIO_WORKER_RESOURCES_MEASUREMENT=1` and was not activated. Its
one-request-per-tester guard, pinned source/tester digests, exact installed
package path, process/probe checks and `limitations` handling are compiled into
the tester but unexecuted.

## Evidence-owned tester

One compile was run from the accepted source:

`native/scripts/go.ps1 -GoArguments @('test','-c','-o','F:\AIproject\LevNote\native\handoffs\drawio-worker-resources-harness-final-validation-20261110\drawio-resources.test.exe','./core/engineruntime')`

Exit code was 0. The exact tester is
`native/handoffs/drawio-worker-resources-harness-final-validation-20261110/drawio-resources.test.exe`,
SHA256 `e737a93cb11331b09b83113984b9d279aef6cab129ff590063fd21fdee845398`,
7,528,448 bytes.

The earlier tester `native/drawio-resources.test.exe` remains preserved with
SHA256 `50552a1b3ad8a83a366f910fdda8b9773ae5f36af9ebef4aced517ddcd8514c3`,
7,519,744 bytes; it is historical evidence and does not represent the accepted
source checkpoint.

## Exact validation records

One default-SKIP invocation of the exact tester ran with both gates disabled:

`drawio-resources.test.exe @('-test.run','^(TestDrawioWorkerResourcesHarnessPreflight|TestDrawioWorkerResourceMeasurement)$','-test.count=1','-test.v')`

Exit code was 0. `TestDrawioWorkerResourcesHarnessPreflight` and
`TestDrawioWorkerResourceMeasurement` both reported SKIP.

The validation wrapper emitted one unrelated PowerShell error before invoking
the tester: `Get-ChildItem:LevNoteDrawioWorkerResourceEnv`. The statement was
unused, did not affect the tester invocation or its captured result, and is
preserved in `default-skip-wrapper-error.txt`. The tester itself was not rerun.

One no-process preflight then ran with only
`LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS=1` and a fresh absent output directory.
Exit code was 0 and `preflight.json` reports `PASS`, with no failures. It
exercised the measurement JSON sentinel and all 12 matrix mappings:

1. NormalMultiPage `.20`, ordinal1
2. NormalMultiPage `.21`, ordinal1
3. NormalMultiPage `.21`, ordinal2
4. NormalMultiPage `.20`, ordinal2
5. NormalMultiPage `.20`, ordinal3
6. NormalMultiPage `.21`, ordinal3
7. MetadataHeavy `.21`, ordinal1
8. MetadataHeavy `.20`, ordinal1
9. MetadataHeavy `.20`, ordinal2
10. MetadataHeavy `.21`, ordinal2
11. MetadataHeavy `.21`, ordinal3
12. MetadataHeavy `.20`, ordinal3

Commands, stdout/stderr and exit codes are preserved under
`native/handoffs/drawio-worker-resources-harness-final-validation-20261110/`.
`preflight-output/preflight.json` is 8,797 bytes, SHA256
`1722314a3c99f4f70eb42f9e8b13078a3230a8f9642d2359d1884e9a9688911c`.
The preflight artifacts are in that directory's `preflight-output/` subdirectory.
Earlier final and development evidence directories remain unchanged.

## Fixed identities confirmed by preflight

- NormalMultiPage: 28,976 bytes, SHA256
  `ab822c28db64279a793810c45bbff7f6d7ecfd1cd44aaa1f59c48e00fedcaea2`;
  8 pages / 148 vertices / 42 edges; selected index2, ID `page-3`.
- MetadataHeavy: 93,639 bytes, SHA256
  `504b19ffcf126353b79b34bcb5735f6a1cff21d5d515b23bc294342c7a61bcb1`;
  6 pages / 41 vertices / 60 edges; selected index3, ID `page-4`.
- Expected SVG for both: 598 bytes, SHA256
  `3f990a4aac3209bafad94f44026bf14f3cec27ea44bdbd3169555947936472bf`.
- `.20`: NormalMultiPage expects two ordered selected-page
  `UNSUPPORTED_ELEMENT` warnings for canonical cells `0`, then `1`;
  MetadataHeavy likewise at selected page3. `.21` expects none.
- `.20` worker SHA256
  `70c15942339ecf074f81192c01ce652b19fdbc7dbdfaba139fed22e66c46fe9a`;
  release manifest
  `53c0b71c317734001596988f6ab2bf44201879cc322ba9db3b9ab494e553dbfa`;
  engine manifest
  `6175456b457fc85bd96e1a2ce2dbb8172474cdf6cdcde790ac80453f45c73fac`.
- `.21` worker SHA256
  `16e51c34688d39625626eab8a4d68aeb507ecc1cb9ffba7ea7478f9b8c0ff5d4`;
  release manifest
  `c2556b0e372466b5f67febf7c9fbacba0d44050b00f9892c64837ed16ce9a824`;
  engine manifest
  `caff96ac3181d9c5160a099ff9282fe37df9b11374c95472bd2237539fa14c18`.

Package artifacts were identical before and after the no-process preflight. No
host/UI/Tauri memory, speed, sandbox, physical UI or IME claim is made.
