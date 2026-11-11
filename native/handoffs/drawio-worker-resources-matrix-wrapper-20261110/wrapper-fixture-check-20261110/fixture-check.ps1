$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$wrapperRoot = 'F:\AIproject\LevNote\native\handoffs\drawio-worker-resources-matrix-wrapper-20261110'
$fixtureRoot = 'F:\AIproject\LevNote\native\handoffs\drawio-worker-resources-matrix-wrapper-20261110\wrapper-fixture-check-20261110'
$checks = [Collections.Generic.List[object]]::new()
function Check([string]$Name, [bool]$Pass, [string]$Detail = '') {
    $checks.Add([ordered]@{ Name = $Name; Pass = $Pass; Detail = $Detail })
}

$repositoryRoot = [IO.Path]::GetFullPath((Join-Path $wrapperRoot '..\..\..'))
$expectedRepositoryRoot = 'F:\AIproject\LevNote'
$marker = Join-Path $repositoryRoot 'AGENTS.md'
$goMarker = Join-Path $repositoryRoot 'native\scripts\go.ps1'
Check 'repository root arithmetic and markers' (
    $repositoryRoot -ceq $expectedRepositoryRoot -and
    (Test-Path -LiteralPath $marker -PathType Leaf) -and
    (Test-Path -LiteralPath $goMarker -PathType Leaf)
) "repositoryRoot=$repositoryRoot"

$preflightPath = Join-Path $wrapperRoot '..\drawio-worker-resources-harness-final-validation-20261110\preflight-output\preflight.json'
$preflight = Get-Content -LiteralPath $preflightPath -Raw | ConvertFrom-Json
$preflightFailures = @($preflight.failures)
if ($preflightFailures.Count -eq 1 -and $null -eq $preflightFailures[0]) { $preflightFailures = @() }
$normalized = @()
if ($null -ne $preflight.failures) { $normalized = @($preflight.failures) }
Check 'successful null-array preflight admission' (
    $preflight.status -ceq 'PASS' -and $normalized.Count -eq 0
) "jsonNull=$($null -ne $preflight.failures), normalized=$($normalized.Count), nullWrapped=$($preflightFailures.Count)"

$layoutRun = [int]([Environment]::GetEnvironmentVariable('LEVNOTE_MATRIX_FIXTURE_CHECK_RUN', 'Process') -as [int])
$layoutRoot = Join-Path $fixtureRoot ("layout-{0:D2}" -f $layoutRun)
$sampleRoot = Join-Path $layoutRoot 'sample-01'
$measurementRoot = Join-Path $sampleRoot 'measurement'
New-Item -ItemType Directory -Path $sampleRoot | Out-Null
$expectedBefore = $true
$expectedAfter = $true
$beforeAbsent = -not (Test-Path -LiteralPath $measurementRoot)
New-Item -ItemType Directory -Path $measurementRoot | Out-Null
$afterPresent = Test-Path -LiteralPath $measurementRoot -PathType Container
$measurementPath = Join-Path $measurementRoot 'measurement.json'
$configPath = Join-Path $measurementRoot 'config'
$expectedBefore = $beforeAbsent -and -not (Test-Path -LiteralPath $measurementPath)
$expectedAfter = $afterPresent -and -not (Test-Path -LiteralPath $measurementPath) -and -not (Test-Path -LiteralPath $configPath)
Check 'absent measurement child output layout' ($expectedBefore -and $expectedAfter) "sampleRoot=$sampleRoot, measurementRoot=$measurementRoot"

