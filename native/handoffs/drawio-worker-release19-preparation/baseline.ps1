$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not [string]::Equals([IO.Path]::GetPathRoot($ctx.candidateDirectory), 'F:\', [StringComparison]::OrdinalIgnoreCase)) {
    throw "The fresh .19 candidate directory must be on F:, resolved to $($ctx.candidateDirectory)."
}
if (Test-Path -LiteralPath $ctx.candidateDirectory) {
    throw "Fresh .19 candidate directory already exists: $($ctx.candidateDirectory)"
}
if (Test-Path -LiteralPath (Join-Path $evidenceRoot 'baseline-result.json')) {
    throw 'Baseline already completed in this evidence directory.'
}

$sourcePaths = @(Get-DeliverySourceInputPaths $ctx.repositoryRoot $ctx.nativeRoot)
$sourceBefore = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
$formatsPrefix = (Join-Path $ctx.nativeRoot 'formats').TrimEnd('\') + '\'
$formatsCount = @($sourceBefore | Where-Object {
    ([IO.Path]::GetFullPath([string]$_.Path)).StartsWith($formatsPrefix, [StringComparison]::OrdinalIgnoreCase)
}).Count
$noticePath = [IO.Path]::GetFullPath((Join-Path $ctx.nativeRoot 'portable/NOTICE-PREAMBLE.md'))
if ($formatsCount -eq 0 -or @($sourceBefore | Where-Object {
    [IO.Path]::GetFullPath([string]$_.Path) -eq $noticePath
}).Count -ne 1) {
    throw 'Scoped build inputs must include native/formats and portable/NOTICE-PREAMBLE.md.'
}

$protectedBefore = @(Get-DirectoryInventory $ctx.protectedReleaseDirectory)
if ($protectedBefore.Count -ne [int]$ctx.protectedReleaseFileCount -or
    (Get-InventoryTotalBytes $protectedBefore) -ne [Int64]$ctx.protectedReleaseTotalBytes -or
    (Get-InventoryIdentity $protectedBefore) -cne [string]$ctx.protectedReleaseInventoryIdentity) {
    throw 'Protected .18 package does not match its accepted complete inventory.'
}
$protectedManifest = Read-JsonFile (Join-Path $ctx.protectedReleaseDirectory 'release-manifest.json')
$protectedHost = Get-FileFingerprint (Join-Path $ctx.protectedReleaseDirectory 'levnote-native.exe')
$protectedManifestFile = Get-FileFingerprint (Join-Path $ctx.protectedReleaseDirectory 'release-manifest.json')
if ($protectedManifest.version -cne $ctx.protectedReleaseVersion -or
    $protectedHost.SHA256 -cne $ctx.protectedReleaseHostSHA256 -or
    $protectedManifestFile.SHA256 -cne $ctx.protectedReleaseManifestSHA256) {
    throw 'Protected .18 version, host or release manifest differs from acceptance.'
}

$protectedEvidenceBefore = @(Get-DirectoryInventory $ctx.protectedEvidenceDirectory)
if ($protectedEvidenceBefore.Count -ne [int]$ctx.protectedEvidenceFileCount -or
    (Get-InventoryTotalBytes $protectedEvidenceBefore) -ne [Int64]$ctx.protectedEvidenceTotalBytes -or
    (Get-InventoryIdentity $protectedEvidenceBefore) -cne [string]$ctx.protectedEvidenceInventoryIdentity) {
    throw 'Protected .18 direct evidence does not match its accepted complete inventory.'
}
$protectedAggregate = Get-FileFingerprint (Join-Path $ctx.protectedEvidenceDirectory 'aggregate.json')
if ($protectedAggregate.SHA256 -cne $ctx.protectedEvidenceAggregateSHA256) {
    throw 'Protected .18 aggregate hash differs from acceptance.'
}

$acceptedDirectories = @(
    foreach ($directory in @($ctx.acceptedEvidenceDirectories)) {
        [ordered]@{
            Directory = [IO.Path]::GetFullPath([string]$directory)
            Inventory = @(Get-DirectoryInventory ([string]$directory))
        }
    }
)
$acceptedGates = @(
    foreach ($path in @($ctx.acceptedGatePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$stage2Before = @(
    foreach ($entry in @($ctx.FinalStage2Files)) {
        $absolute = [IO.Path]::GetFullPath((Join-Path $ctx.repositoryRoot ([string]$entry.RelativePath)))
        $actual = Get-FileFingerprint $absolute
        if ($actual.Bytes -ne [Int64]$entry.Bytes -or $actual.SHA256 -cne [string]$entry.SHA256) {
            throw "Final Stage2 file changed before delivery: $($entry.RelativePath)"
        }
        [pscustomobject][ordered]@{
            RelativePath = [string]$entry.RelativePath
            Path = $actual.Path
            Bytes = $actual.Bytes
            SHA256 = $actual.SHA256
            LastWriteUtcTicks = $actual.LastWriteUtcTicks
        }
    }
)
$fixture = Get-FileFingerprint $ctx.SmokeFixturePath
if ($fixture.Bytes -ne [Int64]$ctx.SmokeFixtureBytes -or $fixture.SHA256 -cne $ctx.SmokeFixtureSHA256) {
    throw 'Reviewed managed smoke fixture differs from its accepted identity.'
}

Write-JsonFile (Join-Path $evidenceRoot 'inputs-before.json') $sourceBefore
Write-JsonFile (Join-Path $evidenceRoot 'protected-before.json') $protectedBefore
Write-JsonFile (Join-Path $evidenceRoot 'protected-evidence-before.json') $protectedEvidenceBefore
Write-JsonFile (Join-Path $evidenceRoot 'accepted-directories-before.json') $acceptedDirectories 16
Write-JsonFile (Join-Path $evidenceRoot 'accepted-gates-before.json') $acceptedGates
Write-JsonFile (Join-Path $evidenceRoot 'final-stage2-files.json') $stage2Before

$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    CandidateAbsent = $true
    CandidateDirectory = $ctx.candidateDirectory
    Version = $ctx.version
    SourceInputCount = $sourceBefore.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceBefore
    NativeFormatsFileCount = $formatsCount
    NoticePreamblePresent = $true
    ProtectedReleaseFileCount = $protectedBefore.Count
    ProtectedReleaseTotalBytes = Get-InventoryTotalBytes $protectedBefore
    ProtectedReleaseInventoryIdentity = Get-InventoryIdentity $protectedBefore
    ProtectedEvidenceFileCount = $protectedEvidenceBefore.Count
    ProtectedEvidenceTotalBytes = Get-InventoryTotalBytes $protectedEvidenceBefore
    ProtectedEvidenceInventoryIdentity = Get-InventoryIdentity $protectedEvidenceBefore
    ProtectedEvidenceAggregateSHA256 = $protectedAggregate.SHA256
    AcceptedEvidenceDirectoryCount = $acceptedDirectories.Count
    AcceptedGateFileCount = $acceptedGates.Count
    FinalStage2FileCount = $stage2Before.Count
    SmokeFixture = $fixture
    ExpectedGuiBuilds = $ctx.ExpectedGuiBuilds
    ExpectedCandidateFiles = $ctx.ExpectedCandidateFiles
}
Write-JsonFile (Join-Path $evidenceRoot 'baseline-result.json') $result
$result | ConvertTo-Json -Depth 6
