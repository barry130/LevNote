# Fixed engine store — Astra implementation transfer

2026-11-09. ACTIVE for astra_marker_view. Scene refresh source is accepted under
its root acceptance. GLM original broad store assignment was stopped before
any store source landed; it now owns ONLY package_inventory.go and its test.
You own the remaining store in core/engineruntime. No child delegation.

Read fixed-engine-package-core-activation.md once for exact public API/18fixed
files/digest/admission/state/cancellation/test decisions. Those apply fully;
the outdated assignee/header does not. fixed-engine-package-core-checkpoint.md
defines public API. Don't repeat the older conceptual design; its receipts.json
and variable inventory allowlist were rejected. GLM pure decoder source now
exists: parseReleaseInventory returns releaseInventory{Version,Files map},
releaseFile{Path,Size,SHA256}; names and caps in inventory activation. Don't edit
those2files; request fixes via root if needed. Root reviews them concurrently.

Implement new package_*.go/types/store/windows/state/verification and tests as
appropriate, avoiding inventory*. Don't change Runner/run/wire/catalog/app or
package scripts yet. You may narrowly extract bundle directory/handle helpers
if genuinely needed, keeping existing semantics/test scope. Windows only is
current target; non-Windows must compile or return clear unsupported behavior.
No real app/worker execution/build. Existing .8/.7 packages and prior evidence
remain immutable; source fixture bytes are harmless small exe placeholders.

Core requirements highlighted: hard limits before unbounded allocations,
stream/hash32KiB held files, all15 notices required + manifest/exe/release18,
exclusive publish to exact digest, cleanup only created staging paths, no global
receipt file, retained prior versions/12per-engine, Import not Activate, exact
copied receipts, actual verification for Installed/Activate, explicit corrupt
selection blocking and RepairResetAll, write failure never changes live state.
Separate cancellable I/O gate from short state read lock. Future runtime needs
private immutable selection+LoadError read atomically and held verifier, but no
Runner integration yet. Snapshot must not hash or block behind import.

Write source in sensible small chunks and publish actual private/public seams
to fixed-engine-package-store-checkpoint.md when usable; continue tests without
another idle gate. Only Package-prefixed tests plus engineruntime vet; if shared
bundle helper changed, affected Bundle tests as justified. No full resource/
process matrix. Capture actual stdout/stderr/exit and scoped before/after files;
wrapper exit LASTEXITCODE. Preserve failures; no large generic validator.
Finish fixed-engine-package-store-result.md. Root reviews actual source/results
then activates runtime/app. This is authorized continuation, no user question.
