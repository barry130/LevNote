# Confirmed volume-root defect: fix and fresh .10 delivery

2026-11-09. Root reviewed actual diagnosis, packageGuard.lockDir,
packagePathParts and NewPackageManager. DeepSeek
read_native_handoffs_resource_workload_next_task_md is sole implementation,
verification and delivery owner. GLM/Astra remain inactive. No children.
Compact before 200K context. This file supersedes the .9 delivery activation.

## Confirmed failure and scope

The real .9 managed attempt failed before a production worker request:
mkdir C:\\: Access is denied. Host/tester is on F:, normal TEMP and production
UserConfigDir are on C:. lockDir(create=true) calls Mkdir on the first unheld
volume root. Same-volume host/state fixtures mask this because the root was
already held by the first hostDir traversal. Do not change TEMP/APPDATA to evade
the issue. Diagnostic source was restored; .9 and all failure evidence stay
immutable and must not be offered as an accepted release.

Own a narrow correction in core/engineruntime/package_files.go plus meaningful
Windows regression tests. Traverse and hold the existing volume root without
ever attempting to create it. Open/validate existing ancestors; create only
missing descendants when create=true and their parent has already been held.
Keep cancellation, handle lifetime, reparse/identity/non-directory validation,
real filesystem errors and close cleanup semantics. Never reinterpret denied
access or an invalid existing component as an absent directory. No broad
filesystem abstraction or other source refactor.

Prove a fresh guard create=true against a normal existing drive root and missing
descendant succeeds (the old code should fail on this host), already existing
descendants are validated, create=false refuses missing descendants, and a file
in the ancestor chain is refused. Reuse existing link/path tests for protection.
Also verify NewPackageManager with host F:/state default C: where available,
without assuming all developer machines have F:. No arbitrary drive mutation.

Record the new regression failing on the old behavior when practical, then run
the affected package/managed tests and scoped vet through scripts/go.ps1
-GoArguments only. Preserve raw args/stdout/stderr/exit. Default opt-in real
smoke remains skipped in ordinary tests. A full repository suite or resource
matrix is not required. Publish the correction/result/source hashes to
handoffs/engine-package-volume-root-fix-result.md promptly so root can review.

## Delivery authorization after focused checks pass

Continue without waiting for another permission or activation: reuse the existing
finite delivery scripts in a fresh timestamped evidence directory and build
fresh absent bin/levnote-native-worker-20261109.10, version
native-worker.20261109.10. Preserve accepted .8 and failed .9 exact inventories.
Do not overwrite or retrofit either. New file hashes are real inputs, not an
excuse to rewrite the accepted app source inventory: explicitly record the
narrow runtime correction delta and link its evidence. App/rendering sources
are unchanged, so reuse the accepted four screenshots; no recapture needed.

One existing packaging invocation/four GUI builds, a complete isolated package
copy, then wrapper-compile the runtime tester once beside the copied executables.
Invoke the compiled tester with quoted runtime arguments:
-test.run ^TestManagedRealOldReleaseThenBundledResetSmoke$ -test.v -test.count=1.
Use normal default TEMP on C: and unchanged .8 as the source release. Exactly
two intended production worker requests: installed .8 Markdown then reset .10
bundled Markdown. Record actual counts if any attempt fails; retain all attempts.
Harness-only repairs do not need another build or tester compile.

Then one owned hidden .10 host startup with isolated config/session using the
existing WM_CLOSE helper, default bundled selection, unchanged Markdown bytes,
normal exit/no forced cleanup. For the normal UserConfigDir cross-volume case,
prefer an isolated run-owned config directory on the default C: temp volume
while the executable stays on F:; preserve/hash it as evidence, never use the
user's existing config. Validate resolved paths before any recursive cleanup.
No physical UI/IME/resource claim from this startup.

Adapt the existing compact aggregate rather than building another gate framework.
Compare all accepted app source hashes unchanged, allow only explicitly accepted
runtime fix/test files relative to the earlier broader inventory, and prove
post-fix build inputs unchanged through delivery. Sum bytes with an Int64 loop.
Reuse package validator facts, verify exact copied candidate, old .8/.9 unchanged,
notices, two managed outputs and one host exit. Publish actual .10 identities,
bytes and evidence to handoffs/fixed-engine-package-delivery.md. If a new product
defect occurs, preserve evidence and report the exact cause before broadening
source scope. Root reads and accepts; product docs follow actual acceptance.
