$ErrorActionPreference = 'Stop'
$evidenceRoot = 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110'
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$closedPath = $evidenceRoot + '-closed-inventory.json'
Assert-AbsentAbsolutePath $closedPath 'external closed inventory' | Out-Null
$packageGate = Read-PackageGate $evidenceRoot
$repairAdopted = -not (Test-Path -LiteralPath (
    Join-Path $evidenceRoot 'package-result.json') -PathType Leaf)
$aggregate = Read-JsonFile (Join-Path $evidenceRoot 'aggregate.json')
if (-not $aggregate.Pass -or $aggregate.SuccessfulAttempt -notin @(1,2)) { throw 'Expected successful final aggregate attempt' }
foreach ($entry in @($aggregate.Evidence)) {
    $actual = Get-FileFingerprint $entry.Path
    if (-not (Test-FingerprintEqual $entry $actual)) { throw "Aggregate evidence changed after its writer exited: $($entry.Path)" }
}
$stageRecords = @(
    foreach ($name in @('parse','initialize')) {
        $path = Join-Path $evidenceRoot "$name-invocation.json"
        $record = Read-JsonFile $path
        if ($record.ExitCode -ne 0) { throw "Expected successful $name" }
        [ordered]@{Name=$name;Path=$path;ExitCode=$record.ExitCode}
    }
    if ($repairAdopted) {
        foreach ($repairParse in @(
            [pscustomobject]@{ Name = 'repair-parse'; Expected = 1 }
            [pscustomobject]@{ Name = 'repair-correction-parse'; Expected = 0 }
            [pscustomobject]@{ Name = 'repair-final-parse'; Expected = 0 }
            [pscustomobject]@{ Name = 'repair-post-adoption-parse'; Expected = 0 }
        )) {
            $path = Join-Path $evidenceRoot ($repairParse.Name + '-invocation.json')
            $record = Read-JsonFile $path
            if ($record.ExitCode -ne $repairParse.Expected) {
                throw "Unexpected $($repairParse.Name) exit: expected $($repairParse.Expected), actual $($record.ExitCode)"
            }
            [ordered]@{Name=$repairParse.Name;Path=$path;ExitCode=$record.ExitCode}
        }
    } elseif (Test-Path -LiteralPath (Join-Path $evidenceRoot 'repair-parse-invocation.json') -PathType Leaf) {
        $path = Join-Path $evidenceRoot 'repair-parse-invocation.json'
        $record = Read-JsonFile $path
        if ($record.ExitCode -ne 0) { throw 'Expected successful direct repair parse' }
        [ordered]@{Name='repair-parse';Path=$path;ExitCode=$record.ExitCode}
    }
    foreach ($name in @('baseline','package','host-smoke')) {
        foreach ($prefix in @('stage','outer')) {
            $path = Join-Path $evidenceRoot "$prefix-$name-invocation.json"
            $record = Read-JsonFile $path
            $expectedExit = if ($repairAdopted -and $name -eq 'package') { 1 } else { 0 }
            if ($record.ExitCode -ne $expectedExit) {
                throw "Unexpected $prefix $name exit: expected $expectedExit, actual $($record.ExitCode)"
            }
            [ordered]@{Name="$prefix-$name";Path=$path;ExitCode=$record.ExitCode}
        }
    }
    foreach ($attempt in 1..([int]$aggregate.SuccessfulAttempt)) {
        foreach ($prefix in @('stage','outer')) {
            $path = Join-Path ($evidenceRoot + "-aggregate-capture-attempt-$attempt") "$prefix-aggregate-attempt-$attempt-invocation.json"
            $record = Read-JsonFile $path
            $expectedExit = if ($attempt -eq [int]$aggregate.SuccessfulAttempt) { 0 } else { 1 }
            if ($record.ExitCode -ne $expectedExit) { throw "Unexpected aggregate exit: $path" }
            [ordered]@{Name="$prefix-aggregate-attempt-$attempt";Path=$path;ExitCode=$record.ExitCode}
        }
    }
)
$context = Read-DeliveryContext $evidenceRoot
$protectedNow = Get-ProtectedReleaseFacts $context.NativeRoot
Assert-ProtectedReleaseUnchanged $context.ProtectedReleaseFacts $protectedNow
$roots = @(
    $closedRootPaths = @($evidenceRoot) + @(foreach ($attempt in 1..([int]$aggregate.SuccessfulAttempt)) { $evidenceRoot + "-aggregate-capture-attempt-$attempt" })
    foreach ($path in $closedRootPaths) {
        $files = @(Get-DirectoryInventory $path)
        [ordered]@{Root=$path;FileCount=$files.Count;TotalBytes=(Get-InventoryTotalBytes $files);InventoryIdentity=(Get-InventoryIdentity $files);Files=$files}
    }
)
[int]$closedFileCount = 0
[Int64]$closedTotalBytes = 0
foreach ($entry in $roots) {
    $closedFileCount += [int]$entry.FileCount
    $closedTotalBytes += [Int64]$entry.TotalBytes
}
$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    PackageMode = $(if ($repairAdopted) { 'metadata-repair adoption of preserved exit1 package' } else { 'direct successful package' })
    OriginalPackageFailurePreserved = $repairAdopted
    PackageGateStatus = [string]$packageGate.Status
    ProductRerun = if ($repairAdopted) { [bool]$packageGate.ProductRerun } else { $false }
    Scope = 'Closed external inventory after package/host/aggregate and all stage wrappers exited. Direct .26 evidence and its completed aggregate capture siblings; any metadata repair preserves the original exit1 package without a product rerun. No writer logs within inventoried roots remain open.'
    AggregateEvidenceRecordsChecked = @($aggregate.Evidence).Count
    AggregateEvidenceUnchanged = $true
    ProtectedReleaseUnchanged = $true
    StageExitCodes = $stageRecords
    TotalFileCount = $closedFileCount
    TotalBytes = $closedTotalBytes
    Roots = $roots
}
Write-JsonFile $closedPath $result 18
[ordered]@{
    ClosedInventory = Get-FileFingerprint $closedPath
    TotalFileCount = $result.TotalFileCount
    TotalBytes = $result.TotalBytes
    Roots = @($roots | ForEach-Object { [ordered]@{Root=$_.Root;FileCount=$_.FileCount;TotalBytes=$_.TotalBytes;InventoryIdentity=$_.InventoryIdentity} })
} | ConvertTo-Json -Depth 8
