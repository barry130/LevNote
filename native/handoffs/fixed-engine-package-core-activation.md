# Local fixed-engine package core — activated after .8 acceptance

2026-11-09. ACTIVE. Assigned to GLM
read_native_handoffs_editable_engine_next_design_task_md. Root accepted the
exact .8 bundle in format-routing-delivery-root-acceptance.md. Implement this
finite core slice now; do not wait for another
source-only permission checkpoint. Publish actual signatures early while
continuing tests. No subagents, compact before200K.

Own ONLY new package-store source/tests in native/core/engineruntime plus
narrow shared bundle helper extraction if required. Existing Runner behavior,
wire/catalog workers, app/UI, scripts/docs and accepted package directories
remain unchanged. No full suite, actual worker/application launch or build.
After core acceptance Astra will own runner/app integration; don't implement it.

## Public contract

Use PackageManager, PackageReceipt, PackageFile, PackageStatus and APIs from
fixed-engine-package-next-design.md with these concrete root decisions:

- NewPackageManager(hostDir,stateDir) creates/checks a store rooted at
  stateDir/engine-packages. HostDir is only the known application's release dir,
  never a per-request executable override. Treat paths as absolute local disk.
- ImportFromReleaseDir(ctx,releaseDir,engine) returns an owned receipt and does
  NOT activate. Installed(ctx,engine) returns verified receipts plus meaningful
  per-package issues/error as needed, never silently omits a corrupt selection.
- Activate(ctx,engine,digest), Reset(ctx,engine), Snapshot() copied values.
  Exact64lowerhex digest; no version sorting. Persist only after verifying target;
  live selection updates only on successful atomic state write. On corrupt state
  return a usable manager with explicit load error and block engine resolution
  until explicit all-selection repair; don't silently use bundled. Absent state
  is normal bundled selection. Invalid startup state is not overwritten on read.
  Ordinary Activate/Reset must refuse while the entire state is corrupt rather
  than discard unreadable selections for other engines. Add RepairResetAll(ctx)
  only for a future explicitly labeled UI action that resets ALL3 selections to
  bundled, retaining installed packages. Success clears load error, write failure
  leaves it blocked. This does not claim recovery of the unreadable preferences.
- Expose no public generic executable path resolver. Add a private immutable
  selected-package DTO if needed for future runner integration; keep engine ID,
  selected digest, opaque version and exe hash distinct.

## Exact stored package

Directory engine-packages/<fixedEngineID>/<digest>/ contains:
manifest.json, catalog's fixed executable filename, original copied
release-manifest.json, and the following15 fixed authored/attribution files:
LICENSE, README.md, THIRD_PARTY_NOTICES.md;
attribution/SOURCE_THIRD_PARTY_NOTICES.txt;
attribution/mygo-README.md; attribution/purego-README.md;
attribution/mygo-v0.8.3-no-startup-cleanup.patch;
attribution/dependency-patches-README.md;
licenses/Go-BSD-3-Clause.txt; licenses/go-text-typesetting.txt;
licenses/mygo-MIT.txt; licenses/purego-Apache-2.0.txt;
licenses/x-image-BSD-3-Clause.txt; licenses/x-net-BSD-3-Clause.txt;
licenses/x-text-BSD-3-Clause.txt.

All15 are REQUIRED, not 'when present'; do not enumerate arbitrary inventory
paths or import other engines/host executable. The exact source manifest and
exe paths come from catalog.RelativePath, mapped to flat stored filenames.
Other15 relative paths are preserved, and the original release inventory is
copied unchanged. It intentionally lists original release files not stored here;
never describe it as the stored-package manifest or publisher authentication.

No extra global receipts.json: derive the bounded list from content-addressed
directories and verified files, with optional per-package receipt only if needed
for fast UI (not trusted over actual bytes). Maximum12packages per engine,
36total; no auto-delete or overwrite. Enforce limit before copying a new one;
exact existing package import can return same receipt after verification.
Staging uses a fresh exact random child of this private root. Only files created
by this import may be cleaned, and no computed recursive delete outside the
checked staging child. Keep older packages available for explicit activation.

## Admission and budgets

Reuse current strict engine manifest decoder and held Windows disk handles.
Every source and destination ancestor/file must reject reparse/links and type
mismatch. Fixed local volume paths only; reject UNC/device namespaces.
Release inventory <=2MiB/256entries, exact schema1/windows/amd64/v1 fields,
no duplicate/wrongcase/null/unknown keys, strict UTF8/JSON escape handling.
Each file entry path must be canonical slash-relative, no absolute/dotdot/drive/
ADS/duplicates (including case aliases); validate ALL entries syntactically,
but verify/copy only fixed selected18files. Inventory excludes itself; don't
attempt to verify its hash against itself. Record its actual hash in receipt.
Engine manifest<=16KiB; exe<=64MiB; combined15notices<=2MiB; state<=64KiB.
Hard read/stream caps even if file grows; state and manifest bounds before
allocating/decoding. Do not trust only stat. Retain existing32KiB hash/copy loop.

Digest: SHA256 of ASCII domain `LevNoteEnginePackageV1\x00`, then fixed-order
entries with big-endian count/pathByteLen/size and raw32byte file digest. Use
u32 count, u16 UTF8 path length, u64 size. Order is lexicographic stored relative
path bytes, includes copied release-manifest and all17other files. Engine ID is
already verified in manifest; optionally domain includes its bounded length.
Specify one exact algorithm in comment/test with independent fixed oracle.

Copy held validated handles into new staging with32KiB chunks, verify stream
hashes/lengths and sync each file; close owned handles needed before exclusive
rename to digest directory. Recheck locked parent identity as necessary, keep
private handle helpers narrow. Existing destination requires complete verified
equality; never replace it. Cancellation, failed publication or cleanup is visible;
none changes selected engine. Do not claim arbitrary adversarial filesystem or
executable sandboxing. Bound simultaneous store operations with mutex/context.

Keep the operation gate separate from the small live-selection lock. Snapshot
and LoadError must copy metadata quickly and never wait behind a64MiB import or
12-package verification scan, because the app reads them for rendering/admission.
Context cancellation must also interrupt waiting to acquire the I/O operation
gate; a plain mutex held for the whole copy without context wait is insufficient.
Do not cache a directory as verified forever; later activation verifies actual
bytes. Installed may return healthy owned receipts plus an error identifying
bad packages; don't suppress all useful choices because one package is corrupt.

## Meaningful tests and execution

Use small real-filesystem fixtures matching current release structure; no child
processes. Cover all three IDs, raw receipt copies and new/reopen/activate/reset,
import-only state, exact reuse, selection persistence failure/corrupt state,
unknown/wrongcase/duplicate/null inventory, traversal/case-alias paths, missing
required license, filesize/hash failures, cap refusals before large allocation,
link/reparse refusal where OS permits, cancellation/partial copy cleanup, package
limit retaining prior contents, bad selected package refusing fallback.
Prove no unrelated host/other-engine file is required to be hashed/copied.
No test should run an imported executable. Small fail-injection private seams
are permitted only for publication/cancellation failures that can't otherwise be
deterministically triggered; no exported runtime bypasses.

Run only new Package-prefixed tests in core/engineruntime and scoped vet first.
If shared bundle helpers change, run existing bundle-focused tests as appropriate,
not full fake-process/resource matrices. Keep actual stdout/stderr/exit codes in
one fresh evidence dir, wrapper MUST exit LASTEXITCODE. Record scoped before/after
owned inputs, preserve failures, publish source/result handoffs. Don't create a
large new generic validator. This core stage does not mark feature shipped.
