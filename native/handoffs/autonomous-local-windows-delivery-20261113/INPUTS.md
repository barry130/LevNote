# Complete named input contract (schema1)

[Runner](<run-host-smoke.ps1>) accepts only InputContractPath,
InputContractSHA256 and the explicit ExecuteReviewed switch. The UTF-8 JSON
contract is separately root-created and SHA256-pinned; no command/argument
injection, source package selection or executable override is accepted.
[Input template](<input-contract.template.json>) deliberately contains false
booleans and invalid REQUIRED_* strings: it is **not executable authority**.
Never overwrite this preparation template to create runtime evidence.

## Contract members

| Member | Required value and root responsibility |
| --- | --- |
| schema | Integer1. |
| executionAuthorized, sourceFrozen, packageFrozen | JSON booleans true only after exact root review/freeze; numeric1/stringtrue do not qualify. |
| repositoryRoot | Ordinary absolute local Windows repository path; this assignment uses F:/AIproject/LevNote. |
| candidateDirectory | Reviewed fresh complete package, direct native/bin child; not accepted .29 or closed .30. No runtime writes to candidate. |
| version | Exact current manifest/package-result version, 1..64 ASCII alphanumeric/dot/underscore/plus/hyphen; not the closed .29/.30 versions. |
| evidenceDirectory | Absent ordinary direct autonomous-local-windows-* child of native/artifacts/acceptance or native/handoffs; no existing/consumed directory and no overlap with candidate/helpers. |
| expectedWindowTitle | Exactly 轻笺 · 原生版; only exact held PID/title/hidden HWND is eligible for close. |
| readinessMilliseconds | 1000..20000; template20000. |
| settlingMilliseconds | 5000..10000, monotonic measured alive duration; template5000. |
| closeExitMilliseconds | 1000..15000; template15000. Cleanup is separate fixed10000ms host join. |
| powerShellRuntime | Object with absolute path, real64-hex sha256, exact PSVersionTable.PSVersion version string. Must match held runner process image/version; root records x64/native/.NET readiness. |
| helpers | Exactly3 unique name/real64-hex sha256 entries: run-host-smoke.ps1, delivery-lib.ps1, owned-host.cs from this prepared directory. Root hashes final unchanged files before parser/compiler review and execution. |
| sourceInventory | Ordinary preexisting path + real sha256 of complete frozen shipping-source schema below; not a Go package selection snapshot. |
| sourceAcceptance | Ordinary preexisting path + real sha256 of root exact-source acceptance. Root checks current complete source identity, actual Windows tests/vet/build receipts, failed raw evidence and unresolved issues; harness checks hash only. |
| rootReview | Ordinary preexisting path + real sha256 of root helper/runtime and single host activation review. Must bind helper pins, runtime review, source/package accepted inputs, fixture semantics, fresh evidence root and nonclaims. Harness checks hash only. |
| packageRecord | Ordinary preexisting path + real sha256 of current successful packaging result, binding exact version/target/output directory/host and four worker+manifest hashes; see below. No packaging/build performed by this runner. |
| packageManifestSHA256 | Real sha256 for candidate release-manifest.json; it must enumerate exact payload files/count/size/hash, excluding itself only. |
| textFixture | Ordinary preexisting path + real sha256. Root reviews draw.io text capable of diagram-notes/builtin.drawio preview, UTF-8 BOM, CRLF only, nonempty≤4MiB. Runner copies unchanged to route-gate.diagramnote; fixture signature alone does not prove preview rendering. |
| pngFixture | Ordinary preexisting path + real sha256 of root-reviewed real PNG,33B..16MiB. Runner checks PNG signature/first IHDR only; copies unchanged to image-fixture.png. |
| diagramNotes | Ordinary preexisting path + real sha256 of reviewed data-only schema3/plugin diagram-notes, apiVersion1, sole extension.diagramnote, nativeRoute builtin.drawio. Root reviews name/version as well. Installed copy must remain exact and is never executed. |