$summaryCaseRoot = Join-Path $fixtureRoot ("summary-{0:D2}" -f $layoutRun)
New-Item -ItemType Directory -Path $summaryCaseRoot | Out-Null
$passRoot = Join-Path $summaryCaseRoot 'pass'
New-Item -ItemType Directory -Path $passRoot | Out-Null
$measurementSummary = [ordered]@{
    Failures = @()
    Limitations = @('fixture-only limitation')
    ActualWarnings = @()
    ExportElapsedNS = [Int64]100
    SampleIntervalNS = [Int64]2000000
    MaxAttempts = 20000
    ProbeAttempts = 3
    ProbeFailedReads = 0
    SampledMaxWorkingSetBytes = [UInt64]1024
    SampledMaxPrivateUsageBytes = [UInt64]2048
    ObservedOSPeakWorkingSetBytes = [UInt64]4096
    ObservedOSPeakPagefileBytes = [UInt64]8192
    FinalMemoryReadOK = $true
    FinalCPUReadOK = $true
    LifetimePeakCountersComplete = $false
    PeakCoverage = 'fixture-only incomplete coverage'
    CPUKernel100ns = [UInt64]10
    CPUUser100ns = [UInt64]20
    ProcessLifetimeNS = [Int64]300
}
$passRow = [ordered]@{
    Index = 1
    MatrixPosition = 1
    Fixture = 'NormalMultiPage'
    Version = 'native-worker.20261110.20'
    Ordinal = 1
    Status = 'PASS'
    ExitCode = 0
    TimedOut = $false
    Violations = @()
    MeasurementSummary = $measurementSummary
}
$failRow = [ordered]@{
    Index = 2
    MatrixPosition = 2
    Fixture = 'NormalMultiPage'
    Version = 'native-worker.20261110.21'
    Ordinal = 1
    Status = 'FAIL'
    ExitCode = 2
    TimedOut = $false
    Violations = @('measurement.json missing')
    MeasurementSummary = $null
}
$runRecord = [ordered]@{
    Schema = 1
    Status = 'FAIL'
    StoppedAtFirstFailure = $true
    Rows = @($passRow, $failRow)
}
$runPath = Join-Path $passRoot 'matrix-run.json'
[IO.File]::WriteAllText($runPath, ($runRecord | ConvertTo-Json -Depth 24), [Text.UTF8Encoding]::new($false))
$summarizer = Join-Path $wrapperRoot 'summarize-matrix.ps1'
& $summarizer -MatrixOutputRoot $passRoot | Out-Null
$summaryPath = Join-Path $passRoot 'matrix-summary.json'
$summary = Get-Content -LiteralPath $summaryPath -Raw | ConvertFrom-Json
$group = @($summary.descriptiveGroups | Where-Object { $_.fixture -ceq 'NormalMultiPage' -and $_.version -ceq 'native-worker.20261110.20' })[0]
$failedDiagnostic = @($summary.failureDiagnostics | Where-Object { $_.index -eq 2 })[0]
Check 'partial-record summary preserves pass, failure, limitation, and units' (
    $summary.totalRows -eq 2 -and $summary.passedRows -eq 1 -and $summary.failedRows -eq 1 -and
    $group.rowCount -eq 1 -and
    $group.stats.SampledMaxWorkingSetBytes.unit -ceq 'bytes' -and
    $group.stats.CPUKernel100ns.unit -ceq '100ns' -and
    $group.stats.ProcessLifetimeNS.unit -ceq 'ns' -and
    $group.stats.ProcessLifetimeNS.quantization -like 'Windows process creation-to-exit wall lifetime*' -and
    @($summary.limitations).Count -eq 1 -and
    $failedDiagnostic.measurementSummaryPresent -eq $false -and
    @($failedDiagnostic.violations)[0] -ceq 'measurement.json missing'
) "pass=$($summary.passedRows), fail=$($summary.failedRows), limitationCount=$(@($summary.limitations).Count)"

$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    FixtureOnly = $true
    Scope = 'No-process wrapper fixture checks only: path arithmetic, null-array preflight admission, absent child output layout, and synthetic partial-record summary behavior. No tester, worker, host, build, preflight, or product execution.'
    Pass = @($checks | Where-Object { -not $_.Pass }).Count -eq 0
    Checks = $checks
}
$resultPath = Join-Path $fixtureRoot 'fixture-check.json'
[IO.File]::WriteAllText($resultPath, ($result | ConvertTo-Json -Depth 12), [Text.UTF8Encoding]::new($false))
$result | ConvertTo-Json -Depth 8
if (-not $result.Pass) { exit 1 }
exit 0
