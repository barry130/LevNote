# Conditional real-worker CI plan — inactive and incomplete coverage

Date: 2026-11-14. Task20 preparation only. No executable in this plan was built or launched by the preparation owner. The base [runner](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/run-native-ci.ps1>) clears every inherited `LEVNOTE_*` opt-in and reports real-worker status **NOT RUN**. Native unit/helper processes are not renamed into four real shipped-worker requests.

## Preconditions for a separate later root activation

1. Root has a frozen current shipping-source inventory, final helper review, actual supported Windows x64/PowerShell7/Go1.27.3 identity, full base native test/vet/build evidence and a new five-real-PE package. Neither an accepted old package nor legacy frontend tests is a substitute.
2. Use an ordinary, absent run-owned evidence tree and config roots. Preserve the candidate package, source, tools and all historical packages. No touching the user's real profile, no reuse of previously consumed evidence, no fixture file smuggled into the shipped candidate.
3. Build the reviewed Windows integration test executable from the current `./core/engineruntime` package into a fresh **evidence-owned copy** of the candidate, not into the candidate. Capture build/run arguments, source/helper/runtime/bundle pins, streams, timeouts, actual held-process identity, owned Job/join outcome and before/after inventories. A console test image is not a substitute for the actual GUI host.
4. Set only the exact test-specific environment, retain default offline/cache/telemetry policy, run each required named test once. Assert both expected test PASS and zero SKIP; exit0 alone is insufficient. Read the test's own result/status/process fields and independently bind them to the candidate bytes.
5. Failed launch/timeout/cleanup, missing evidence or unavailable runner layout remains FAIL/NOT RUN, never a platform PASS. Root reviews new raw outcomes; no implicit retries or resource/physical input assertions.

## Existing literal test contracts, not invented all-engine coverage

| Contract | Required environment | Actual limited scope |
|---|---|---|
| [Markdown managed old-release/reset test:62–86](<F:/AIproject/LevNote/native/core/engineruntime/managed_bundle_test.go#L62-L86>) | `LEVNOTE_MANAGED_REAL_SMOKE=1`; `LEVNOTE_MANAGED_REAL_SMOKE_RELEASE` is an explicit absolute **different release** | Import/activate one Markdown engine from an old release, export once; Reset then export once from current bundle adjacent to the test image. Requires old/new versions differ; exact HTML and source/engine metadata are checked. This is **Markdown only**. |
| [draw.io managed candidate test:23–28](<F:/AIproject/LevNote/native/core/engineruntime/drawio_managed_bundle_test.go#L23-L28>) and [preflight:136–206](<F:/AIproject/LevNote/native/core/engineruntime/drawio_managed_bundle_test.go#L136-L206>) | `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE=1`; absolute `..._CANDIDATE`, `..._EVIDENCE`, `..._CONFIG` | Test executable runs from a separate bundle **copy below evidence root**; absent JSON outside that copy; absent config below evidence root, nonnested with copy. Candidate/evidence trees separate; copy's draw.io bytes equal candidate. Fixed fourth-engine import/activate/selected-page SVG and process observation; **draw.io only**. |

The exact draw.io variable names are `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE`, `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE`, and `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG`. The current Markdown test prints its managed record into verbose output; the draw.io test creates a separate JSON. The entire bodies and cleanup contracts require final root review before use; this preparation read the admission/preflight contracts, not all544 draw.io lines.

Future command shapes (design only; use later reviewed exact paths, not shell interpolation):

```text
go test -c -o <fresh-copy>/engineruntime.test.exe ./core/engineruntime
<fresh-copy>/engineruntime.test.exe -test.run ^TestManagedRealOldReleaseThenBundledResetSmoke$ -test.v -test.count=1 -test.timeout=2m
<fresh-copy>/engineruntime.test.exe -test.run ^TestDrawioManagedRealBundleSmoke$ -test.v -test.count=1 -test.timeout=2m
```

No `./...` or regular `go test` temp executable is substituted for the adjacent shipped-worker bundle contracts. The integration executable may be reused across these two separately captured named commands only if root binds exact bytes and distinct contract-specific evidence/config.

## Remaining four-worker acceptance gap

These two tests do **not** exercise mindmap SVG and mindmap readonly scene requests. The [five-PE checker](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1>) validates those worker bytes/manifests but never launches them. Future complete real-worker acceptance must add/review explicit independent payload oracles for both mindmap engines and all four failure/owned-cleanup routes; this task neither implemented nor activated that product/test work.

Existing [worker resource test](<F:/AIproject/LevNote/native/core/engineruntime/worker_resources_test.go>) and [draw.io measurement harness](<F:/AIproject/LevNote/native/core/engineruntime/drawio_worker_resources_test.go>) are separate opt-in measurement protocols. They are not enabled as a cheap functional smoke, not claimed as whole-app low-resource acceptance, and not covered by this base CI draft.

No physical GUI, keyboard/mouse, OS clipboard/IME, visual rendering, host-session/WM_CLOSE or resource/full-delivery status is established by this plan.