Every path+hash object uses keys `path` and `sha256`. Hashes must be real64
hexadecimal characters, not template strings or hashes invented from metadata.
All named inputs remain outside the fresh evidence root. The contract, helper
files, runtime executable, review receipts, source inventory, package result,
manifest, fixtures and source plugin are fingerprinted before and after the host.

## Complete frozen shipping-source inventory

[Source template](<source-inventory.template.json>) has schema1, repositoryRoot,
directories, rootFiles, files. The templates intentionally omit actual inventory
entries and cannot pass. Each file entry needs exact repository-relative POSIX
path, byte length, sha256, **lastWriteUtcTicks** (UTC DateTime.Ticks, NOT Unix time
or FILETIME). No leading slash, backslash, dot/dotdot component, colon or control
character; exact path spelling and case are checked.

Minimum directory scopes (each is recursively scanned, including hidden and
untracked regular files, rejecting reparses):

```text
native/app
native/cmd
native/core
native/formats
native/modules
native/ui
native/vendor
native/dependency-patches
native/scripts
native/portable
native/examples
native/licenses
```

Minimum explicit rootFiles:

```text
LICENSE
native/go.mod
native/go.sum
native/THIRD_PARTY_NOTICES.md
```

Root may add reviewed nonoverlapping source/asset directory scopes and extra
regular root files needed for current package provenance. All present files
within declared scopes must appear exactly once in files, and vice versa; added
files/count changes, duplicate paths, changed bytes/hash/mtime and reparses fail.
The runner rejects directory scopes in native/artifacts, native/bin,
native/handoffs, native/tools or .localtools to avoid consuming outputs,
historical evidence or toolchain as shipping source. Existing product+vendor
snapshot is **not** this contract; root must create the expanded shipping-source
inventory only after ongoing source/package-related work has frozen.

## Package/review receipts

Current result shape derives from
[current packaging source](<F:/AIproject/LevNote/native/scripts/package-markdown-worker.ps1>):
Status exact `PACKAGED; build and asset checks passed, product acceptance is separate`,
Version, TargetGOOS windows, TargetGOARCH amd64, TargetGOAMD64 v1,
OutputDirectory, HostBytes, HostSHA256, ReleaseManifestSHA256, and exactly4 Engines.
Each engine has unique Engine, correct Format and RelativePath, exact WorkerPath,
WorkerBytes, WorkerSHA256, ManifestPath and ManifestSHA256. Runtime inventory,
five PE headers and four engine manifests are independently checked. Root also
reviews actual single packaging invocation/five build raw exits and source
receipt relationship; a hash-consistent fabricated result is not acceptance.

[Current release manifest source](<F:/AIproject/LevNote/native/scripts/portable-package-lib.ps1>)
requires schema1/version/windows/amd64/v1/configuration `%APPDATA%/LevNoteNative`,
files path/size/sha256 covering complete payload and excludes sole
release-manifest.json. Four engine manifests bind schema1/protocol1/current
engineVersion/filename/size/hash/windows/amd64 and fixed format IDs. Exactly five
.exe files are allowed; all expected fixed paths must exist as amd64 PE32+ GUI
subsystem2. No engine request is made in this gate.

Review receipt file formats themselves are intentionally not parsed as authority.
Root must read them and decide they actually bind the final source/helpers/runtime/
package/fixture inputs. Once frozen, SHA256 checks only prevent unnoticed changes.
There is no assumption that historical source acceptance or package .30 preparation
activates this fresh candidate. Root source acceptance, package acceptance and
explicit process activation are separate decisions.

## Retained evidence and failure boundaries

Fresh evidence retains inputs-before, launch-intent, exact process receipt,
owned app stdin/stdout/stderr, pre/postclose session bytes/facts, postguard
inventories and result.json with original run and cleanup/postguard errors.
The actual GUID C: config path is recorded and retained with complete inventory.
Binary/session/config files may contain sensitive local data; no upload occurs.
Session preferences permit only the current documented keys and require
boolean wordWrap; selection permits only nonnegative integer Anchor/Head.
Controller startup/pre-evidence failure and external interruption require the
root caller receipt. An absent final result is not a PASS or an empty-Job proof.
See [README limits](<README.md>) before interpreting any actual result.
