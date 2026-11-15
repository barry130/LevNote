# Workspace search snapshot correction — source acceptance, 2026-11-15

Status: source regression and both independent reviews passed; Windows package and whole-application delivery remain pending.

## Problem and correction

A workspace match retained offsets from the searched disk content. Editing and saving the open document made it clean again, so the previous dirty/version checks admitted the stale offsets. Searching `zero\nneedle\n`, saving `xxxxx\nzero\nneedle\n`, and activating the old result selected `\nzero\n` instead of preserving the current selection. An unopened same-length file could also change without changing its modification time and still be adopted using stale coordinates.

Workspace now creates a private shared identity from the exact decoded source supplied to the matcher: decoded byte length and SHA256. A match verifies the actual Buffer using its read-only WriteTo method. Matching coordinates and identity describe the same decoded text, including Unicode and line endings; no full source text is retained by results. Missing identity reports that a new search is needed.

The existing open and navigation paths carry that identity through asynchronous loading, duplicate detection and adoption. Checks around target/source Flush, Select and Reveal retain the existing root/search generation, tab/document/editor/buffer ownership, dirty-state and version protections. A stale result leaves the current selection, source bytes, undo/redo history and metadata intact. A valid newly opened result still reports the opened filename. The actionable FAR virtualization fixture now obtains a real Workspace result; its old UI actions and assertions remain.

Six other files received formatting-only cleanup. Neither test assertions nor production behavior changed in that cleanup.

## Frozen source and review

Base commit: `61952ac206c2eaea9ead51b7c5602e5fe87ed7c4`.

- Eight-file source manifest: `.localtools/workspace-snapshot-20261115/source-final.sha256`, SHA256 `9f213861fdfc73a05cf4472fdbbe86e773580fe7542441d66e42c4d47b59a236`.
- Complete 687-line source diff: `.localtools/workspace-snapshot-20261115/source-final.diff`, SHA256 `2dae45414274eda591100948e4516891dec5101110160783959e6755309f0b42`.
- Implementation evidence report: `.localtools/workspace-snapshot-20261115/REPORT.md`, SHA256 `40c31bdf0026e0cd6bd84f0fb3149a75400653fec8102f8d1adea82277ea7134`.
- Independent Standards report: `.localtools/workspace-snapshot-standards-20261115.md`, SHA256 `588a57d98b13b3a05a3e5b7d8aa15b84e51da0cf7647783e0b3c866aeb413729`; zero blocking findings, including the six formatting-only files.
- Independent Spec report: `.localtools/workspace-snapshot-spec-20261115.md`, SHA256 `c3a5cf9577be5b5e40e8e6a79d802bd4c344c0b686021f1f658b72f196345ac2`; zero findings. Root read both reports in full and accepted their common source binding.

Three actual behavior failures have unchanged-oracle passing counterparts: saved-clean selection, changed unopened content, and the success filename notice. Focused controls cover unchanged navigation, a later query edit, same-length changed content, re-search, absent identities, reentrant Flush, stale asynchronous adoption, streamed batches and decoded Unicode/BOM/CRLF. Earlier focused controls are evidence for their own source versions; the final full run below validates the final frozen source.

## Final local source gates

Pinned offline Windows amd64/v1, CGO0, Go1.27.3, vendor dependencies; six product roots only. All runs retained separate raw stdout/stderr, input inventories and exit receipts. No test, vet or build was repeated to conceal a failure.

| Gate | Result | Receipt SHA256 |
|---|---|---|
| `test -json -count=1 -timeout=300s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...` | exit 0 | `dab7181ad4c4e9b70f00f6e43dfe9daf7cb995f437e412dd40cbc6c0baebb711` |
| `vet` | exit 0; stdout/stderr empty | `28753ea57e14396deb675223388f895de85bd8d4cf2006cd7afd58a7b26707d4` |
| `build` | exit 0; stdout/stderr empty | `e7f7c393942885fa9b7c0aed5e0a188a271ed9cad5ff57fbae90c0944a72aa95` |

Receipts are `.localtools/delivery-20261115-{test,vet,build}-01/result.json`. Full-test stdout is 3,048,940 bytes, SHA256 `0b1402f5875f622e851cc28e19101897be8ac4dd6ba031004873ecb6d7a388af`; stderr is empty. Every gate confirms source and tool pins unchanged. Receipt PID identifies the wrapper; argv/environment are expected launch values, not captured OS Go command lines. The gate script uses its existing GOTMPDIR override.

The historical identity audit is `EXACT_UNION_CLOSED`: baseline 3019 PASS plus 118 additional identities from the latest affected suite gives 3137 prior PASS; the 15 new search nodes give 3152 current PASS. All 49 SKIP identities match the historical set, for 3201 RUN with one terminal event each. Missing, extra, duplicate, FAIL and new SKIP counts are zero; PASS and SKIP sets were compared separately by complete `(Package, Test)` identity. Package totals are 27 START, 26 PASS, one existing package without tests, zero FAIL. Expected/current identity-set SHA256 is `94e5cc8075d85da30013b086223dfab34e200659bd1214d607c4a368ccd04d28`.

Audit report `.localtools/workspace-snapshot-full-audit-20261115/REPORT.md` SHA256 `e88a45704707f9542a20256139fe7394e11e3df702e2aeee2da60bf34d5e37b7`; result SHA256 `1c8d571f0800e8d6fb6eaa76c3014ccf0361e33005f06e44aa6443a6a7fe578b`. Root read the complete 148-line analysis helper and 35-line report and checked the result summary. A first analysis failed on Windows path separator normalization before producing a result; that failure remains recorded, the helper alone was corrected, and the subsequent analysis exited 0. No product gate was repeated.

## Windows delivery state

The original packaging script was invoked once for new candidate `20261115.33`, output `native/bin/levnote-native-worker-20261115.33`, evidence `native/artifacts/packages/autonomous-local-windows-package-20261115-03`. It failed at `package-markdown-worker.ps1:24` while creating the evidence directory: `Access to the path ... is denied`. Exit 1, zero PE builds, zero worker launches and zero host launches. Original streams and input guards remain in `.localtools/delivery-20261115-package-33/`; stderr SHA256 `5a925368bcfc0fb3f22a4af945b74357de0dbfbd7ef51b6fe0938f01676873e7`. Source and tool pins remained unchanged.

The accepted `.29` package and historical failed attempts remain untouched. Creating or editing source files successfully does not establish permission to create children under the package evidence directory. No alternative writer, parent, ACL/owner change or elevated retry was used after the failure.

The current candidate still needs five-PE packaging, four real bundled workers and their cleanup paths, owned native-window close and second-launch/session restoration, physical Windows GUI/IME/clipboard/pickers, and resource measurements. Headless Tester controls and Go source gates do not establish those outcomes. GitHub CI remains disabled until development completes; no additional CNB disable requirement is implied. Whole application: **NO-GO** until those acceptance requirements are satisfied.
