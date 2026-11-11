param([Parameter(Mandatory = $true)][string]$MatrixOutputRoot)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function ConvertTo-Array {
    param($Value)
    if ($null -eq $Value) { return @() }
    return @($Value)
}

function Test-RowProperty {
    param(
        [Parameter(Mandatory = $true)][object]$Row,
        [Parameter(Mandatory = $true)][string]$Name
    )
    @($Row.PSObject.Properties.Name) -contains $Name
}

function Get-Values {
    param(
        [Parameter(Mandatory = $true)][object[]]$Rows,
        [Parameter(Mandatory = $true)][string]$Property
    )
    return @($Rows | Where-Object {
        $null -ne $_.measurementSummary -and
        @($_.measurementSummary.PSObject.Properties.Name) -contains $Property -and
        $null -ne $_.measurementSummary.$Property
    } | ForEach-Object { $_.measurementSummary.$Property })
}

function Get-Stats {
    param($Parameter, $Values)
    $valueList = [Collections.Generic.List[object]]::new()
    foreach ($value in $Values) { $valueList.Add($value) }
    if ($valueList.Count -eq 0) { return $null }
    $sorted = [object[]]($valueList | Sort-Object)
    $middle = [int][Math]::Floor(($sorted.Count - 1) / 2)
    $median = if ($sorted.Count % 2 -eq 1) {
        $sorted[$middle]
    } else {
        if ($sorted[$middle] -is [UInt64] -or $sorted[$middle] -is [Int64]) {
            [UInt64](($sorted[$middle] + $sorted[$middle + 1]) / 2)
        } else {
            $sorted[$middle]
        }
    }
    [ordered]@{
        Unit = $Parameter.Unit
        Count = $sorted.Count
        Minimum = $sorted[0]
        Median = $median
        Maximum = $sorted[-1]
        Quantization = $Parameter.Quantization
    }
}

function Get-GroupStats {
    param([Parameter(Mandatory = $true)][object[]]$Rows)
    $fields = @(
        @{ Name = 'RequestElapsedNS'; Unit = 'ns'; Quantization = 'host stopwatch nanoseconds' },
        @{ Name = 'SampledMaxWorkingSetBytes'; Unit = 'bytes'; Quantization = 'process working-set samples' },
        @{ Name = 'SampledMaxPrivateUsageBytes'; Unit = 'bytes'; Quantization = 'process private commit samples' },
        @{ Name = 'ObservedOSPeakWorkingSetBytes'; Unit = 'bytes'; Quantization = 'OS-observed working-set peak' },
        @{ Name = 'ObservedOSPeakPagefileBytes'; Unit = 'bytes'; Quantization = 'OS-observed pagefile/commit peak' },
        @{ Name = 'CPUKernel100ns'; Unit = '100ns'; Quantization = 'Windows process CPU counter' },
        @{ Name = 'CPUUser100ns'; Unit = '100ns'; Quantization = 'Windows process CPU counter' },
        @{ Name = 'ProcessLifetimeNS'; Unit = 'ns'; Quantization = 'Windows process creation-to-exit wall lifetime; not CPU time' }
    )
    $stats = [ordered]@{}
    foreach ($field in $fields) {
        $values = Get-Values $Rows $field.Name
        $stats[$field.Name] = Get-Stats $field $values
    }
    $stats
}

$wrapperRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$matrixOutputRoot = [IO.Path]::GetFullPath($MatrixOutputRoot)
$runPath = Join-Path $matrixOutputRoot 'matrix-run.json'
$summaryPath = Join-Path $matrixOutputRoot 'matrix-summary.json'
if ([IO.Path]::GetPathRoot($matrixOutputRoot) -cne [IO.Path]::GetPathRoot($wrapperRoot)) {
    throw 'Summary output must stay on the evidence drive.'
}
if (-not (Test-Path -LiteralPath $runPath -PathType Leaf)) { throw "Missing matrix run record: $runPath" }
if (Test-Path -LiteralPath $summaryPath) { throw "Summary already exists: $summaryPath" }

