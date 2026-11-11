param([Parameter(Mandatory)][string]$EvidenceRoot)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$wrapperRoot = Split-Path -Parent $PSScriptRoot
$runScript = Join-Path $wrapperRoot 'run-matrix.ps1'
$summaryScript = Join-Path $wrapperRoot 'summarize-matrix.ps1'
if (Test-Path -LiteralPath $EvidenceRoot) { throw 'Fixture evidence must be fresh.' }
New-Item -ItemType Directory -Path $EvidenceRoot | Out-Null
[IO.File]::WriteAllText((Join-Path $EvidenceRoot 'FIXTURE-ONLY.txt'), 'Synthetic wrapper verification. Start-Process is mocked. No tester, worker, host, Go test, build, or preflight execution.')
$checks = [Collections.Generic.List[object]]::new()
function Check([string]$Name, [bool]$Pass) {
    $checks.Add([ordered]@{ Name = $Name; Pass = $Pass })
    if (-not $Pass) { throw "Fixture check failed: $Name" }
}
function Load-ActualFunctions([string]$Path) {
    $tokens = $null; $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$parseErrors)
    if ($parseErrors.Count -ne 0) { throw ($parseErrors | Out-String) }
    foreach ($definition in $ast.EndBlock.Statements) {
        if ($definition -is [Management.Automation.Language.FunctionDefinitionAst]) { $definition.Extent.Text }
    }
}
foreach ($definition in (Load-ActualFunctions $runScript)) { . ([scriptblock]::Create($definition)) }
foreach ($item in @(@{ Name = 'null'; Value = $null; Count = 0 }, @{ Name = 'empty'; Value = @(); Count = 0 },
    @{ Name = 'singleton'; Value = @('one'); Count = 1 }, @{ Name = 'multiple'; Value = @('one', 'two'); Count = 2 })) {
    $actual = ConvertTo-Array $item.Value
    Check "actual array helper: $($item.Name)" ($actual -is [array] -and $actual.Count -eq $item.Count)
}
$emptyWarningFixture = [pscustomobject]@{ expectedWarnings = [pscustomobject]@{ version = $null } }
Check 'actual warning helper admits null arrays' (@(Get-WarningViolations ([pscustomobject]@{ actualWarnings = $null }) $emptyWarningFixture 'version').Count -eq 0)
$warning = [pscustomobject]@{ code = 'fixture'; message = 'fixture'; start = 1; end = 2 }
Check 'actual warning helper compares singleton' (@(Get-WarningViolations ([pscustomobject]@{ actualWarnings = $warning }) ([pscustomobject]@{ expectedWarnings = [pscustomobject]@{ version = @($warning) } }) 'version').Count -eq 0)
$truncated = Get-MeasurementSummary ([pscustomobject]@{ probe = [pscustomobject]@{ finalTimes = $null } })
Check 'absent summary and null finalTimes preserve null counters' ($null -eq $truncated.CPUKernel100ns -and $null -eq $truncated.ExportElapsedNS -and $null -eq $truncated.FinalCPUReadOK -and $truncated.Failures.Count -eq 0)
$terminal = Initialize-RunRecord 'fixture' @{} @{} 'fixture' @(@{})
for ($i = 0; $i -lt 12; $i++) { $terminal.Rows.Add([ordered]@{ Status = 'PASS' }) }
Complete-RunRecord $terminal $false ''
Check 'twelve rows cannot pass before package protection' ($terminal.Status -ceq 'FAIL')
$terminal.PackageProtectionEqual = $true
Complete-RunRecord $terminal $false ''
Check 'twelve rows plus package protection passes' ($terminal.Status -ceq 'PASS')
$terminal.Rows.RemoveAt(11)
Complete-RunRecord $terminal $false ''
Check 'partial terminal run cannot pass' ($terminal.Status -ceq 'FAIL')

