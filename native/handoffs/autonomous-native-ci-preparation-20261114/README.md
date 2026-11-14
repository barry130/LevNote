# Native Windows CI preparation — Task20, inactive

Date: 2026-11-14. Owner: `windows-delivery`; activation/acceptance owner: `lead`.

**Preparation only.** This task created drafts exclusively in this directory. No parser/compiler, Go/test/vet/build/package, product process, network/download/install, MCP, Git, child delegation or CI activation was performed by the preparation owner. Task16 remains frozen and existing workflows/helpers/product/docs remain unchanged.

## Contents

- [Disabled CI proposal](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/native-windows-ci.yml.disabled>): outside workflow discovery, `.disabled` suffix, literal job `if: false`; manual-only contract proposal, pinned action references pending root review, no Node/Rust/legacy tests.
- [Guarded runner](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/run-native-ci.ps1>) and [integrity library](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1>): unexecuted code drafts; explicit root activation and frozen contract/pins required.
- [Input contract template](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/input-contract.template.json>) and [source inventory template](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/source-inventory.template.json>): empty/fail-closed, never runnable approvals. Root creates distinct complete per-checkout files later.
- [Conditional real-worker plan](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/REAL-WORKER-PLAN.md>): exact current Markdown/draw.io opt-ins and limitations; no false four-worker/full-delivery claim.
- [Documentation proposal](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/DOCUMENTATION-PROPOSAL.md>): current Task13/15 limited completion, T05/Task19 pending and legacy-vs-native evidence separation; no edits to existing docs.

## Fixed toolchain/vendor inputs

The local [official archive metadata](<F:/AIproject/LevNote/native/tools/go-toolchain.json>) records `go1.27.3.windows-amd64.zip`, **79010309 bytes**, SHA256:

```text
f26081f438319757f8a1b46d7d0b2c5ec7583d4038e485e0aa725cfde7885fff
```

The [complete patch manifest](<F:/AIproject/LevNote/native/dependency-patches/mygo-v0.8.3-patch-files.json>) must remain schema1 / `github.com/egoist/mygo v0.8.3` / exactly26 members and SHA256:

```text
9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a
```

Every manifest member is rehashed; the startup marker, disabled cleanup call and fixed module are also checked. Hashes bind bytes, not reviewer truth, publisher authentication, signatures, sandboxing or acceptance. Preparation read these local records; it did not redownload/reverify an archive.

## Intended later runner behavior

1. Refuse absent `-ExecuteReviewed`; require Windows x64 PS7, exact reviewed contract/hash, true root assertions, real source/review/helper/runtime pins and complete shipping-source inventory including untracked files. No current source pin is fabricated while source changes concurrently.
2. Require a **new disposable checkout** with absent native tools/cache directories and ordinary absent direct evidence child below `native/artifacts/ci`. No deletion/reuse of current checkout or historical artifacts. Root provides the existing evidence parent and reviewed input files before activation.
3. Verify archive size/hash before extraction. Validate canonical paths/no aliases/symlinks; extract complete official Go tree into the existing packaging wrapper's fixed `native/tools/go` path. Use no system/raw Go or dynamic bootstrap.
4. Use project-local caches/module-cache/tmp and telemetry mode file; isolate C: profile/config/temp under an absent GUID directory. Child env clears inherited Go selectors and every `LEVNOTE_*` opt-in, fixes Windows/amd64/v1/CGO0 and offline vendor. Runtime env flags are not an OS egress firewall; normal runner policy must separately prevent product-code network access.
5. Read-only reuse of pinned [Task16 owned-process helper](<F:/AIproject/LevNote/native/handoffs/autonomous-local-windows-delivery-20261113/owned-host.cs>); compile only in the future authorized run. Each command starts suspended, joins a kill-on-close Job before resume, captures exact image/time/PID, has a wall-clock bound, retains complete streams/exit/errors and demands Job active0. This is command ownership, not GUI/IME automation; no WM_CLOSE/input APIs are invoked here.
6. Six serialized command gates: actual `go version`; existing `check-build-guard -CaseSet All`; six-root test; six-root vet; six-root build; fresh fixed packaging. Failure stops progression, with no retry. Existing guard Junction semantics remain intact; unsupported token/runner capability is FAIL, not a fixture fallback.
7. Independently check host+all four GUI worker PEs (machine8664 / PE32+020B / subsystem2), fixed distinct engine/format/version manifests, package-result byte/hash/path bindings, all25 shipped files plus self-excluded release manifest (26 total), attribution/license source copies, generated README and local links. Never substitute the old three-engine fake-byte asset checker.
8. Require complete source/vendor/helpers/archive/runtime/toolchain and package postguards. Retain caches, C: config and raw failure evidence; no cleanup/deletion script. Only a future fully successful limited run may emit `PASS_LIMITED_NATIVE_CI`, never application acceptance. Tests preserve skip details; real bundle opt-ins/physical GUI remain NOT RUN.