$run = Get-Content -LiteralPath $runPath -Raw | ConvertFrom-Json
$rawRows = ConvertTo-Array $run.rows
if ($run.schema -ne 1 -or $rawRows.Count -lt 1) {
    throw 'Matrix run record is not a populated schema1 record.'
}

$passedRows = @($rawRows | Where-Object { $_.status -ceq 'PASS' })
$failedRows = @($rawRows | Where-Object { $_.status -ceq 'FAIL' })
$groups = @(
    foreach ($fixture in @('NormalMultiPage', 'MetadataHeavy')) {
        foreach ($version in @('native-worker.20261110.20', 'native-worker.20261110.21')) {
            $rows = @($passedRows | Where-Object {
                $_.fixture -ceq $fixture -and $_.version -ceq $version -and
                $null -ne $_.measurementSummary
            })
            if ($rows.Count -eq 0) { continue }
            [ordered]@{
                Fixture = $fixture
                Version = $version
                RowCount = $rows.Count
                Stats = Get-GroupStats $rows
                LifetimePeakCompleteRows = @($rows | Where-Object { $_.measurementSummary.LifetimePeakCountersComplete }).Count
                LifetimePeakLimitedRows = @($rows | Where-Object { -not $_.measurementSummary.LifetimePeakCountersComplete }).Count
            }
        }
    }
)

$limitations = @(
    foreach ($row in $rawRows) {
        if ($null -eq $row.measurementSummary) { continue }
        foreach ($limitation in (ConvertTo-Array $row.measurementSummary.limitations)) {
            [ordered]@{
                Index = [int]$row.index
                MatrixPosition = [int]$row.matrixPosition
                Limitation = [string]$limitation
            }
        }
    }
)
$diagnostics = @(
    foreach ($row in $failedRows) {
        $available = [ordered]@{}
        if ((Test-RowProperty $row 'stdout') -and $null -ne $row.stdout) { $available.Stdout = $row.stdout }
        if ((Test-RowProperty $row 'stderr') -and $null -ne $row.stderr) { $available.Stderr = $row.stderr }
        if ((Test-RowProperty $row 'measurement') -and $null -ne $row.measurement) { $available.Measurement = $row.measurement }
        if ((Test-RowProperty $row 'config') -and $null -ne $row.config) { $available.Config = $row.config }
        [ordered]@{
            Index = [int]$row.index
            MatrixPosition = [int]$row.matrixPosition
            Fixture = [string]$row.fixture
            Version = [string]$row.version
            Ordinal = [int]$row.ordinal
            ExitCode = $row.exitCode
            TimedOut = [bool]$row.timedOut
            Violations = ConvertTo-Array $row.violations
            MeasurementSummaryPresent = $null -ne $row.measurementSummary
            AvailableArtifacts = $available
        }
    }
)

$summary = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    RunStatus = [string]$run.status
    TotalRows = $rawRows.Count
    PassedRows = $passedRows.Count
    FailedRows = $failedRows.Count
    StoppedAtFirstFailure = [bool]$run.stoppedAtFirstFailure
    RawRows = $rawRows
    DescriptiveGroups = $groups
    Limitations = $limitations
    FailureDiagnostics = $diagnostics
    Scope = 'Twelve or fewer local draw.io worker requests only. Raw rows are preserved before descriptive statistics. Request elapsed, CPU kernel/user, process wall lifetime, sampled working/private commit, and OS-observed peaks are distinct. Missing counters are not replaced with zero. Limited lifetime peak rows are retained and are not represented as complete lifetime peaks. No host/editor/Tauri/global memory or speed claim.'
}
[IO.File]::WriteAllText($summaryPath, ($summary | ConvertTo-Json -Depth 24), [Text.UTF8Encoding]::new($false))
$summary | ConvertTo-Json -Depth 8
