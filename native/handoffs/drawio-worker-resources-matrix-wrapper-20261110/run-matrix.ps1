param(
    [Parameter(Mandatory = $true)][string]$OutputRoot,
    [Parameter(Mandatory = $true)][ValidateSet('RUN_FIXED_12_DRAWIO_WORKER_MATRIX')][string]$Activation
)

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

function Get-FileIdentity {
    param([Parameter(Mandatory = $true)][string]$Path)
    $item = Get-Item -LiteralPath $Path
    if ($null -eq $item) { throw "Missing file: $Path" }
    [ordered]@{
        Path = $item.FullName
        Bytes = [Int64]$item.Length
        SHA256 = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
        LastWriteTimeUtc = $item.LastWriteTimeUtc.ToString('o')
    }
}

function Get-Inventory {
    param([Parameter(Mandatory = $true)][string]$Root)
    $rootFull = [IO.Path]::GetFullPath($Root)
    @(
        foreach ($item in (Get-ChildItem -LiteralPath $rootFull -Recurse -File | Sort-Object FullName)) {
            [ordered]@{
                RelativePath = [IO.Path]::GetRelativePath($rootFull, $item.FullName)
                Bytes = [Int64]$item.Length
                SHA256 = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
                LastWriteTimeUtc = $item.LastWriteTimeUtc.ToString('o')
            }
        }
    )
}

function Assert-InventoryEqual {
    param(
        [Parameter(Mandatory = $true)][object[]]$Before,
        [Parameter(Mandatory = $true)][object[]]$After,
        [Parameter(Mandatory = $true)][string]$Name
    )
    if ($Before.Count -ne $After.Count) {
        throw "$Name inventory count differs: $($Before.Count) -> $($After.Count)"
    }
    for ($index = 0; $index -lt $Before.Count; $index++) {
        foreach ($property in @('RelativePath', 'Bytes', 'SHA256', 'LastWriteTimeUtc')) {
            if ([string]$Before[$index].$property -cne [string]$After[$index].$property) {
                throw "$Name differs at $index/$($Before[$index].RelativePath): $property"
            }
        }
    }
}

function Initialize-RunRecord {
    param(
        [Parameter(Mandatory = $true)][string]$RepositoryRoot,
        [Parameter(Mandatory = $true)][object]$Tester,
        [Parameter(Mandatory = $true)][object]$PreflightRecord,
        [Parameter(Mandatory = $true)][string]$OutputRoot,
        [Parameter(Mandatory = $true)][object[]]$PackageProtection
    )
    [ordered]@{
        Schema = 1
        Status = 'RUNNING'
        CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
        RepositoryRoot = $RepositoryRoot
        Tester = $Tester
        PreflightRecord = $PreflightRecord
        OutputRoot = $OutputRoot
        SampleTimeoutSeconds = 120
        CleanupTimeoutSeconds = 10
        PackageProtection = $PackageProtection
        Rows = [Collections.Generic.List[object]]::new()
    }
}

function Save-RunRecord {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][object]$Record
    )
    [IO.File]::WriteAllText($Path, ($Record | ConvertTo-Json -Depth 24), [Text.UTF8Encoding]::new($false))
}

function Complete-RunRecord {
    param(
        [Parameter(Mandatory = $true)][object]$Record,
        [Parameter(Mandatory = $true)][bool]$StoppedAtFirstFailure,
        [AllowEmptyString()][string]$ErrorText = ''
    )
    $Record.StoppedAtFirstFailure = $StoppedAtFirstFailure
    $Record.CompletedRows = $Record.Rows.Count
    $Record.PassedRows = @($Record.Rows | Where-Object { $_.Status -ceq 'PASS' }).Count
    $Record.FailedRows = @($Record.Rows | Where-Object { $_.Status -ceq 'FAIL' }).Count
    $Record.Status = 'FAIL'
    if (-not [string]::IsNullOrWhiteSpace($ErrorText)) { $Record.Error = $ErrorText }
    elseif (-not $StoppedAtFirstFailure -and $Record.CompletedRows -eq 12 -and $Record.PassedRows -eq 12 -and
        $Record.FailedRows -eq 0 -and (Get-OptionalProperty $Record 'PackageProtectionEqual') -eq $true) {
        $Record.Status = 'PASS'
    }
    $Record.FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
}

