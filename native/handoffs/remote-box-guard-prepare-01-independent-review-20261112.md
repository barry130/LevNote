# Independent evidence review: remote guard prepare 01 (2026-11-12)

Reviewer: `guard-evidence-review`; shared `task-3`, claimed from pending revision 1.
Target: **authorized main box**, through the already connected authorized connector.
This report deliberately records neither the real box identifier nor a connection URL.
The DSH lead remains the main responsible party; the original Codex thread
`01a1b6b1-3caf-76b1-81e2-183c090b7b86` is the historical baseline, not an execution request.

## Decision

**Recommend one unmodified, guarded diagnostic test only after explicit lead activation.**
This is a review recommendation, not a product-stage activation or a product pass.
The prepared receipt is genuinely `GUARD_READY_NOT_PRODUCT_ACCEPTANCE`.
No duplicate test/vet attempt was found at the recorded observation time.
The existing nonverbose test cannot establish absence of MyGo fallback; Linux cannot
close Windows-specific tests, physical IME, or `.30` delivery. Those gates remain held.

The old review's complete InputIdentity and ToolchainIdentity are correct; no
prefix was completed or guessed. Its backend-enforcement conclusion needs the
clarification in the corrections section below.

## Scope and current instructions

Read the current head of [native coordination](<F:/AIproject/LevNote/native/AGENTS.md#L14-L24>),
the [old root review](<F:/AIproject/LevNote/native/handoffs/remote-box-guard-prepare-01-root-review-20261111.md>),
[runbook 02](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/remote-runbook-02.md>),
and [transfer identities](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/transfer-files-02.json>).
The current instruction requires this independent review before `task-4` and
explicit lead activation; a passed test must be reviewed before any separate vet activation.

Only read-only directory/process metadata, named evidence files, JSON comparisons,
and SHA256 calculations were performed remotely. Local work was text inspection and
this new handoff. No Go/list/version/test/vet/guard invocation, installation,
upgrade, product launch, local acceptance, assertion change, skip, commit or push
was performed by this reviewer. Original preparation inputs, old evidence, old
review, coordination file, and `.29` were not modified.

The local and box git clones being `23054e7` is the reconciled delivery baseline,
not the product execution source. The prepared execution source is the separate
`/home/user/levnote-remote-20261111-01/source/native` tree. Static source observations
below use the retained frozen snapshot 02, not a claim that the clone was tested.

## No previous or concurrent product phase found

At **2026-11-12T02:11:27.212205+00:00**, the real run root contained only
`evidence`, `guard-ready.json`, `source`, `state-prepare`, `toolchain`, and `transport`.
Its evidence directory contained only `prepare-01`; `test-01` and `vet-01` did not exist.
The caller directory contained only the five prepare capture files.

A read-only `/proc` metadata scan found no `go`, `compile`, `vet`, `link`,
`*.test`, `levnote*`, or other Python process apart from the audit reader itself.
It did not read process arguments or process environment. This is a point-in-time
observation, not an unlimited historical guarantee; the execution owner must check
again immediately before any authorized single phase and never repeat an existing attempt.

The real [prepare result](</home/user/levnote-remote-20261111-01/evidence/prepare-01/result.json>)
records `Phase=prepare`, `ProductAttempted=false`, `ProductExecuted=false`,
`CandidateSuccess=true`, `PostGuardErrors=[]`, and `WrapperExitCode=0`.
The receipt independently records `ProductExecuted=false`.

## Full identities checked against real evidence

SHA256 calculations returned successfully at **2026-11-12T02:19:08.148124+00:00**.
The real [guard receipt](</home/user/levnote-remote-20261111-01/guard-ready.json>) is **723 bytes**:

```text
Receipt SHA256:   da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853
InputIdentity:    96e0a103b8538f9810333082ed05217e47b1360e963a7dbc8dbef0585654e12d
ToolchainIdentity:9806db80da121ddc6891941ef94ce2da8e15855f15af3ee0e1e33eb51249f7b9
```

InputIdentity comes from the actual receipt and prepare result; ToolchainIdentity
comes from the actual result's ToolchainBefore/After, not an invented receipt field.
Both were independently recomputed from the actual before inventories, sorting Path
by UTF-16 ordinal order and hashing the UTF-8 records
`Path + TAB + decimal Length + TAB + lowercase Sha256 + LF`, exactly as the
[fixed wrapper defines](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/linux-guarded-validation-02.py#L132-L138>).
The recomputed 64-character values exactly match the real result and the
[old review's complete values](<F:/AIproject/LevNote/native/handoffs/remote-box-guard-prepare-01-root-review-20261111.md#L25-L31>).

| Actual before/after evidence pair | Files / bytes | Comparison and raw JSON SHA256 |
| --- | --- | --- |
| [inputs-before](</home/user/levnote-remote-20261111-01/evidence/prepare-01/inputs-before.json>) / [inputs-after](</home/user/levnote-remote-20261111-01/evidence/prepare-01/inputs-after.json>) | 1,208 / 17,045,848 | Parsed JSON and raw bytes equal; both `8a522eef4901375db812d97f20f7c030b62c5821d5c4900f481c7d0deb50a881` |
| [toolchain-before](</home/user/levnote-remote-20261111-01/evidence/prepare-01/toolchain-before.json>) / [toolchain-after](</home/user/levnote-remote-20261111-01/evidence/prepare-01/toolchain-after.json>) | 15,657 / 244,139,206 | Parsed JSON and raw bytes equal; both `ea06e7739c3743026d540ab75b503dd152f48467f2c1429145f19d45ea81dffc` |

The result records 5 frozen picker files and 26 patch files before and after.
It records Go binary length 17,164,222, SHA256
`5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b`,
VERSION content `go1.27.3`, and `Executed=false` in both toolchain summaries.
This reviewer did not execute Go to inspect its version.

The actual transferred inbox files were hashed, rather than trusting abbreviated
hashes in prose. All lengths and complete hashes matched the fixed transfer list
and official archive pin in the runbook:

| File | Actual bytes | Actual SHA256 |
| --- | ---: | --- |
| [source ZIP 02](</home/user/levnote-inbox-20261111-01/levnote-source-inputs-20261111-02.zip>) | 5,227,577 | `457503deecbcf0a534cc3240027048454bdf758076e0ee8d3a0a64f7773db7f9` |
| [raw manifest 02](</home/user/levnote-inbox-20261111-01/inputs-02.json>) | 244,715 | `a51734c0e9b40051ad1ba06ad4b5e0acc4d92ef1a3de5261259fa435b5df2e51` |
| [wrapper 02](</home/user/levnote-inbox-20261111-01/linux-guarded-validation-02.py>) | 33,772 | `7d53c6b7374a52c9d3e6749e423449d073778ac1c33c49888a121b97d43e17d7` |
| [Go Linux archive](</home/user/levnote-inbox-20261111-01/go1.27.3.linux-amd64.tar.gz>) | 70,588,678 | `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40` |

Raw manifest SHA256, aggregate input identity, and raw evidence-JSON hashes are
different identities and are not substituted for one another. Equal saved inventories
establish the prepare observations; they do not replace the future phase's live
source/transport/toolchain pre/post checks. No new full-tree guard was run in this audit.

## Caller capture, invocation and isolated environment

The real caller capture has [UTC before](</home/user/levnote-remote-caller-evidence-20261111-01/prepare-utc-before.txt>)
`2026-11-11T09:02:23Z`, [UTC after](</home/user/levnote-remote-caller-evidence-20261111-01/prepare-utc-after.txt>)
`2026-11-11T09:02:47Z`, [outer exit](</home/user/levnote-remote-caller-evidence-20261111-01/prepare-exit.txt>) `0`,
[stdout](</home/user/levnote-remote-caller-evidence-20261111-01/prepare-stdout.txt>) one 169-byte JSON line
reporting prepare/guard-ready/exit 0, and [stderr](</home/user/levnote-remote-caller-evidence-20261111-01/prepare-stderr.txt>) zero bytes.
Internal result UTC is `09:02:25.417711+00:00` to `09:02:47.952204+00:00`.

The actual [invocation](</home/user/levnote-remote-20261111-01/evidence/prepare-01/invocation.json>)
selects wrapper 02, `prepare`, the authorized main box, the original inbox and the
original run root; the wrapper identity matches the pin. Platform is Ubuntu 24.04.5
Linux x86_64, UID 1000, Python 3.12.3; display socket handling is metadata-only.
`CredentialFilesRead=false`, four ambient names were recorded, and
`NoAmbientVariableValuesCopied=true`. No credential store or token value was read.

The saved [clean environment](</home/user/levnote-remote-20261111-01/evidence/prepare-01/environment.json>)
uses the run-root-owned toolchain, HOME/XDG/cache/temp/GOPATH/telemetry directories,
restricted PATH, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOENV=off`, empty GOFLAGS,
`GOOS=linux`, `GOARCH=amd64`, `CGO_ENABLED=0`, `GOPROXY=off`, `GOSUMDB=off`,
`GOVCS=*:off`, `GOTELEMETRY=off`, C.UTF-8, UTC and explicit display `:0`.
It is the documented CGO0/offline Linux configuration, not default-CGO or Windows equivalence.

## Saved prerequisite evidence, not backend proof

The actual [prerequisite JSON](</home/user/levnote-remote-20261111-01/evidence/prepare-01/prerequisites-before/prerequisites.json>)
exactly equals the result's PrerequisitesBefore. The same-key Libraries/Ldconfig/FcMatch
fields exactly match [before-load identities](</home/user/levnote-remote-20261111-01/evidence/prepare-01/prerequisites-before/library-identities-before-load.json>).
All eight recorded roles are available: gobject, pango, glib, pangocairo, cairo,
fontconfig, harfbuzz and pangoft2; six required and two optional. Resolved libraries
have recorded lengths and SHA256 under `/usr/lib/x86_64-linux-gnu`.
There are 78 mandatory symbols and `FontMapAndContextCreated=true`.

The saved [ldconfig exit](</home/user/levnote-remote-20261111-01/evidence/prepare-01/prerequisites-before/ldconfig/exit.json>)
and three [font-1](</home/user/levnote-remote-20261111-01/evidence/prepare-01/prerequisites-before/font-1/exit.json>),
[font-2](</home/user/levnote-remote-20261111-01/evidence/prepare-01/prerequisites-before/font-2/exit.json>),
[font-3](</home/user/levnote-remote-20261111-01/evidence/prepare-01/prerequisites-before/font-3/exit.json>) exits
all have Spawned true, ReturnCode 0, no signal/error/cleanup errors and empty stderr.
Their stdout sizes are 46,739 / 60 / 69 / 72 bytes. Font records are DejaVu Sans,
DejaVu Sans Mono and Noto Sans CJK SC. These are observations from the saved prepare
probes; no ldconfig/font/ctypes probe was rerun by this reviewer.

Crucially the evidence explicitly states `ProbeProcess=Python ctypes, separate from Go/MyGo`,
`MyGoBackendObserved=false`, `PhysicalUIOrIMEObserved=false`, and warns that a successful
nonverbose test can suppress logs. This is not a hermetic OS closure or an observed
MyGo text backend.

## Corrections or clarifications to the old root review

1. **No identity correction.** The complete values in the [identity paragraph](<F:/AIproject/LevNote/native/handoffs/remote-box-guard-prepare-01-root-review-20261111.md#L22-L31>)
   are exact real evidence and recompute correctly. The raw receipt hash also matches.
2. **Backend conclusion must remain open.** The [conclusion at 57–63](<F:/AIproject/LevNote/native/handoffs/remote-box-guard-prepare-01-root-review-20261111.md#L57-L63>)
   already acknowledges a caveat, but calling the fallback-log check an enforcement point
   must not imply complete detection. The [fixed test vector](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/linux-guarded-validation-02.py#L50-L54>)
   is `test -mod=vendor ./app -count=1`, without `-v`. The [check](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/linux-guarded-validation-02.py#L549-L556>)
   rejects visible `text will not show` bytes only. Successful package output may hide
   that warning, so absence of the substring and even COMMAND_PASSED_WITH_GUARDS do
   not prove that MyGo did not fall back. No wrapper edit, verbose rerun or added probe
   is authorized by this review.
3. **Current activation is diagnostic and manual.** The [old authorization](<F:/AIproject/LevNote/native/handoffs/remote-box-guard-prepare-01-root-review-20261111.md#L65-L82>)
   is subject to the latest lead/task ordering. Test first, once, only after explicit
   activation; inspect real result and raw child exit/streams. Vet is not automatic:
   require a passed test plus lead review and separate activation. Any failure holds
   vet and all further attempts; preserve the failed evidence/run root. No fresh run
   or retry is authorized here.
4. **Keep concrete platform blockers explicit, not greened away.** Static frozen source
   has C:-path/backslash equality expectations in [reference tests](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/reference_test.go#L31-L49>),
   while [resolution](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/reference.go#L55-L79>)
   and [file URLs](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/reference.go#L146-L158>)
   use host-dependent filepath normalization/absolute-path checks. Linux slash/drive
   semantics therefore have known static incompatibilities. The [recent-close test](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/reopen_closed_test.go#L266-L280>)
   uppercases a real temporary path before reopening, assuming Windows-style case behavior
   that a case-sensitive Linux filesystem does not provide. These are predicted
   platform issues from source, not an invented observed test failure.
   Four frozen app tests are Windows-tagged and absent from the Linux suite:
   [drawio capture](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/drawio_worker_stage2_capture_test.go#L1>),
   [engine capture](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/engine_packages_capture_test.go#L1>),
   [engine store](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/engine_packages_store_test.go#L1>),
   [engine lifecycle](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/snapshot-02/native/app/engine_packages_lifecycle_test.go#L1>).
   Their automatic platform exclusion is not a newly authorized skip and is not test coverage.
   Do not add skips, change assertions, normalize away failure, or narrow the fixed suite.
5. **No Linux substitute for `.30`.** The [Windows limitation](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/remote-runbook-02.md#L88-L95>)
   still requires Windows GUI PE, five GUI builds, C: configuration, WM_CLOSE and
   Windows process/Job cleanup. Linux test, cross-compilation and Wine do not replace
   those gates; `.29` remains the accepted immutable runnable.

## Handoff to lead

Review this report, then decide whether to activate `task-4` for the single exact
fixed diagnostic test on the prepared run root, with the reviewed receipt SHA256
and existing `picker-final-app-linux-20261111` interlock. The execution owner must
recheck absence of prior/concurrent attempts and keep the source, wrapper, assertions,
manifest, 26 patches and toolchain unchanged. No product call was made by this reviewer.

A green result may be considered guarded Linux command evidence only, never backend,
Windows IME or `.30` acceptance. Failure stops the phase; no vet, retry, skipped test
or repaired assertion follows without a new explicit decision. The old root review
is preserved unchanged; this new report supplies the independent correction layer.

Tool caveat: one read-only comparison query exited 1 because an array was initially
treated as an object; the failure was immediately reported, no product or write
occurred, the failed batch was not repeated, and a schema-confirmed narrow calculation
provided the successful hashes above. A whole-object-versus-array library comparison
was likewise not treated as an evidence mismatch; comparison of the three correct
same-key fields returned true. Incidental history-index/context-cleanup failures
were reported and not retried. The authorized remote connector and named evidence
reads succeeded.
