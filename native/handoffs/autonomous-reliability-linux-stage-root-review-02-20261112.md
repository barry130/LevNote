# Autonomous reliability: Root Linux/source-stage closure02

Date: 2026-11-12. Reviewer: DSH lead. This is the successor to [the manual source review](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-root-review-20261112.md>); the earlier report and original execution records remain unchanged.

## Decision and exact boundary

The five-source reliability changes and the recorded Linux execution evidence are accepted as a finite development stage. The original affected command remains **FAILED**, not GREEN. The unchanged Windows uppercase-path fixture remains a consolidated Windows acceptance requirement. No whole-app, formal parallel Standards/Spec, integration, package or platform acceptance is granted.

The remaining task5 report/raw byte-mirror is documentation delivery work. It does not block the separately scoped task11 document fingerprint development. Native-reliability remains the only product-command owner and must finish that report mirror without rerunning any consumed command or label.

## Independently verified source and inventories

Root read the complete production changes, the public autonomous tests and the scoped working-tree diff described in [review01](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-root-review-20261112.md>). All five source mirrors have the full identities recorded there. No host Go/gofmt/test/vet/build/run or semantic source implementation occurred.

Both actual labels have three identical inventories: source-before, source-after and snapshot-after, each `199410 B`, SHA256 `2a087081672f38a6161a8e9f25aff421a00f6e8b9ac8d0829ba803779b5c6bda`. Root independently parsed every inventory and compared its roots and all **1192** file size/SHA entries with the label's expected manifest; every comparison was equal, with no missing, added or mismatching entry. These were read-only metadata comparisons, not product execution.

- Affected expected manifest: `201209 B`, SHA256 `f118f82abf54b7083289f953eb0b65097b15e3314caa999973df5b764b29a86f`.
- Vet expected manifest: `201204 B`, SHA256 `72aaa56953f8603b95e64ff4d45fb0ca21158f0cab5c286e517fe6bf1c99e642`.
- Both resolve their ordinary development input HEAD to `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`. This is not an assertion that the combined current host mirror was tested as one source group.

## Actual affected execution: FAIL preserved

Primary evidence: [affected result](</home/user/levnote-dev-reliability-20261112-01/runs/native-path-affected-01/evidence/result.json>), [numeric command receipt](</home/user/levnote-dev-reliability-20261112-01/runs/native-path-affected-01/evidence/05-product-command.json>) and [complete stdout](</home/user/levnote-dev-reliability-20261112-01/runs/native-path-affected-01/evidence/05-product-command.stdout.bin>). Root read the complete 142-line wrapper result and all 251 stdout lines, not only a summary or failure header.

```text
go test -v -count=1 -timeout=300s -run '^(TestAutonomousReliability.*|TestReopenClosed.*|TestResolveLocalReferenceClassification|TestEditorHost.*|TestNativeImage.*|TestBinaryInspectorLabelsOriginalBytes|TestLiveTabAdmissionAsyncBinaryCleanupAndDuplicate)$' ./app
```

- Go PID/owned PGID `23843`; launch `2026-11-12T09:16:00.042582Z`, end `09:16:26.094242Z`; elapsed `26.052711416006787 s`; Go exit `1`, wrapper exit `2`.
- Limit `360 s`; timed_out=false; owned_cleanup=[]; owned_group_remaining_after_wait=false; stderr `0 B`.
- Stdout `17463 B`, SHA256 `d0440f9b2a8b9ac995985b175ff120998c80c49be80ecb1e5caa38b92291575e`.
- Actual top-level disposition: **47 started = 45 PASS + 1 FAIL + 1 original opt-in capture SKIP**, with 76 nested PASS. The selector completed without a panic. This is not a total-app coverage count.
- All five new public tests passed. The original running-image cancellation flow and formerly empty closed-history indexing flow also passed.
- The sole failure is [the unchanged reopen dedup fixture](<F:/AIproject/LevNote/native/app/reopen_closed_test.go#L266-L283>): line279 `reclose did not dedup with a new ID`. It uppercases the entire native temporary root and filename. On Unix that identifies a distinct/missing native path; Unix path identity must not impersonate Windows folded identity. The assertion, source and failure are retained without a new skip or weakening. Its real Windows result is still PENDING.

## Actual affected vet: limited PASS

Primary evidence: [vet result](</home/user/levnote-dev-reliability-20261112-01/runs/native-path-vet-01/evidence/result.json>) and [numeric receipt](</home/user/levnote-dev-reliability-20261112-01/runs/native-path-vet-01/evidence/05-product-command.json>). Root read the complete 132-line wrapper and the complete numeric receipt. The exact existing label is `native-path-vet-01`; no substitute label or command rerun was created.

```text
go vet ./app
```

- Go PID/owned PGID `24554`; launch `2026-11-12T09:24:11.825077Z`, end `09:24:46.435805Z`; elapsed `34.614990396017674 s`; Go exit `0`, wrapper exit `0`.
- Limit `240 s`; timed_out=false; owned_cleanup=[]; owned_group_remaining_after_wait=false; stdout and stderr both `0 B`.
- Actual wrapper outcome `COMMAND_PASSED_LIMITED_SCOPE`. It does not change the affected test FAIL or grant Windows/full-app acceptance.

## Helper, environment and next stage

The actual executions use Go1.27.3, linux/amd64, CGO=0, vendor, local toolchain, proxy/sumdb/telemetry off, GOENV off, GOWORK off and GOVCS `*:off`, with isolated label HOME/XDG/tmp. The Root-reviewed helper03 (`17964 B`, SHA256 `6f3dd79076e9f284691f24be1a8e3b0c91d57d9c0ed5766d07cc87445acdff89`) has a separate real child lifecycle RED/GREEN/normal-control proof. That harness has `product_executed=false`; it is not Go or platform acceptance. Original helper02 and every consumed label remain unchanged.

Next: native-reliability finishes the original-byte report/raw delivery and begins task11 only under its separate `native/core/document` scope. Task11 must bound Save, post-save readback **and CheckExternalChange** by actual encoded expected length plus one, preserve same-bytes/SameFile mtime-only compatibility, exact digest and honest post-publication failures. It must not modify these frozen path sources or interpret this decision as permission to repeat their regression.

Task6's Tree/Outline failures and task8's draw.io Undo failure retain their own owners and fresh correction inputs. Feature convergence is followed by formal independent review, consolidated integration/platform gates, packaging/docs, CI restoration and dual-main delivery. `.29` remains the latest accepted runnable; `.30` remains unexecuted preparation.
