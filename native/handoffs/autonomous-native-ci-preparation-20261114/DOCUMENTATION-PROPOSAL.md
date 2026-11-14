# Current delivery documentation proposal — no existing docs edited

Date: 2026-11-14. Task20 draft for lead review. The paragraphs below are proposed current-status additions, **not** an acceptance record. No existing README, validation record, workflow, product source, packaging helper or frozen Task16 file was edited by this owner.

## Proposed status addition

> The current development line is native Go1.27.3 + MyGo Surface and project-authored engines on local Windows. The latest accepted runnable remains `native-worker.20261110.29`; new autonomous source is not contained in that immutable package. Current source/test completion, command PASS, new package creation and application delivery acceptance are separate states.
>
> Task13 Markdown main-document history/List lifecycle and Task15 Windows retry-conflict/publication-failure semantics have limited source and affected-suite completion with root-reported independent evidence checks. Those focused results do not prove the entire native application, all workflows, physical input/IME, new package/session behavior or performance. Historical Task15 F1–F3 are corrected in current source; do not carry the old audit's pending defect status forward after corrections. Publication is still not filesystem CAS, and actual failure/draft status must remain explicit.
>
> T05 Copy All remains pending final Task19 correction/review at this preparation snapshot: application identity, export occasion and oversized UTF-8-safe output budget/disclosure are required. Do not replace this statement with completion based only on the older partial implementation or an agent's unsourced GREEN summary. Lead owns the next reviewed current-state update; concurrent Task19/21 changes are not silently included in this draft's input identity.
>
> Native CI restoration is preparation-only. A disabled Windows draft and guarded runner exist; no active workflow has been installed. Future CI must select exact Go1.27.3 archive bytes, pin the complete26-file MyGo patch manifest and every member, use offline vendor/local caches and explicit six product roots, and check a fresh host plus all four actual GUI worker PEs. The old three-engine non-executable fixture checker is not full four-worker package validation.
>
> Native headless/method/helper tests, a complete PE/manifest package check, and conditional real-worker tests each establish only their documented scopes. Physical Windows GUI/clipboard/IME, current-candidate hidden host/session/WM_CLOSE/owned Job, four actual worker requests/failure cleanup, fresh-profile restoration/second launch, and resource/response workloads remain separately required. No physical-UI automation workaround or Tester/hidden-start substitute is proposed.
>
> The historical1,918 Tauri/React tests describe the legacy frontend. They are not native test counts, do not exercise the current MyGo UI, and cannot close native acceptance. Legacy workflow source stays disabled and unchanged.

## Proposed documentation targets after root review

| Existing document | Narrow proposed change |
|---|---|
| [Top-level README:3–19](<F:/AIproject/LevNote/README.md#L3-L19>) | Keep native-line and accepted .29 identification; add a concise current-autonomous-development section with separate limited-completion/pending-final gates. Retain legacy sections explicitly as legacy. |
| [Native README](<F:/AIproject/LevNote/native/README.md>) | Add current development commands/boundaries, not a claim of a replacement accepted executable. Preserve historical release descriptions and declared format limitations. |
| [Native validation record](<F:/AIproject/LevNote/native/VALIDATION.md>) | Add a new dated section only after actual root gates; bind each command/source/package/runtime/raw/exit/skip and cleanup result. Do not overwrite earlier .29/closed failures. |
| [Native coordination instructions](<F:/AIproject/LevNote/native/AGENTS.md>) and [autonomous plan](<F:/AIproject/LevNote/native/handoffs/autonomous-native-delivery-plan-20261112.md>) | Root updates fresh ownership/results; newest local execution-plane amendment remains authoritative over historical box-only paragraphs. |

## Current local command text

Future docs should recommend only these actual product scopes, matching the [current local gate](<F:/AIproject/LevNote/.localtools/gate.sh>) rather than its historical “eleven roots” comment:

```text
bash .localtools/gate.sh test ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/... -json -count=1
bash .localtools/gate.sh vet
bash .localtools/gate.sh build
```

No repository-wide `./...` that includes handoff snapshots. The local gate's `fmt` branch actually invokes vet; it must not be documented as a successful formatter. Formatting and semantic gates have separate recorded outcomes.

Current packaging's [Go wrapper](<F:/AIproject/LevNote/native/scripts/go.ps1>) selects `native/tools/go/bin/go.exe`, while ordinary local development selects `.localtools/go1.27.3`. The CI draft resolves this mismatch only in a **new disposable checkout** by verified official archive extraction into the fixed packaging path, never by installing system Go, replacing current local tools, or weakening existing wrappers.

## Evidence boundaries and known environment failure

- [Task15 current handoff:1–21](<F:/AIproject/LevNote/native/handoffs/autonomous-windows-save-conflict-correction-20261113.md#L1-L21>) reports limited Document67+46 and App379+391 affected PASS with28 existing App opt-in skips, and distinguishes unresolved format/root-final status at its own snapshot. This Task20 owner read the report, not all raw receipts; root owns the reported independent check and later changes.
- [Independent Spec findings:7–9](<F:/AIproject/LevNote/native/handoffs/autonomous-final-spec-review-20261113.md#L7-L9>) are the specific T05 basis; [Gate B/C:37–38](<F:/AIproject/LevNote/native/handoffs/autonomous-final-spec-review-20261113.md#L37-L38>) keeps actual platform and complete delivery work separate.
- Root's current message reports Task16 official PS7 parser/C# compile PASS with unchanged helper pins, **no native call**. That is Task16 preparation validation, not a parse/run result for these new Task20 helpers.
- Root reports the [existing build-guard](<F:/AIproject/LevNote/native/scripts/check-build-guard.ps1#L18-L20>) fresh `All` call failed at the isolated tools Junction with Access denied, wrapperexit1, zero stdout and266-byte stderr; no guard case ran. This is FAIL/unverified environment evidence, not a native product failure or PASS. Root preserves the raw namespace; no copy/alternate-shell/ACL bypass is proposed or performed here.

- Final root message reports the current local whole-product Go test/vet/build gates succeeded on source freeze02 (1211files, inventory SHA256 `874bc1fd23a3d02e432c9c6305842ee540afdca1cdad23692905da80d01b2bdf`). Shipping input1239files SHA256 `405d71a737a83f9563fb0e992ab4b9e78b24df37e8e129e30108ac9f55e5128a`. These are **root-reported**, not independently read raw gates or a runnable Task20 CI contract. No counts/skips/timings are invented here.
- The same message reports the current fresh package call failed in the existing script at line24 `New-Item` with Access denied before output directory creation or any GUI build. Exit1, stdout0B, stderr385B; raw SHA256 `39df0306422b71fc88a02baae14574149d58a0a09c539b76a866697016ad0cb1`; both intended target directories confirmed NOT_FOUND. Root retained [package failure evidence](<F:/AIproject/LevNote/.localtools/package-root-20261114-01>). **No package, PE, real worker or host** was created/run. No output change/precreate/copy/alternate-shell workaround was performed or proposed. CI preparation remains independent of the host permission failure.

The draft CI keeps that existing guard case intact for a normal authorized Windows runner. Activating the draft is still contingent on reviewed helpers, provisioning/runner policy, actual successful fresh gates and an explicit root decision; ordinary production correction continues independently of the blocked formatting/Junction lane.
