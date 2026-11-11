# Schema3 core gate — passed

2026-11-09. Actual scoped tests and vet passed; proceeding directly to the
already activated app slice. No app, worker, package or format-engine edits
were part of this gate.

Evidence: `artifacts/acceptance/format-routing-core-20261109-093553/`.
`run.ps1` invokes retained `go-check.ps1` through hidden PowerShell children;
the latter uses the pinned `scripts/go.ps1` with explicit arrays:

```
test ./core/plugins ./modules -count=1 -v
vet ./core/plugins ./modules
```

Actual raw `test.stdout`: plugins PASS2.463s, direct modules PASS0.905s;
test exit0 (4.0648749s), vet exit0 (1.1874558s), empty stderr. The14 recorded
inputs have identical before/after bytes and SHA256. `result.json` SHA256:
`554790CEE2B01AB32CDFEC033170BC2E791D398A5609B2BB21F3B03CCBCDE0B3`.

New substantive checks cover the two routes, omitted-only defaults, strict
fields/types/nulls/duplicates/schema disguise/surrogates, exact64KiB admission,
legacy defaults/duplicates, actual exclusive install, normalized metadata and
defensive copies, enable/disable/same-ID refresh, settings refusal and a real
post-publication256-manifest rollback. Direct composition checks exercise
native templates and semantic outlines, capability pointer/slice isolation,
builtin disable independence and the schema2-only outline factory.

The preliminary direct command exposed a missing brace in the new test loop;
`schema3_test.failed.go.txt` retains that source, and
`preliminary-tool-transcript.txt` is explicitly labeled a copied tool-output
excerpt (not a redirected raw stream). The test-only syntax repair passed the
recorded final gate. Preliminary compile/legacy-only checks are also disclosed
there; no unreported product failure or simulated pass.

Final source SHA256:

| File | SHA256 |
|---|---|
| core/plugins/declarative.go | 909AFCFC82DB3BE6C6D472C085A5F28FF4C21850D1E4A20697F31AAFB040C037 |
| core/plugins/native_route.go | 9D73C47A9FD2EDF68C69D19A727C78CD74C33019829DB5F12946A291714923F0 |
| core/plugins/schema3_test.go | 3C19F2D7700F8E5F05460B2A23B1F2D4EDDA4F5285DC59D3D9DF6727A4642C0D |
| modules/modules.go | E15E5D0900F7339C2021CB9CDF61B53737263E4C3F2C33459E28B755AD005986 |
| modules/native_route_test.go | 5FB92B747391759A947299E98FA2FA36DA350C916DD5F3ABB68E7A1B3EB25813 |

API/source details are in `format-routing-core-checkpoint.md`. Accepted `.7`
artifacts are untouched. This proves local configuration/composition only;
schema3 is not an independently installed engine or an OS isolation boundary.
