# Fixed engine package core — root acceptance

2026-11-09. Accepted as a source-level prerequisite for managed runtime work.
Root reviewed production sources, focused test bodies, actual raw results,
command wrappers/correction and the independent read-only review. Root did not
implement or execute tests. This is not yet connected to the shipped .8 app.

Inventory decoder: corrected actual focused run PASS 0.725s, scoped vet exit0;
the original exact-regex invocation ran no tests and is retained as such. The
original commands.txt is not rewritten; commands-correction.md/json identify the
actual passing command. Frozen decoder/test SHA values are recorded in
fixed-engine-package-inventory-result.md and match the store fingerprints.

Store: attempt01 covers10 top-level cases PASS3.145s, vet exit0; followup02
extends the cancellation oracle and adds healthy-list/bad-selection coverage,
two selected cases PASS1.024s, vet exit0. Together11distinct top-level cases.
Both gates record18 unchanged scoped inputs. The production files stayed fixed
between gates. Hardlink refusal passed; both symlink cases skipped for Windows
privilege and are not represented as verified behavior. All fixture executables
are inert; no imported process, application build, or resource measurement ran.

Accepted behavior: fixed3catalog IDs, exact18stored files, independent digest
identity, bounded streaming reads/copy, import separate from activation, reuse
with fresh verification,12retained-package cap, atomic durable selection,
explicit corrupt-state repair and no hidden fallback. Metadata snapshots do
not wait for the I/O gate. The independent review found no concrete defect in
its specified state/publication/cleanup/metadata responsiveness slice.

Runtime/app must retain the disclosed boundaries: one Manager serialization,
not cross-process state merging; stage handles close for rename/cleanup; no
hostile-filesystem or publisher-authentication claim. A selection rename commits
before a later cleanup error: an error does not universally imply rollback.
Installed list errors may coexist with healthy receipts and must stay visible.

Evidence and actual source/result hashes:
- handoffs/evidence/fixed-engine-package-inventory-20261109-051510/
- artifacts/acceptance/fixed-engine-package-store-20261109-131851-attempt01/
- artifacts/acceptance/fixed-engine-package-store-20261109-132244-followup02/
- fixed-engine-package-store-fingerprints.json
- fixed-engine-package-core-independent-review.md

Next: activate only the finite managed-runtime slice. Accepted .8/.7 packages
and historical evidence remain immutable. App integration follows the runtime
checkpoint; source tests/binaries are not described as delivered UI features.
