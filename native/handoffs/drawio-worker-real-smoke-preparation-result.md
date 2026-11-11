# Fourth worker real managed smoke: preparation result

Status: test fixture prepared and frozen. Not compiled, executed or launched.
Production source and the accepted `.18` package are unchanged.

Active task:
[`drawio-worker-real-smoke-preparation-task.md`](drawio-worker-real-smoke-preparation-task.md).

## Test fixture

Added one Windows-only opt-in test:

- `native/core/engineruntime/drawio_managed_bundle_test.go`
- 21,108 bytes, SHA256
  `D3420A97DF382C546EA73D014AD000DFDB2AB887152BC6CE768AA3E9AC374B48`

`TestDrawioManagedRealBundleSmoke` skips before filesystem or process work unless
its exact enable variable is `1`. The future run must use `-test.run` for this
test only.

The fixture:

- Requires the test executable to run from a complete evidence-owned bundle COPY,
  outside the immutable candidate release.
- Creates a fresh run-owned config root and writes a valid legacy schema-1 state
  containing explicit bundled selections for the old three engines.
- Confirms the fourth engine is bundled and the legacy state file remains
  byte-identical while `NewPackageManager` loads it.
- Imports the chosen draw.io package from the explicit candidate release, checks
  that this does not activate it, and verifies the old three selections remain
  unchanged and the fourth remains bundled.
- Calls `Activate` explicitly with the import receipt digest, verifies the exact
  installed selection, and verifies the old three selections again.
- Makes exactly one `NewManaged` call and exactly one `Runner.Export` call.
- Requests page index `1` with exact page ID `second`.
- Compares output to the frozen literal SVG oracle copied from the accepted
  worker second-page test. It never invokes a format renderer to derive expected
  output.
- Asserts the exact two-warning order and metadata: `UNSUPPORTED_ELEMENT` for
  selected-page cells `0` then `1`, each spanning the whole source.
- Confirms the original XML bytes and hash are unchanged.
- Confirms `Result.Engine` has the installed source, candidate version and
  executable SHA256, and the exact receipt package digest.
- Re-reads the candidate executable after the run to prove its hash did not
  change.

The accepted second-page source is below 1 KiB. This is a functional managed
worker smoke only; it makes no huge-input, UI-export, resource, memory or CPU
claim.

## Process evidence

The test installs only observation callbacks through the existing private
`testHooks`. It does not override the executable, arguments, stdio, request,
response or lifecycle.

The callbacks and test record:

- Admission, verification, Start and Wait observation counts.
- The exact exported page index and ID.
- Start and Wait timestamps plus process PID.
- A duplicated exact owned process handle from `afterStart`.
- `QueryFullProcessImageNameW` result and the expected installed executable path.
- `GetProcessTimes` creation FILETIME and UTC value.
- Process exit code and handle close result.
- Explicit `Runner.Close` result and any idempotent cleanup fallback.
- Export, source, payload and warning hashes/byte counts.

The JSON record uses explicit observation booleans plus separate error strings,
so a missing observation is distinguishable from an observed failure. Failures
are accumulated before the test returns. The JSON evidence file is created with
exclusive-create semantics so cleanup or a repeated run cannot erase or overwrite
prior evidence.

## Environment contract

All paths must be absolute local drive paths:

| Variable | Contract |
| --- | --- |
| `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE` | Exactly `1` enables the test. |
| `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE` | Existing complete candidate release directory. It must not contain the evidence root. |
| `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE` | Absent JSON output path below the fresh evidence root and outside `bundle-copy`. |
| `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG` | Absent config root below the fresh evidence root, separate from `bundle-copy`; the test creates it. |

The test executable directory must be a descendant of the evidence root and is
the package-manager host directory. The candidate release, evidence bundle COPY,
config root and evidence JSON are separate trees. The old
`LEVNOTE_MANAGED_REAL_SMOKE` smoke variables are not used by this fixture.

## Deferred .19 execution command

Not executed in this preparation stage. After `.19` acceptance and review of
this fixture, use a fresh evidence root and copy the complete candidate to its
`bundle-copy` before building the test executable there:

```powershell
$candidate = '<ABS_CANDIDATE_RELEASE>'
$evidenceRoot = '<ABS_FRESH_RUN_EVIDENCE_ROOT>'
$bundleCopy = Join-Path $evidenceRoot 'bundle-copy'
$testExe = Join-Path $bundleCopy 'engineruntime.test.exe'
$configRoot = Join-Path $evidenceRoot 'config'
$evidenceJSON = Join-Path $evidenceRoot 'drawio-managed-real-smoke.json'

New-Item -ItemType Directory -Path $bundleCopy | Out-Null
Get-ChildItem -LiteralPath $candidate -Force |
    Copy-Item -Destination $bundleCopy -Recurse -Force

& .\native\scripts\go.ps1 -GoArguments @(
    'test','-c','-o',$testExe,'./core/engineruntime'
)
```

The later run command, with the same working directory, is:

```powershell
$env:LEVNOTE_MANAGED_REAL_SMOKE = $null
$env:LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE = '1'
$env:LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE = $candidate
$env:LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE = $evidenceJSON
$env:LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG = $configRoot

& $testExe -test.run '^TestDrawioManagedRealBundleSmoke$' -test.v
```

## Evidence naming

The future run should preserve this structure:

```text
<fresh-run-evidence-root>/
  bundle-copy/                     complete immutable candidate copy plus test exe
  config/                          test-created run-owned legacy state/store
  test-build.log                   wrapper `test -c` output and exit code
  test-run.log                     exact test executable output and exit code
  drawio-managed-real-smoke.json   exclusive smoke record
```

The candidate directory remains the import source and is not the host directory.
The future runner must preserve raw build/run output and every failure attempt.

## Frozen and preserved inputs

The new fixture is the only product-tree addition:

```text
D3420A97DF382C546EA73D014AD000DFDB2AB887152BC6CE768AA3E9AC374B48  21108  native/core/engineruntime/drawio_managed_bundle_test.go
```

The existing smoke and accepted correction inputs remain unchanged:

```text
47E0162DF75F73E2146469AABF24BF5C3F976E99A18BFA55CB8C7A913FCA0A90  native/core/engineruntime/managed_bundle_test.go
AF39CBEF0CAE6DDDBBF8AD1DB1BDDBD944810097D77BCFC69AFC61A15AF3603C  native/handoffs/drawio-worker-stage1-warning-correction-result.md
1B8EA397373D1CD3A2CC6F09D4FA6B9EA27E81B6F2FC0241722E7D7D4B85BE20  native/cmd/levnote-drawio-svg/main.go
06007F93A9CCF105A16943B0F780322D5103266D84808145F2BF28FDE03E88E5  native/cmd/levnote-drawio-svg/main_test.go
```

No build, test, benchmark, launch, UI action, resource measurement, packaging
change or `.18` modification was performed.

The test fixture is frozen for root review. Actual `.19` execution remains
deferred until root activates the future command.
