# Fresh local Windows host acceptance preparation

**Prepared only: not parsed, compiled, executed or product-accepted.** Task-16
created only this new directory; no Go/test/vet/build, command, parser/compiler,
launch, Git, MCP/network or delegation. Root owns review and later execution.
Accepted .29 and all closed .30/prepare02/box evidence remain immutable.

## Root preflight and later invocation

Review [runner](<run-host-smoke.ps1>), [library](<delivery-lib.ps1>) and
[suspended launcher](<owned-host.cs>); freeze final helper hashes. Verify native
x64 PowerShell7 exact path/version/hash and .NET readiness. Record parser/C#
compiler outcomes in fresh caller evidence **without product launch** first.
Then freeze current source/package, review real acceptance/activation receipts,
create a new real [input contract](<input-contract.template.json>) following
[INPUTS](<INPUTS.md>) and the [shipping-source shape](<source-inventory.template.json>).
The current product+vendor snapshot is insufficient. Templates contain false
booleans/invalid pins and do not authorize a run. Hash the final reviewed contract.

Later, root alone invokes the exact reviewed runtime once and preserves the
command, raw stdout/stderr/exit and failures in separate fresh caller evidence:

```powershell
& $ReviewedPwshExe -NoLogo -NoProfile -NonInteractive -File `
  'F:/AIproject/LevNote/native/handoffs/autonomous-local-windows-delivery-20261113/run-host-smoke.ps1' `
  -InputContractPath $FinalReviewedContract -InputContractSHA256 $RealContractSHA256 `
  -ExecuteReviewed
```

All pins are checked before launch. Evidence must be an absent ordinary direct
`native/artifacts/acceptance/autonomous-local-windows-<new-id>` or
`native/handoffs/autonomous-local-windows-<new-id>`; candidate a fresh ordinary
`native/bin/<new-id>`. Never use this existing helper directory as evidence.

## Limited acceptance and cleanup

Exact manifest/five amd64 GUI PEs/complete source → exact copied bundle → fresh
ordinary default C:TEMP GUID config and pinned plugin → ordered BOM/CRLF UTF-8
text then PNG → CreateProcessW suspended, restricted stdio inheritance,
kill-on-close Job assignment/held identity **before resume** → native-title
hidden HWND≤20s, alive settling≥5000ms → strict fresh clean two-tab session →
**one** owned WM_CLOSE≤15s exit0/Job total1 active0 → newer close publication →
unchanged protected inputs. No fallback close, name-based kill or input/repaint.

Failures stay FAIL; preserve errors/streams/session/config, terminate only owned
Job/held suspended process, join host≤10s, inspect Job active0 and dispose handles.
Host join/Job close does **not** prove empty Job if accounting still has active
processes: asynchronous cleanup remains unverified/FAIL. Nothing is deleted;
failed namespaces cannot be reused. Pre-evidence/controller interruption requires
a caller receipt; absent result is never PASS. Future cleanup needs exact-path review.

## Nonclaims and pending risks

Not physical UI/IME, visual PNG, dirty save/history, second-launch restore, worker
operation, resources/security/installer or whole-product delivery. PNG gate is
signature/IHDR only. P/Invoke layout/inheritance, hidden-window behavior, PowerShell
bindings, timestamps and actual compilation/runtime remain unvalidated. Job is
lifecycle ownership, not a network/filesystem sandbox; externally killed controller
before Job assignment needs supervision. Hashes bind bytes, not truthful acceptance.
Root independently reviews actual receipts; this preparation activates no runnable.
Design sources/final pins: [PROVENANCE](<PROVENANCE.md>).