function Get-WarningViolations {
    param(
        [Parameter(Mandatory = $true)][object]$Measurement,
        [Parameter(Mandatory = $true)][object]$Fixture,
        [Parameter(Mandatory = $true)][string]$Version
    )
    $actual = ConvertTo-Array $Measurement.actualWarnings
    $expected = ConvertTo-Array $Fixture.expectedWarnings.$Version
    if ($actual.Count -ne $expected.Count) { return @('warnings') }
    for ($index = 0; $index -lt $actual.Count; $index++) {
        if ([string]$actual[$index].code -cne [string]$expected[$index].code -or
            [string]$actual[$index].message -cne [string]$expected[$index].message -or
            [UInt32]$actual[$index].start -ne [UInt32]$expected[$index].start -or
            [UInt32]$actual[$index].end -ne [UInt32]$expected[$index].end) {
            return @('warnings')
        }
    }
    return @()
}

function Get-MeasurementSummary {
    param([Parameter(Mandatory = $true)][object]$Measurement)
    $probe = Get-OptionalProperty $Measurement 'probe'
    $summary = Get-OptionalProperty $probe 'summary'
    $finalTimes = Get-OptionalProperty $probe 'finalTimes'
    $times = Get-OptionalProperty $finalTimes 'times'
    $result = [ordered]@{
        Failures = ConvertTo-Array (Get-OptionalProperty $Measurement 'failures')
        Limitations = ConvertTo-Array (Get-OptionalProperty $Measurement 'limitations')
        ActualWarnings = ConvertTo-Array (Get-OptionalProperty $Measurement 'actualWarnings')
    }
    foreach ($name in @('ExportElapsedNS', 'SampleIntervalNS', 'MaxAttempts')) {
        $result[$name] = Get-OptionalProperty $Measurement $name
    }
    $result.ProbeAttempts = Get-OptionalProperty $summary 'attempts'
    $result.ProbeFailedReads = Get-OptionalProperty $summary 'failedReads'
    foreach ($name in @('SampledMaxWorkingSetBytes', 'SampledMaxPrivateUsageBytes',
        'ObservedOSPeakWorkingSetBytes', 'ObservedOSPeakPagefileBytes', 'FinalMemoryReadOK',
        'FinalCPUReadOK', 'LifetimePeakCountersComplete', 'PeakCoverage')) {
        $result[$name] = Get-OptionalProperty $summary $name
    }
    $result.CPUKernel100ns = Get-OptionalProperty $times 'kernel100ns'
    $result.CPUUser100ns = Get-OptionalProperty $times 'user100ns'
    $result.ProcessLifetimeNS = Get-OptionalProperty $times 'lifetimeNS'
    return $result
}

function Add-RowArtifacts {
    param([object]$Row, [string]$SampleRoot)
    if ($null -eq $Row -or [string]::IsNullOrEmpty($SampleRoot)) { return }
    $errors = [Collections.Generic.List[string]]::new()
    $paths = [ordered]@{
        Stdout = (Join-Path $SampleRoot 'stdout.txt')
        Stderr = (Join-Path $SampleRoot 'stderr.txt')
        Measurement = (Join-Path $SampleRoot 'measurement\measurement.json')
    }
    foreach ($name in $paths.Keys) {
        try {
            $Row[$name] = if (Test-Path -LiteralPath $paths[$name] -PathType Leaf) {
                Get-FileIdentity $paths[$name]
            } else { $null }
        } catch { $errors.Add("${name}: $($_.Exception.Message)") }
    }
    try {
        $configPath = Join-Path $SampleRoot 'measurement\config'
        $Row.Config = if (Test-Path -LiteralPath $configPath -PathType Container) {
            [ordered]@{ Path = $configPath; Inventory = @(Get-Inventory $configPath) }
        } else { $null }
    } catch { $errors.Add("Config: $($_.Exception.Message)") }
    $Row.ArtifactErrors = $errors.ToArray()
}