Fixed product patterns are `./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...`, not repository `./...`. `CGO_ENABLED=0` matches the current local lane, not another platform/default-CGO acceptance assertion.

## Root contract preparation and activation checklist

- Review final two new PowerShell sources statically, parse them with the verified PS7, and review pinned Task16 owned-process helper provenance. Preparation supplies no parse result for Task20.
- Independently freeze the final current code/docs/assets. Serialize `SourceInventory` output as schema1 with actual current `repositoryRoot` and every `path/absolutePath/bytes/sha256/lastWriteUtcTicks`. Inventory must be generated in the **same final disposable checkout** after checkout normalization, not copied with stale absolute paths/mtimes.
- Complete a new input contract: that exact inventory/review file and SHA256; final new runner/library hashes; actual PS7 absolute executable/version/hash; exact archive absolute path/hash/size; repository; absent evidence path; new `ci.*` package identity; command bound60..1200s. Read it back and independently pin its final bytes. Template false/empty fields are not permission.
- Provision the proposed disposable Windows runner pool and offline archive/PS7 from separately reviewed official artifacts. Proposed YAML labels/actions are not created/verified by this task; review action IDs and hosted/self-hosted platform support before installing any active CI.
- Run genuine supported fresh local/native gates and review outcomes before deliberate CI activation. Parent-reported previous Junction Access denied is preserved and cannot be upgraded by this draft. No permission escalation, copy fallback, alternate shell or production helper modification is proposed.
- Keep candidate source/package identity, all known skips, conditional worker status and required final Windows/session/resources gates explicit. Add reviewed current documentation through root, without rewriting historical evidence.
- Only after successful reviewed results may root deliberately install the draft under workflow discovery/remove the literal false guard. Manual contract/provisioning lifecycle must be implemented before adding automatic push/PR triggers. Merely renaming the draft is insufficient.

Future invocation shape (design only):

```text
<verified-pwsh> -NoLogo -NoProfile -NonInteractive -File <this-directory>/run-native-ci.ps1 -InputContractPath <reviewed-contract> -InputContractSHA256 <actual-hash> -ExecuteReviewed
```

Official provisioning/checkout/artifact upload are CI transport and may use network outside product gates; product runtime must stay pure-local. This draft does not pretend the whole GitHub runner job is globally air-gapped, and does not install a firewall or upload evidence now.

## Current root-reported local gate status

Root reports current whole-product native test/vet/build succeeded on final1211-file source freeze02. The fresh existing package call then failed at its initial evidence-directory `New-Item` with Access denied/exit1, before any GUI build; both intended output/evidence targets remain absent. No new PE/worker/host exists from that attempt. See [documentation proposal](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/DOCUMENTATION-PROPOSAL.md>) for root-reported pins and preserved failure. Task20 does not independently revalidate those raw results, does not import these pins into empty CI templates, and does not request a retry/fallback. Root's parse-only review of these new drafts remains next.

## Not delivered by Task20

No active CI; no source/helper/runtime PASS for these drafts; no full four-worker request matrix; no physical GUI/OS clipboard/IME; no actual current-candidate host/session/WM_CLOSE/second launch; no resource workload; no replacement accepted runnable or dual-remote delivery. Current accepted .29 and closed historical namespaces stay immutable.
