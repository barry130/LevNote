# Draw.io bounded SVG: root source acceptance

2026-11-10. Root accepts the bounded SVG source slice after reading the actual
emitter/types/tests, oracle assertions, graphview caller, original/final raw
logs, correction and independent reconciliation. Root independently verified
the final emitter/test hashes and result/reconciliation identities. Root did
not implement or run tests/builds.

`SVGOptions.MaxBytes` is zero/default64MiB, honors positive lower caps, clamps
above64MiB and rejects negatives. Invalid-page lookup retains precedence.
Counting precedes an exact-size output allocation; failed export returns nil
payload/issues. Successful bytes and ordered issues match the old2148-byte
oracle; warning-free success retains a nonnil empty issue slice. Source/model
and chosen page remain unchanged. Shared edit escaping is unchanged; SVG
escaping uses bounded UTF8-safe chunks verified against encoding/xml semantics,
including the production4096 boundary.

The review found nil/empty compatibility and unbounded continuation after writer
failure. Both corrected. Remaining work inside the current fixed numeric
fragment is bounded; no zero-instruction-after-error claim. Two original full
drawio/graphview suite runs passed (the second was redundant), then only the
changed drawio package was retested0.641s/exit0 and vetted0. Earlier red/compile
and incorrectly narrow-run evidence remains. Graphview callers were unchanged;
their successful prior suite and selected-page regression were not rerun.

Final implementation identities:

- types.go `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845`
- svg.go `47F3E584EEF6E2AD981FAE539EE6E2ECA3E93DDBD16F22B7DEC3D5FA086944CF`
- drawio_test.go `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189`
- svg_budget_test.go `79C1F18B12D35C238FA76E5F919BCA65FEE734AA62FC40BEB48FE43797261801`
- testdata/svg-prechange-oracle.json `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611`
- modules/graphview/graphview_test.go `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0`

Records: [initial result](drawio-svg-budget-result.md),
[review](drawio-svg-budget-independent-review.md),
[correction](drawio-svg-budget-correction-result.md),
[reconciliation](drawio-svg-budget-independent-reconciliation.md).
Correction SHA256 `24EDEC50884AE67B3C396FD2E99439216BE830340C6B252F1A4C01A707D8E611`;
reconciliation SHA256 `ADC9ED6A3D50F8C509909ABFC9BB58D4BBDEED0C00EEBD23DCF0B1D479D09BFF`.

The byte cap is not a process-memory cap. Parsing/source/indexes/issues and
per-edge points still allocate; successful output is held in memory; two passes
repeat geometry. No process memory/performance/physical UI proof. Draw.io still
runs host-side, three fixed worker engines remain. Fourth-worker plan is future.

Source frozen for the authorized .18 delivery in
[release task](drawio-svg-budget-release18-delivery-task.md). `.17` remains
current until root accepts actual .18 binary. All old packages/evidence immutable.