function Invoke-OwnedTester {
    param([object]$Command, [string]$StdoutPath, [string]$StderrPath)
    $process = $null
    $ownedHandle = $null
    $result = [ordered]@{
        ExitCode = $null; TimedOut = $false; CleanupError = $null; ProcessError = $null
        TesterIdentity = $null; IdentityErrors = @()
    }
    try {
        $process = Start-Process -FilePath $Command.FilePath -ArgumentList $Command.Arguments `
            -WorkingDirectory $Command.WorkingDirectory -WindowStyle Hidden `
            -RedirectStandardOutput $StdoutPath -RedirectStandardError $StderrPath -PassThru
        # Acquire and hold the exact returned process handle before optional identity queries.
        $ownedHandle = $process.SafeHandle
        if ($ownedHandle.IsInvalid) { throw 'Returned tester process handle is invalid.' }
        $result.TesterIdentity = [ordered]@{ PID = [int]$process.Id; ProcessStartTimeUtc = $null; ProcessPath = $null }
        try { $result.TesterIdentity.ProcessStartTimeUtc = $process.StartTime.ToUniversalTime().ToString('o') }
        catch { $result.IdentityErrors += "StartTime: $($_.Exception.Message)" }
        try { $result.TesterIdentity.ProcessPath = $process.Path }
        catch { $result.IdentityErrors += "Path: $($_.Exception.Message)" }
        if (-not $process.WaitForExit(120000)) { $result.TimedOut = $true }
    }
    catch { $result.ProcessError = $_.Exception.Message }
    finally {
        if ($null -ne $process) {
            try {
                if (-not $process.HasExited) {
                    $process.Kill()
                    if (-not $process.WaitForExit(10000)) {
                        throw 'Exact tester object did not exit within 10 seconds after Kill.'
                    }
                }
                $result.ExitCode = [int]$process.ExitCode
            }
            catch { $result.CleanupError = $_.Exception.Message }
            finally {
                try { $process.Dispose() }
                catch { $result.CleanupError = "$($result.CleanupError) Dispose: $($_.Exception.Message)" }
                $ownedHandle = $null
            }
        }
    }
    return $result
}

