$ErrorActionPreference = 'Stop'
$evidenceRoot = 'F:\AIproject\LevNote\native\artifacts\acceptance\native-reading-release24-20261110'
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$closedPath = $evidenceRoot + '-closed-inventory.json'
Assert-AbsentAbsolutePath $closedPath 'external closed inventory' | Out-Null
$aggregate = Read-JsonFile (Join-Path $evidenceRoot 'aggregate.json')
if (-not $aggregate.Pass -or $aggregate.SuccessfulAttempt -ne 2) { throw 'Expected accepted final metadata aggregate attempt2' }
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
    foreach ($name in @('baseline','package','host-smoke')) {
        foreach ($prefix in @('stage','outer')) {
            $path = Join-Path $evidenceRoot "$prefix-$name-invocation.json"
            $record = Read-JsonFile $path
            if ($record.ExitCode -ne 0) { throw "Expected successful $prefix $name" }
            [ordered]@{Name="$prefix-$name";Path=$path;ExitCode=$record.ExitCode}
        }
    }
    foreach ($attempt in @(1,2)) {
        foreach ($prefix in @('stage','outer')) {
            $path = Join-Path ($evidenceRoot + "-aggregate-capture-attempt-$attempt") "$prefix-aggregate-attempt-$attempt-invocation.json"
            $record = Read-JsonFile $path
            $expectedExit = if ($attempt -eq 1) { 1 } else { 0 }
            if ($record.ExitCode -ne $expectedExit) { throw "Unexpected aggregate exit: $path" }
            [ordered]@{Name="$prefix-aggregate-attempt-$attempt";Path=$path;ExitCode=$record.ExitCode}
        }
    }
)
$context = Read-DeliveryContext $evidenceRoot
$protectedNow = Get-ProtectedReleaseFacts $context.NativeRoot
Assert-ProtectedReleaseUnchanged $context.ProtectedReleaseFacts $protectedNow
$roots = @(
    foreach ($path in @($evidenceRoot, ($evidenceRoot + '-aggregate-capture-attempt-1'), ($evidenceRoot + '-aggregate-capture-attempt-2'))) {
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
    Scope = 'Closed external inventory after package/host/aggregate and all stage wrappers exited. Direct .24 evidence and both preserved aggregate capture siblings; no writer logs within inventoried roots remain open.'
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
