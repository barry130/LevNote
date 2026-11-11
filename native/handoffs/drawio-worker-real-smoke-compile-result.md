# Fourth worker real managed smoke: disabled compile result

Status: compile-complete and frozen for root review. The opt-in smoke remained
disabled. No real worker or host was launched; no package manager, managed
runner, or smoke filesystem fixture was exercised.

Active task:
[`drawio-worker-real-smoke-compile-task.md`](drawio-worker-real-smoke-compile-task.md).

## Scope and final source

The only product-tree edit was `gofmt` on the new Windows test fixture:

```text
074C4894D1D05F0FEBE7B49F61BF958980BDDF6DFEF9151179A18C05A2FF4548  21150  native/core/engineruntime/drawio_managed_bundle_test.go
```

The prepared fixture hash was
`D3420A97DF382C546EA73D014AD000DFDB2AB887152BC6CE768AA3E9AC374B48`.
No compile failure occurred, so no source repair was made after formatting.

The frozen preparation result was preserved unchanged:

```text
E7898D43E28630F3B596AE0BEA6F6E631BD9E754D79F4BDED6886371282054D4  7241  native/handoffs/drawio-worker-real-smoke-preparation-result.md
```

## Commands and environment

All Go work used the required wrapper from the repository root.

Format:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('fmt','./core/engineruntime/drawio_managed_bundle_test.go')
```

Result: exit `0`, stdout named the formatted file, stderr was empty.

Disabled compile/test:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','./core/engineruntime','-run','^TestDrawioManagedRealBundleSmoke$','-count=1','-v')
```

These variables were absent before the command and were removed before invoking
the test:

```text
LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE
LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE
LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE
LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG
```

Raw result: exit `0`, empty stderr, exactly one `SKIP`, and a passing package
result:

```text
=== RUN   TestDrawioManagedRealBundleSmoke
    drawio_managed_bundle_test.go:122: real managed draw.io smoke requires explicit LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE=1
--- SKIP: TestDrawioManagedRealBundleSmoke (0.00s)
PASS
ok  	levnote.local/native/core/engineruntime	0.735s
```

PowerShell represented the absent process variables as present empty strings in
the command's post-test `finally` snapshot. That raw evidence is preserved in
`environment-restored-after-test.txt`. A separate PowerShell process then
observed all four variables absent in
`environment-after-command-process.txt`; the variables were process-scoped and
were never persisted outside the command.

The skip guard is the first operation in the test. It precedes creation of the
JSON record, candidate/evidence/config path validation, filesystem access, the
package manager, `NewManaged`, `Runner.Export`, and every process observation.
Thus this run proves the new test compiles and preserves default-disabled
behavior; it does not exercise the real worker.

## Evidence

Fresh, non-overwritten evidence is under:

`native/handoffs/drawio-worker-real-smoke-compile-evidence/`

It contains exact command text, environment snapshots and handling note,
separate stdout/stderr, exit codes, and a SHA256 inventory:

```text
C41E781EB340692734D6AE020C89D700487701F649B643CB7899B4455CD93C2B  1310  evidence-manifest.sha256
37DCC06ADD3B7882D0FA4E775BB093E404C515FB151DF00E37C66E190143BFB4   276  test-stdout.log
E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855     0  test-stderr.log
13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354     3  test-exit-code.txt
```

No enabled smoke, worker/host launch, smoke package-manager invocation,
packaging, app edit, full runtime suite, benchmark, resource measurement, or
unrelated vet ran.

## Deferred execution timing

The actual enabled smoke belongs during `.19` candidate delivery, after accepted
stage2/portable source and root review of this fixture, and before root binary
acceptance. The older preparation result's phrase “After `.19` acceptance” is
imprecise; it remains frozen as historical text and is not rewritten here.

The final fixture and this result are frozen pending root review. `.18` and all
prior stage1/correction evidence are unchanged.