$preflightPath = Join-Path $wrapperRoot '..\drawio-worker-resources-harness-final-validation-20261110\preflight-output\preflight.json'
$fixturePreflight = Get-Content -LiteralPath $preflightPath -Raw | ConvertFrom-Json
$global:MatrixFixtureState = [ordered]@{ Mode = ''; Calls = 0; Processes = [Collections.Generic.List[object]]::new(); Preflight = $fixturePreflight; RunningSnapshots = [Collections.Generic.List[object]]::new() }
function Start-Process {
    param($FilePath, $ArgumentList, $WorkingDirectory, $WindowStyle, $RedirectStandardOutput, $RedirectStandardError, [switch]$PassThru)
    $state = $global:MatrixFixtureState
    $state.Calls++
    if ($WindowStyle -cne 'Hidden' -or $ArgumentList[1] -cne '^TestDrawioWorkerResourceMeasurement$') { throw 'Unexpected launch request in fixture.' }
    $output = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS_OUTPUT', 'Process')
    if (Test-Path -LiteralPath $output) { throw 'Measurement child must be absent at mocked start.' }
    $sampleRoot = Split-Path -Parent $output
    $runPath = Join-Path (Split-Path -Parent $sampleRoot) 'matrix-run.json'
    $running = Get-Content -LiteralPath $runPath -Raw | ConvertFrom-Json
    $state.RunningSnapshots.Add([ordered]@{ Status = $running.Status; Rows = $running.Rows.Count })
    [IO.File]::WriteAllText($RedirectStandardOutput, 'FIXTURE ONLY: fake process stdout')
    [IO.File]::WriteAllText($RedirectStandardError, 'FIXTURE ONLY: fake process stderr')
    if ($state.Mode -ceq 'start-error') { throw 'fixture Start-Process failure' }
    New-Item -ItemType Directory -Path (Join-Path $output 'config') | Out-Null
    [IO.File]::WriteAllText((Join-Path $output 'config\fixture.txt'), 'synthetic config')
    $fixtureName = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_FIXTURE', 'Process')
    $fixture = @($state.Preflight.fixtures | Where-Object { $_.name -ceq $fixtureName })[0]
    $version = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_VERSION', 'Process')
    $ordinal = [int][Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_ORDINAL', 'Process')
    $measurement = [ordered]@{
        FixtureOnly = $true; schema = 1; status = 'PASS'; matrixPosition = $state.Calls
        fixture = $fixtureName; version = $version; ordinal = $ordinal
        sourceSHA256 = $fixture.sourceSha256; expectedSourceSHA256 = $fixture.sourceSha256; sourceBytes = $fixture.sourceBytes
        tester = $state.Preflight.tester; actualSVGBytes = $fixture.svgBytes; actualSVGSHA256 = $fixture.svgSha256
        actualWarnings = $fixture.expectedWarnings.$version
        admissionCalls = 1; verifyCalls = 1; startCalls = 1; waitCalls = 1; failures = $null
        limitations = @('FIXTURE ONLY: incomplete lifetime peak coverage')
        exportElapsedNs = 100 + $state.Calls; sampleIntervalNs = 2000000; maxAttempts = 20000
        probe = [ordered]@{
            Summary = [ordered]@{ Attempts = 3; FailedReads = 0; SampledMaxWorkingSetBytes = 1024; SampledMaxPrivateUsageBytes = 2048
                ObservedOSPeakWorkingSetBytes = 4096; ObservedOSPeakPagefileBytes = 8192
                FinalMemoryReadOK = $true; FinalCPUReadOK = $true; LifetimePeakCountersComplete = $false; PeakCoverage = 'fixture only' }
            FinalTimes = [ordered]@{ Times = [ordered]@{ Kernel100ns = 0; User100ns = 20; LifetimeNS = 300 } }
        }
    }
    if ($state.Mode -ceq 'null-final') { $measurement.probe.FinalTimes = $null; $measurement.probe.Summary.FinalCPUReadOK = $false }
    $json = if ($state.Mode -ceq 'malformed') { '{fixture-only malformed json' } else { $measurement | ConvertTo-Json -Depth 12 }
    [IO.File]::WriteAllText((Join-Path $output 'measurement.json'), $json)
    $fake = [pscustomobject]@{ Id = 987654; HasExited = $false; ExitCode = 0; Mode = $state.Mode; Killed = 0; Disposed = 0; Trace = [Collections.Generic.List[string]]::new() }
    $fake | Add-Member ScriptProperty SafeHandle { $this.Trace.Add('handle'); [pscustomobject]@{ IsInvalid = $false } }
    $fake | Add-Member ScriptProperty StartTime { $this.Trace.Add('start-time'); if ($this.Mode -ceq 'identity-error') { throw 'fixture optional identity unavailable' }; [DateTime]::UtcNow }
    $fake | Add-Member ScriptProperty Path { $this.Trace.Add('path'); 'FIXTURE-ONLY-NO-EXECUTABLE' }
    $fake | Add-Member ScriptMethod WaitForExit {
        param([int]$Milliseconds)
        $this.Trace.Add("wait:$Milliseconds")
        if ($Milliseconds -eq 120000 -and $this.Mode -ceq 'wait-error') { throw 'fixture wait exception' }
        if ($Milliseconds -eq 120000 -and $this.Mode -in @('timeout', 'cleanup-timeout')) { return $false }
        if ($Milliseconds -eq 10000 -and $this.Mode -ceq 'cleanup-timeout') { return $false }
        $this.HasExited = $true
        return $true
    }
    $fake | Add-Member ScriptMethod Kill { $this.Trace.Add('kill'); $this.Killed++ }
    $fake | Add-Member ScriptMethod Dispose { $this.Trace.Add('dispose'); $this.Disposed++ }
    $state.Processes.Add($fake)
    return $fake
}

try {
    foreach ($mode in @('pass', 'malformed', 'start-error', 'wait-error', 'timeout', 'cleanup-timeout', 'null-final', 'identity-error')) {
        $global:MatrixFixtureState.Mode = $mode
        $global:MatrixFixtureState.Calls = 0
        $global:MatrixFixtureState.Processes.Clear()
        $global:MatrixFixtureState.RunningSnapshots.Clear()
        $caseRoot = Join-Path $EvidenceRoot $mode
        & $runScript -OutputRoot $caseRoot -Activation 'RUN_FIXED_12_DRAWIO_WORKER_MATRIX'
        $wrapperExit = $LASTEXITCODE
        [IO.File]::WriteAllText((Join-Path $caseRoot 'fixture-wrapper-exit.txt'), [string]$wrapperExit)
        $recordPath = Join-Path $caseRoot 'matrix-run.json'
        $record = Get-Content -LiteralPath $recordPath -Raw | ConvertFrom-Json
        $shouldPass = $mode -in @('pass', 'identity-error')
        Check "$mode actual wrapper exit/state" (($shouldPass -and $wrapperExit -eq 0 -and $record.Status -ceq 'PASS' -and $record.Rows.Count -eq 12) -or
            (-not $shouldPass -and $wrapperExit -ne 0 -and $record.Status -ceq 'FAIL' -and $record.Rows.Count -eq 1 -and $record.Rows[0].Status -ceq 'FAIL'))
        Check "$mode package inventory runs in finally" ($record.PackageProtectionEqual -eq $true -and $record.PackageAfter.Count -eq 2)
        Check "$mode per-row record stays RUNNING" (@($global:MatrixFixtureState.RunningSnapshots | Where-Object { $_.Status -cne 'RUNNING' }).Count -eq 0)
        Check "$mode available streams retained" ($null -ne $record.Rows[0].Stdout -and $null -ne $record.Rows[0].Stderr)
        Check "$mode no retry" ($global:MatrixFixtureState.Calls -eq $(if ($shouldPass) { 12 } else { 1 }))
        foreach ($process in $global:MatrixFixtureState.Processes) {
            Check "$mode handle precedes optional identity" ($process.Trace[0] -ceq 'handle' -and $process.Trace[1] -ceq 'start-time')
            Check "$mode exact fake object disposed once" ($process.Disposed -eq 1)
            if ($mode -in @('wait-error', 'timeout', 'cleanup-timeout')) {
                Check "$mode exact cleanup bounded" ($process.Killed -eq 1 -and $process.Trace.Contains('wait:10000'))
            }
        }
        if ($mode -ceq 'cleanup-timeout') { Check 'cleanup timeout recorded' ($null -ne $record.Rows[0].CleanupError) }
        if ($mode -ceq 'null-final') { Check 'null CPU remains unknown in actual wrapper' ($null -eq $record.Rows[0].MeasurementSummary.CPUKernel100ns) }
        Copy-Item -LiteralPath $recordPath -Destination (Join-Path $caseRoot 'matrix-run.before-fixture-label.json')
        $record | Add-Member NoteProperty FixtureOnly $true
        [IO.File]::WriteAllText($recordPath, ($record | ConvertTo-Json -Depth 24))
        & $summaryScript -MatrixOutputRoot $caseRoot | Out-Null
        $summary = Get-Content -LiteralPath (Join-Path $caseRoot 'matrix-summary.json') -Raw | ConvertFrom-Json
        Check "$mode actual summarizer keeps raw rows" ($summary.TotalRows -eq $record.Rows.Count -and $summary.FixtureOnly)
        if ($shouldPass) {
            $stats = $summary.DescriptiveGroups[0].Stats
            Check "$mode elapsed and CPU statistics have values and units" ($stats.ExportElapsedNS.Count -eq 3 -and $stats.ExportElapsedNS.Median -gt 0 -and $stats.ExportElapsedNS.Unit -ceq 'ns' -and
                $stats.CPUKernel100ns.Count -eq 3 -and $stats.CPUKernel100ns.Median -eq 0 -and $stats.CPUKernel100ns.Unit -ceq '100ns' -and $stats.CPUUser100ns.Median -eq 20 -and $stats.ProcessLifetimeNS.Unit -ceq 'ns')
        }
    }
    $truncatedRoot = Join-Path $EvidenceRoot 'truncated-summary'
    New-Item -ItemType Directory -Path $truncatedRoot | Out-Null
    # Deliberately omit fields; null placeholders would hide StrictMode defects.
    $truncatedRun = [ordered]@{ Schema = 1; Status = 'FAIL'; FixtureOnly = $true; Rows = @(
        [ordered]@{ Index = 1; Status = 'PASS'; Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.20' },
        [ordered]@{ Index = 2; Status = 'FAIL' }
    ) }
    [IO.File]::WriteAllText((Join-Path $truncatedRoot 'matrix-run.json'), ($truncatedRun | ConvertTo-Json -Depth 8))
    & $summaryScript -MatrixOutputRoot $truncatedRoot | Out-Null
    $summary = Get-Content -LiteralPath (Join-Path $truncatedRoot 'matrix-summary.json') -Raw | ConvertFrom-Json
    Check 'actual summary handles omitted measurementSummary/exitCode/timedOut/violations' ($summary.FailureDiagnostics.Count -eq 2 -and $null -eq $summary.FailureDiagnostics[1].ExitCode -and $null -eq $summary.FailureDiagnostics[1].TimedOut -and $summary.FailureDiagnostics[1].Violations.Count -eq 0 -and $summary.DescriptiveGroups.Count -eq 0)

    $callsBefore = $global:MatrixFixtureState.Calls
    & $runScript -OutputRoot $EvidenceRoot -Activation 'RUN_FIXED_12_DRAWIO_WORKER_MATRIX'
    Check 'existing output precondition exits nonzero without launch' ($LASTEXITCODE -ne 0 -and $global:MatrixFixtureState.Calls -eq $callsBefore)
    Push-Location $EvidenceRoot
    try {
        & $runScript -OutputRoot (Join-Path $EvidenceRoot 'wrong-cwd') -Activation 'RUN_FIXED_12_DRAWIO_WORKER_MATRIX'
        Check 'wrong cwd precondition exits nonzero without launch' ($LASTEXITCODE -ne 0 -and $global:MatrixFixtureState.Calls -eq $callsBefore)
    } finally { Pop-Location }
    $protectedOutput = 'F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.21\FIXTURE-ONLY-MUST-NOT-EXIST'
    & $runScript -OutputRoot $protectedOutput -Activation 'RUN_FIXED_12_DRAWIO_WORKER_MATRIX'
    Check 'protected package output rejected before write or launch' ($LASTEXITCODE -ne 0 -and -not (Test-Path -LiteralPath $protectedOutput) -and $global:MatrixFixtureState.Calls -eq $callsBefore)

    foreach ($definition in (Load-ActualFunctions $summaryScript)) { . ([scriptblock]::Create($definition)) }
    $stats = Get-Stats @{ Unit = 'ns'; Quantization = 'fixture' } @(10, 11)
    Check 'actual stats preserve fractional even median' ($stats.Median -eq 10.5)
    $result = [ordered]@{ FixtureOnly = $true; PowerShell = $PSVersionTable.PSVersion.ToString(); Pass = $true; CheckCount = $checks.Count; Checks = $checks; Scope = 'Actual scripts/functions with mocked Start-Process. No real tester/worker/host/test/build/preflight.' }
    [IO.File]::WriteAllText((Join-Path $EvidenceRoot 'fixture-results.json'), ($result | ConvertTo-Json -Depth 10))
    ([pscustomobject]$result) | Select-Object FixtureOnly, PowerShell, Pass, CheckCount | ConvertTo-Json
}
catch {
    [IO.File]::WriteAllText((Join-Path $EvidenceRoot 'fixture-failure.json'), ([ordered]@{ FixtureOnly = $true; Error = $_.Exception.Message; Stack = $_.ScriptStackTrace; Checks = $checks } | ConvertTo-Json -Depth 10))
    throw
}
finally { Remove-Variable MatrixFixtureState -Scope Global }
