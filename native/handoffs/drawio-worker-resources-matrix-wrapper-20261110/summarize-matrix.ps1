param([Parameter(Mandatory = $true)][string]$MatrixOutputRoot)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function ConvertTo-Array {
    param($Value)
    if ($null -eq $Value) { return ,@() }
    return ,@($Value)
}

function Get-OptionalProperty {
    param($Value, [string]$Name)
    if ($null -eq $Value) { return $null }
    if ($Value -is [Collections.IDictionary]) { return ,($Value[$Name]) }
    $property = $Value.PSObject.Properties[$Name]
    if ($null -ne $property) { return ,($property.Value) }
    return $null
}

function Get-Values {
    param(
        [Parameter(Mandatory = $true)][object[]]$Rows,
        [Parameter(Mandatory = $true)][string]$Property
    )
    foreach ($row in $Rows) {
        $summary = Get-OptionalProperty $row 'measurementSummary'
        $value = Get-OptionalProperty $summary $Property
        if ($null -ne $value) { $value }
    }
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
        ([decimal]$sorted[$middle] + [decimal]$sorted[$middle + 1]) / 2
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
        @{ Name = 'ExportElapsedNS'; Unit = 'ns'; Quantization = 'host stopwatch nanoseconds; managed Export request wall elapsed' },
        @{ Name = 'SampledMaxWorkingSetBytes'; Unit = 'bytes'; Quantization = 'process working-set samples' },
        @{ Name = 'SampledMaxPrivateUsageBytes'; Unit = 'bytes'; Quantization = 'process private commit samples' },
        @{ Name = 'ObservedOSPeakWorkingSetBytes'; Unit = 'bytes'; Quantization = 'OS-observed working-set peak' },
        @{ Name = 'ObservedOSPeakPagefileBytes'; Unit = 'bytes'; Quantization = 'OS-observed pagefile/commit peak' },
        @{ Name = 'CPUKernel100ns'; Unit = '100ns'; Quantization = 'Windows process CPU counter units; scheduling/accounting resolution may be coarser, zero is retained' },
        @{ Name = 'CPUUser100ns'; Unit = '100ns'; Quantization = 'Windows process CPU counter units; scheduling/accounting resolution may be coarser, zero is retained' },
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
$rawRows = ConvertTo-Array (Get-OptionalProperty $run 'rows')
if ($run.schema -ne 1) {
    throw 'Matrix run record is not a schema1 record.'
}

$passedRows = @($rawRows | Where-Object { (Get-OptionalProperty $_ 'status') -ceq 'PASS' })
$failedRows = @($rawRows | Where-Object { (Get-OptionalProperty $_ 'status') -ceq 'FAIL' })
$groups = @(
    foreach ($fixture in @('NormalMultiPage', 'MetadataHeavy')) {
        foreach ($version in @('native-worker.20261110.20', 'native-worker.20261110.21')) {
            $rows = @($passedRows | Where-Object {
                (Get-OptionalProperty $_ 'fixture') -ceq $fixture -and
                (Get-OptionalProperty $_ 'version') -ceq $version -and
                $null -ne (Get-OptionalProperty $_ 'measurementSummary')
            })
            if ($rows.Count -eq 0) { continue }
            [ordered]@{
                Fixture = $fixture
                Version = $version
                RowCount = $rows.Count
                Stats = Get-GroupStats $rows
                LifetimePeakCompleteRows = @($rows | Where-Object { (Get-OptionalProperty $_.measurementSummary 'LifetimePeakCountersComplete') -eq $true }).Count
                LifetimePeakLimitedRows = @($rows | Where-Object { (Get-OptionalProperty $_.measurementSummary 'LifetimePeakCountersComplete') -eq $false }).Count
                LifetimePeakUnknownRows = @($rows | Where-Object { $null -eq (Get-OptionalProperty $_.measurementSummary 'LifetimePeakCountersComplete') }).Count
            }
        }
    }
)

$limitations = @(
    foreach ($row in $rawRows) {
        $measurementSummary = Get-OptionalProperty $row 'measurementSummary'
        if ($null -eq $measurementSummary) { continue }
        foreach ($limitation in (ConvertTo-Array (Get-OptionalProperty $measurementSummary 'limitations'))) {
            [ordered]@{
                Index = Get-OptionalProperty $row 'index'
                MatrixPosition = Get-OptionalProperty $row 'matrixPosition'
                Limitation = [string]$limitation
            }
        }
    }
)
$diagnostics = @(
    foreach ($row in $rawRows) {
        $measurementSummary = Get-OptionalProperty $row 'measurementSummary'
        if ((Get-OptionalProperty $row 'status') -ceq 'PASS' -and $null -ne $measurementSummary) { continue }
        $available = [ordered]@{}
        foreach ($name in @('Stdout', 'Stderr', 'Measurement', 'Config')) {
            $value = Get-OptionalProperty $row $name
            if ($null -ne $value) { $available[$name] = $value }
        }
        [ordered]@{
            Index = Get-OptionalProperty $row 'index'
            MatrixPosition = Get-OptionalProperty $row 'matrixPosition'
            Fixture = Get-OptionalProperty $row 'fixture'
            Version = Get-OptionalProperty $row 'version'
            Ordinal = Get-OptionalProperty $row 'ordinal'
            Status = Get-OptionalProperty $row 'status'
            ExitCode = Get-OptionalProperty $row 'exitCode'
            TimedOut = Get-OptionalProperty $row 'timedOut'
            Violations = ConvertTo-Array (Get-OptionalProperty $row 'violations')
            Error = Get-OptionalProperty $row 'error'
            Diagnostic = if ($null -eq $measurementSummary) { 'Measurement summary absent; counters are unknown.' } else { 'Row did not pass; inspect raw evidence.' }
            MeasurementSummaryPresent = $null -ne $measurementSummary
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
    OtherRows = $rawRows.Count - $passedRows.Count - $failedRows.Count
    StoppedAtFirstFailure = Get-OptionalProperty $run 'stoppedAtFirstFailure'
    RunError = Get-OptionalProperty $run 'error'
    PackageProtectionEqual = Get-OptionalProperty $run 'packageProtectionEqual'
    RawRows = $rawRows
    DescriptiveGroups = $groups
    Limitations = $limitations
    FailureDiagnostics = $diagnostics
    Scope = 'Twelve or fewer local draw.io worker requests only. Raw rows are preserved before descriptive statistics. Request elapsed, CPU kernel/user, process wall lifetime, sampled working/private commit, and OS-observed peaks are distinct. Missing counters are not replaced with zero. Limited lifetime peak rows are retained and are not represented as complete lifetime peaks. No host/editor/Tauri/global memory or speed claim.'
}
if ((Get-OptionalProperty $run 'FixtureOnly') -eq $true) { $summary.FixtureOnly = $true }
[IO.File]::WriteAllText($summaryPath, ($summary | ConvertTo-Json -Depth 24), [Text.UTF8Encoding]::new($false))
$summary | ConvertTo-Json -Depth 8