$exitCode = 0
$runRecord = $null
$runPath = $null
$packageBefore = $null
$packageRoots = @()
$row = $null
$sampleRoot = $null
$runError = ''
$stoppedAtFirstFailure = $false
$oldEnvironment = $null
$environmentNames = @(
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_MEASUREMENT',
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_FIXTURE',
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_VERSION',
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_ORDINAL',
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_SOURCE_SHA256',
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_TESTER_SHA256',
    'LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS_OUTPUT'
)
$oldEnvironment = @{}
foreach ($name in $environmentNames) {
    $oldEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

try {
    if ($Activation -cne 'RUN_FIXED_12_DRAWIO_WORKER_MATRIX') {
        throw 'The fixed draw.io worker matrix is not activated.'
    }
    $wrapperRoot = [IO.Path]::GetFullPath($PSScriptRoot)
    $repositoryRoot = [IO.Path]::GetFullPath((Join-Path $wrapperRoot '..\..\..'))
    $expectedRoot = 'F:\AIproject\LevNote'
    if ($repositoryRoot -cne $expectedRoot -or
        -not (Test-Path -LiteralPath (Join-Path $repositoryRoot 'AGENTS.md') -PathType Leaf) -or
        -not (Test-Path -LiteralPath (Join-Path $repositoryRoot 'native\scripts\go.ps1') -PathType Leaf)) {
        throw "Repository root arithmetic or marker differs: $repositoryRoot"
    }
    if ([IO.Path]::GetFullPath((Get-Location)) -cne $repositoryRoot) {
        throw "Run the matrix wrapper from the repository root: $repositoryRoot"
    }
    $outputRoot = [IO.Path]::GetFullPath($OutputRoot)
    if (Test-Path -LiteralPath $outputRoot) { throw "Matrix output directory must be absent: $outputRoot" }
    if ([IO.Path]::GetPathRoot($outputRoot) -cne [IO.Path]::GetPathRoot($wrapperRoot)) {
        throw 'Matrix output must stay on the evidence drive.'
    }

    $testerPath = [IO.Path]::GetFullPath((Join-Path $wrapperRoot '..\drawio-worker-resources-harness-final-validation-20261110\drawio-resources.test.exe'))
    $preflightPath = [IO.Path]::GetFullPath((Join-Path $wrapperRoot '..\drawio-worker-resources-harness-final-validation-20261110\preflight-output\preflight.json'))
    $expectedTesterSha256 = 'e737a93cb11331b09b83113984b9d279aef6cab129ff590063fd21fdee845398'
    $expectedPreflightSha256 = '1722314a3c99f4f70eb42f9e8b13078a3230a8f9642d2359d1884e9a9688911c'
    $tester = Get-FileIdentity $testerPath
    $preflightIdentity = Get-FileIdentity $preflightPath
    if ($tester.SHA256 -cne $expectedTesterSha256 -or $tester.Bytes -ne 7528448) {
        throw "Exact tester identity differs: $($tester.SHA256) $($tester.Bytes)"
    }
    if ($preflightIdentity.SHA256 -cne $expectedPreflightSha256 -or $preflightIdentity.Bytes -ne 8797) {
        throw "Exact preflight identity differs: $($preflightIdentity.SHA256) $($preflightIdentity.Bytes)"
    }

    $preflight = Get-Content -LiteralPath $preflightPath -Raw | ConvertFrom-Json
    $preflightFailures = ConvertTo-Array $preflight.failures
    if ($preflight.schema -ne 1 -or $preflight.status -cne 'PASS' -or $preflightFailures.Count -ne 0) {
        throw 'Pinned preflight is not a clean schema1 PASS record.'
    }
    $fixtureRows = @{}
    foreach ($fixture in @($preflight.fixtures)) { $fixtureRows[[string]$fixture.name] = $fixture }
    if ($fixtureRows.Count -ne 2) { throw 'Pinned preflight must contain exactly two fixtures.' }

    $packages = @(
        [ordered]@{ Version = 'native-worker.20261110.20'; WorkerSHA256 = '70c15942339ecf074f81192c01ce652b19fdbc7dbdfaba139fed22e66c46fe9a' },
        [ordered]@{ Version = 'native-worker.20261110.21'; WorkerSHA256 = '16e51c34688d39625626eab8a4d68aeb507ecc1cb9ffba7ea7478f9b8c0ff5d4' }
    )
    $packageRoots = @()
    foreach ($package in $packages) {
        $suffix = [string]$package.Version -replace '^native-worker\.', ''
        $root = Join-Path $repositoryRoot ("native\bin\levnote-native-worker-" + $suffix)
        if (-not (Test-Path -LiteralPath $root -PathType Container)) { throw "Missing package root: $root" }
        if ($outputRoot.StartsWith($root.TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Matrix output cannot be inside a protected package.'
        }
        $packageRoots += [ordered]@{ Version = $package.Version; Root = $root; WorkerSHA256 = $package.WorkerSHA256 }
    }
    $matrix = @(
        [ordered]@{ Index = 1;  Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.20'; Ordinal = 1 },
        [ordered]@{ Index = 2;  Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.21'; Ordinal = 1 },
        [ordered]@{ Index = 3;  Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.21'; Ordinal = 2 },
        [ordered]@{ Index = 4;  Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.20'; Ordinal = 2 },
        [ordered]@{ Index = 5;  Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.20'; Ordinal = 3 },
        [ordered]@{ Index = 6;  Fixture = 'NormalMultiPage'; Version = 'native-worker.20261110.21'; Ordinal = 3 },
        [ordered]@{ Index = 7;  Fixture = 'MetadataHeavy';  Version = 'native-worker.20261110.21'; Ordinal = 1 },
        [ordered]@{ Index = 8;  Fixture = 'MetadataHeavy';  Version = 'native-worker.20261110.20'; Ordinal = 1 },
        [ordered]@{ Index = 9;  Fixture = 'MetadataHeavy';  Version = 'native-worker.20261110.20'; Ordinal = 2 },
        [ordered]@{ Index = 10; Fixture = 'MetadataHeavy';  Version = 'native-worker.20261110.21'; Ordinal = 2 },
        [ordered]@{ Index = 11; Fixture = 'MetadataHeavy';  Version = 'native-worker.20261110.21'; Ordinal = 3 },
        [ordered]@{ Index = 12; Fixture = 'MetadataHeavy';  Version = 'native-worker.20261110.20'; Ordinal = 3 }
    )
    $packageBefore = @(
        foreach ($package in $packageRoots) {
            [ordered]@{ Version = $package.Version; Root = $package.Root; Inventory = Get-Inventory $package.Root }
        }
    )

    New-Item -ItemType Directory -Path $outputRoot -ErrorAction Stop | Out-Null
    $runRecord = Initialize-RunRecord $repositoryRoot $tester $preflightIdentity $outputRoot $packageBefore
    $runPath = Join-Path $outputRoot 'matrix-run.json'
    Save-RunRecord $runPath $runRecord
    $stoppedAtFirstFailure = $false

    foreach ($sample in $matrix) {
        $row = [ordered]@{
            Index = [int]$sample.Index
            MatrixPosition = [int]$sample.Index
            Fixture = [string]$sample.Fixture
            Version = [string]$sample.Version
            Ordinal = [int]$sample.Ordinal
            Status = 'RUNNING'
            StartedAtUtc = [DateTime]::UtcNow.ToString('o')
        }
        $runRecord.Rows.Add($row)
        $fixtureName = [string]$sample.Fixture
        $fixture = $fixtureRows[$fixtureName]
        $version = [string]$sample.Version
        $ordinal = [int]$sample.Ordinal
        $sampleRoot = Join-Path $outputRoot ("sample-{0:D2}" -f [int]$sample.Index)
        $measurementRoot = Join-Path $sampleRoot 'measurement'
        if ((Test-Path -LiteralPath $sampleRoot) -or (Test-Path -LiteralPath $measurementRoot)) {
            throw "Fresh sample or measurement path already exists: $sampleRoot"
        }
        New-Item -ItemType Directory -Path $sampleRoot -ErrorAction Stop | Out-Null
        $stdoutPath = Join-Path $sampleRoot 'stdout.txt'
        $stderrPath = Join-Path $sampleRoot 'stderr.txt'
        $command = [ordered]@{
            FilePath = $testerPath
            Arguments = @('-test.run', '^TestDrawioWorkerResourceMeasurement$', '-test.count=1', '-test.v')
            WorkingDirectory = $repositoryRoot
            WindowStyle = 'Hidden'
            TimeoutSeconds = 120
            Environment = [ordered]@{
                LEVNOTE_DRAWIO_WORKER_RESOURCES_MEASUREMENT = '1'
                LEVNOTE_DRAWIO_WORKER_RESOURCES_FIXTURE = $fixtureName
                LEVNOTE_DRAWIO_WORKER_RESOURCES_VERSION = $version
                LEVNOTE_DRAWIO_WORKER_RESOURCES_ORDINAL = [string]$ordinal
                LEVNOTE_DRAWIO_WORKER_RESOURCES_SOURCE_SHA256 = [string]$fixture.sourceSha256
                LEVNOTE_DRAWIO_WORKER_RESOURCES_TESTER_SHA256 = $expectedTesterSha256
                LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS_OUTPUT = $measurementRoot
            }
        }
        $row.Command = $command
        Save-RunRecord $runPath $runRecord

        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_MEASUREMENT', '1', 'Process')
        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_FIXTURE', $fixtureName, 'Process')
        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_VERSION', $version, 'Process')
        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_ORDINAL', [string]$ordinal, 'Process')
        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_SOURCE_SHA256', [string]$fixture.sourceSha256, 'Process')
        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_TESTER_SHA256', $expectedTesterSha256, 'Process')
        [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS_OUTPUT', $measurementRoot, 'Process')

        try {
            $processResult = Invoke-OwnedTester $command $stdoutPath $stderrPath
            foreach ($name in $processResult.Keys) { $row[$name] = $processResult[$name] }
        }
        finally {
            foreach ($name in $environmentNames) {
                [Environment]::SetEnvironmentVariable($name, $oldEnvironment[$name], 'Process')
            }
        }

        $row.FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
        Add-RowArtifacts $row $sampleRoot
        $measurementPath = Join-Path $measurementRoot 'measurement.json'

        $violations = [Collections.Generic.List[string]]::new()
        if ($row.TimedOut) { $violations.Add('tester timeout') }
        if ($null -eq $row.ExitCode -or $row.ExitCode -ne 0) { $violations.Add("tester exit $($row.ExitCode)") }
        if ($null -ne $row.ProcessError) { $violations.Add('tester process failure') }
        if ($null -ne $row.CleanupError) { $violations.Add('cleanup failure') }
        if ($row.ArtifactErrors.Count -ne 0) { $violations.Add('artifact collection failure') }
        if ($null -eq $row.Measurement) {
            $violations.Add('measurement.json missing')
        }
        else {
            $measurement = Get-Content -LiteralPath $measurementPath -Raw | ConvertFrom-Json
            $row.MeasurementSummary = Get-MeasurementSummary $measurement
            $row.MeasurementStatus = [string]$measurement.status
            if ($measurement.schema -ne 1 -or $measurement.status -cne 'PASS') { $violations.Add('record status') }
            if ([int]$measurement.matrixPosition -ne [int]$sample.Index) { $violations.Add('matrix position') }
            if ([string]$measurement.fixture -cne $fixtureName -or
                [string]$measurement.version -cne $version -or
                [int]$measurement.ordinal -ne $ordinal) { $violations.Add('sample identity') }
            if ([string]$measurement.sourceSHA256 -cne [string]$fixture.sourceSha256 -or
                [string]$measurement.sourceSHA256 -cne [string]$measurement.expectedSourceSHA256 -or
                [int]$measurement.sourceBytes -ne [int]$fixture.sourceBytes) { $violations.Add('source identity') }
            if ([string]$measurement.tester.SHA256 -cne $expectedTesterSha256) { $violations.Add('tester identity') }
            if ([int]$measurement.actualSVGBytes -ne [int]$fixture.svgBytes -or
                [string]$measurement.actualSVGSHA256 -cne [string]$fixture.svgSha256) { $violations.Add('actual SVG oracle') }
            foreach ($warningViolation in (Get-WarningViolations $measurement $fixture $version)) { $violations.Add($warningViolation) }
            if ([int]$measurement.admissionCalls -ne 1 -or [int]$measurement.verifyCalls -ne 1 -or
                [int]$measurement.startCalls -ne 1 -or [int]$measurement.waitCalls -ne 1) { $violations.Add('operation counts') }
            if ((ConvertTo-Array $measurement.failures).Count -ne 0) { $violations.Add('record failures') }

            if ($row.MeasurementSummary.FinalMemoryReadOK -ne $true -or
                $row.MeasurementSummary.FinalCPUReadOK -ne $true -or
                $null -eq $row.MeasurementSummary.ProbeFailedReads -or
                $row.MeasurementSummary.ProbeFailedReads -ne 0 -or
                $null -eq $row.MeasurementSummary.CPUKernel100ns -or
                $null -eq $row.MeasurementSummary.CPUUser100ns -or
                $null -eq $row.MeasurementSummary.ProcessLifetimeNS) { $violations.Add('probe reads') }
        }

        $row.Violations = $violations.ToArray()
        if ($violations.Count -ne 0) {
            $row.Status = 'FAIL'
            $runRecord.Status = 'FAIL'
            $stoppedAtFirstFailure = $true
        } else { $row.Status = 'PASS' }
        Save-RunRecord $runPath $runRecord
        if ($stoppedAtFirstFailure) { break }
    }
}
catch {
    $runError = $_.Exception.Message
    $stoppedAtFirstFailure = $true
    if ($null -ne $row) {
        $row.Status = 'FAIL'
        $row.Error = $runError
        $row.FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
        $row.Violations = @((ConvertTo-Array (Get-OptionalProperty $row 'Violations'))) + @('wrapper exception')
        Add-RowArtifacts $row $sampleRoot
    }
    [Console]::Error.WriteLine($runError)
    $exitCode = 1
}
finally {
    foreach ($name in $environmentNames) {
        [Environment]::SetEnvironmentVariable($name, $oldEnvironment[$name], 'Process')
    }
    if ($null -ne $packageBefore) {
        $packageAfter = [Collections.Generic.List[object]]::new()
        $protectionErrors = [Collections.Generic.List[string]]::new()
        foreach ($package in $packageBefore) {
            try {
                $after = [ordered]@{ Version = $package.Version; Root = $package.Root; Inventory = @(Get-Inventory $package.Root) }
                $packageAfter.Add($after)
                Assert-InventoryEqual @($package.Inventory) @($after.Inventory) "package $($package.Version)"
            } catch { $protectionErrors.Add("$($package.Version): $($_.Exception.Message)") }
        }
        if ($null -ne $runRecord) {
            $runRecord.PackageAfter = $packageAfter.ToArray()
            $runRecord.PackageProtectionErrors = $protectionErrors.ToArray()
            $runRecord.PackageProtectionEqual = $protectionErrors.Count -eq 0 -and $packageAfter.Count -eq $packageBefore.Count
            Complete-RunRecord $runRecord $stoppedAtFirstFailure $runError
            if ($runRecord.Status -cne 'PASS') { $exitCode = 1 }
            try { Save-RunRecord $runPath $runRecord }
            catch { [Console]::Error.WriteLine("Final run record: $($_.Exception.Message)"); $exitCode = 1 }
        }
        elseif ($protectionErrors.Count -ne 0) {
            [Console]::Error.WriteLine(($protectionErrors -join '; '))
            $exitCode = 1
        }
    }
}

exit $exitCode
