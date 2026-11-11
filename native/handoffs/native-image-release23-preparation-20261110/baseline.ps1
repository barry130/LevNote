$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not [string]::Equals(
        [IO.Path]::GetPathRoot([string]$ctx.candidateDirectory),
        'F:\',
        [StringComparison]::OrdinalIgnoreCase)) {
    throw "The fresh .23 candidate directory must be on F:, resolved to $($ctx.candidateDirectory)."
}
if (Test-Path -LiteralPath $ctx.candidateDirectory) {
    throw "Fresh .23 candidate directory already exists: $($ctx.candidateDirectory)"
}
if (Test-Path -LiteralPath (Join-Path $evidenceRoot 'baseline-result.json')) {
    throw 'Baseline already completed in this evidence directory.'
}

$sourcePaths = @(Get-DeliverySourceInputPaths $ctx.repositoryRoot $ctx.nativeRoot)
$sourceBefore = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
$sourceIdentity = Get-InventoryIdentity $sourceBefore
if ($sourceIdentity -cne [string]$ctx.SourceInputIdentity) {
    throw 'Scoped product/build source identity differs from initialization.'
}
$formatsPrefix = (Join-Path $ctx.nativeRoot 'formats').TrimEnd('\') + '\'
$formatsCount = @($sourceBefore | Where-Object {
    ([IO.Path]::GetFullPath([string]$_.Path)).StartsWith(
        $formatsPrefix, [StringComparison]::OrdinalIgnoreCase)
}).Count
$noticePath = [IO.Path]::GetFullPath(
    (Join-Path $ctx.nativeRoot 'portable/NOTICE-PREAMBLE.md'))
$bitmapPatchPath = [IO.Path]::GetFullPath(
    (Join-Path $ctx.nativeRoot 'dependency-patches/mygo-v0.8.3-bitmap-dispose.patch'))
$startupPatchPath = [IO.Path]::GetFullPath(
    (Join-Path $ctx.nativeRoot 'dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch'))
$patchManifestPath = [IO.Path]::GetFullPath(
    (Join-Path $ctx.nativeRoot 'dependency-patches/mygo-v0.8.3-patch-files.json'))
foreach ($required in @($noticePath, $bitmapPatchPath, $startupPatchPath, $patchManifestPath)) {
    if (@($sourceBefore | Where-Object {
        [IO.Path]::GetFullPath([string]$_.Path) -eq $required
    }).Count -ne 1) {
        throw "Scoped source/build inputs omit required file: $required"
    }
}
if ($formatsCount -eq 0) {
    throw 'Scoped build inputs must include native/formats.'
}

$portableReadme = Get-FileFingerprint $ctx.portableSourceReadme
$portableBefore = [pscustomobject]$ctx.PortableSourceReadmeFingerprint
if (-not (Test-FingerprintEqual $portableBefore $portableReadme)) {
    throw 'Portable README differs from its initialization identity.'
}

$acceptanceBefore = @(
    foreach ($path in @($ctx.RootSourceAcceptancePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$acceptanceExpected = @($ctx.RootSourceAcceptanceFingerprints)
if ($acceptanceBefore.Count -ne $acceptanceExpected.Count) {
    throw 'Root image source acceptance record set is incomplete.'
}
Assert-InventoriesEqual $acceptanceExpected $acceptanceBefore 'Root image source acceptance'
$architectureAcceptances = @(
    [pscustomobject]([ordered]@{
        Kind = 'composition'
        Path = [string]$ctx.CompositionAcceptancePath
        Expected = $ctx.CompositionAcceptanceFingerprint
    }),
    [pscustomobject]([ordered]@{
        Kind = 'app'
        Path = [string]$ctx.AppAcceptancePath
        Expected = $ctx.AppAcceptanceFingerprint
    })
)
foreach ($entry in $architectureAcceptances) {
    $actual = Get-FileFingerprint $entry.Path
    if (-not (Test-FingerprintEqual ([pscustomobject]$entry.Expected) $actual)) {
        throw "$($entry.Kind) acceptance differs from initialization."
    }
}

$frozenFacts = Get-FrozenBitmapFacts $ctx.nativeRoot $ctx.FrozenBitmapFilesPath
$patchFacts = Get-PinnedPatchFacts $ctx.nativeRoot $frozenFacts
if ($frozenFacts.FileCount -ne [int]$ctx.FrozenBitmapFileCount -or
    $frozenFacts.ChangedOrAdded -ne [int]$ctx.FrozenBitmapChangedOrAdded -or
    $frozenFacts.UnchangedReferences -ne [int]$ctx.FrozenBitmapUnchangedReferences) {
    throw 'Frozen bitmap lifecycle counts differ from context initialization.'
}
if (@($patchFacts.Entries).Count -ne [int]$ctx.PatchManifestEntryCount -or
    -not (Test-FingerprintEqual ([pscustomobject]$ctx.PatchManifest) $patchFacts.Manifest) -or
    -not (Test-FingerprintEqual ([pscustomobject]$ctx.BitmapPatch) $patchFacts.BitmapPatch) -or
    -not (Test-FingerprintEqual ([pscustomobject]$ctx.StartupPatch) $patchFacts.StartupPatch)) {
    throw 'Pinned patch manifest or patch files differ from context initialization.'
}

$protectedBefore = @(Get-DirectoryInventory $ctx.protectedReleaseDirectory)
$protectedEvidenceBefore = @(Get-DirectoryInventory $ctx.protectedEvidenceDirectory)
$protectedCaptureBefore = @(Get-DirectoryInventory $ctx.protectedCaptureDirectory)
foreach ($pair in @(
        @{ Inventory = $protectedBefore; Count = [int]$ctx.ProtectedReleaseFileCount;
           Bytes = [Int64]$ctx.ProtectedReleaseTotalBytes;
           Identity = [string]$ctx.ProtectedReleaseInventoryIdentity
           Label = 'Protected .22 package' },
        @{ Inventory = $protectedEvidenceBefore; Count = [int]$ctx.ProtectedEvidenceFileCount;
           Bytes = [Int64]$ctx.ProtectedEvidenceTotalBytes;
           Identity = [string]$ctx.ProtectedEvidenceInventoryIdentity
           Label = 'Protected .22 direct evidence' },
        @{ Inventory = $protectedCaptureBefore; Count = [int]$ctx.ProtectedCaptureFileCount;
           Bytes = [Int64]$ctx.ProtectedCaptureTotalBytes;
           Identity = [string]$ctx.ProtectedCaptureInventoryIdentity
           Label = 'Protected .22 aggregate capture' })) {
    if ($pair.Inventory.Count -ne $pair.Count -or
        (Get-InventoryTotalBytes $pair.Inventory) -ne $pair.Bytes -or
        (Get-InventoryIdentity $pair.Inventory) -cne $pair.Identity) {
        throw "$($pair.Label) does not match its accepted complete inventory."
    }
}
$closedBefore = Get-FileFingerprint (
    Join-Path $ctx.nativeRoot 'artifacts/acceptance/tab-workflow-release22-20261110-closed-inventory.json')
if (-not (Test-FingerprintEqual ([pscustomobject]$ctx.ProtectedClosedInventory) $closedBefore)) {
    throw 'Protected .22 closed inventory differs from initialization.'
}

$diagramNotes = Get-FileFingerprint $ctx.ReviewedDiagramNotesPath
$mixedText = Get-FileFingerprint $ctx.MixedTextFixturePath
$imageFixture = Get-FileFingerprint $ctx.ImageFixturePath
if (-not (Test-FingerprintEqual ([pscustomobject]$ctx.ReviewedDiagramNotes) $diagramNotes) -or
    -not (Test-FingerprintEqual ([pscustomobject]$ctx.MixedTextFixture) $mixedText) -or
    -not (Test-FingerprintEqual ([pscustomobject]$ctx.ImageFixture) $imageFixture)) {
    throw 'Reviewed plugin or mixed text/PNG fixtures changed after initialization.'
}

$readmeText = [IO.File]::ReadAllText(
    $ctx.portableSourceReadme, [Text.Encoding]::UTF8)
$imageReadmeContract = Get-ImageReadmeContract
foreach ($needle in @($imageReadmeContract.Required)) {
    Assert-ExpectedText $readmeText $needle 'Portable README'
}
foreach ($needle in @($imageReadmeContract.Forbidden)) {
    Assert-ForbiddenText $readmeText $needle 'Portable README'
}
$jsonRoutes = @(
    [regex]::Matches($readmeText, '(?m)^\{"schema":3,"id":"[^\r\n]+\}$') |
        ForEach-Object { [string](($_.Value | ConvertFrom-Json).nativeRoute) }
)
if (([regex]::Matches($readmeText, '\{\{VERSION\}\}')).Count -ne 1 -or
    $jsonRoutes.Count -ne 3 -or
    (@($jsonRoutes | Sort-Object) -join ',') -cne
        (@('builtin.drawio', 'builtin.markdown', 'builtin.mindmap') -join ',')) {
    throw 'Portable README marker or accepted schema3 example set changed.'
}

Write-JsonFile (Join-Path $evidenceRoot 'inputs-before.json') $sourceBefore
Write-JsonFile (Join-Path $evidenceRoot 'source-gates-before.json') $acceptanceBefore
Write-JsonFile (Join-Path $evidenceRoot 'architecture-acceptances-before.json') $architectureAcceptances
Write-JsonFile (Join-Path $evidenceRoot 'protected-before.json') $protectedBefore
Write-JsonFile (Join-Path $evidenceRoot 'protected-evidence-before.json') $protectedEvidenceBefore
Write-JsonFile (Join-Path $evidenceRoot 'protected-capture-before.json') $protectedCaptureBefore
Write-JsonFile (Join-Path $evidenceRoot 'frozen-bitmap-before.json') $frozenFacts
Write-JsonFile (Join-Path $evidenceRoot 'pinned-patches-before.json') $patchFacts

$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    CandidateAbsent = $true
    CandidateDirectory = $ctx.candidateDirectory
    Version = $ctx.version
    SourceInputCount = $sourceBefore.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceBefore
    SourceInputIdentity = $sourceIdentity
    NativeFormatsFileCount = $formatsCount
    NoticePreamblePresent = $true
    PortableReadme = $portableReadme
    PortableReadmeMarkerCount = 1
    PortableReadmeInlineSchema3JSONCount = $jsonRoutes.Count
    PortableReadmeRoutes = $jsonRoutes
    RootSourceAcceptanceFileCount = $acceptanceBefore.Count
    CompositionAcceptance = Get-FileFingerprint $ctx.CompositionAcceptancePath
    AppAcceptance = Get-FileFingerprint $ctx.AppAcceptancePath
    FrozenBitmapFileCount = $frozenFacts.FileCount
    FrozenBitmapIdentity = $frozenFacts.FrozenFiles.SHA256
    PatchManifestEntryCount = @($patchFacts.Entries).Count
    PatchManifest = $patchFacts.Manifest
    BitmapPatch = $patchFacts.BitmapPatch
    StartupPatch = $patchFacts.StartupPatch
    GuardedFilesMatchFrozenList = $patchFacts.GuardedFilesMatchFrozenList
    ProtectedReleaseFileCount = $protectedBefore.Count
    ProtectedReleaseTotalBytes = Get-InventoryTotalBytes $protectedBefore
    ProtectedReleaseInventoryIdentity = Get-InventoryIdentity $protectedBefore
    ProtectedEvidenceFileCount = $protectedEvidenceBefore.Count
    ProtectedEvidenceTotalBytes = Get-InventoryTotalBytes $protectedEvidenceBefore
    ProtectedEvidenceInventoryIdentity = Get-InventoryIdentity $protectedEvidenceBefore
    ProtectedCaptureFileCount = $protectedCaptureBefore.Count
    ProtectedCaptureTotalBytes = Get-InventoryTotalBytes $protectedCaptureBefore
    ProtectedCaptureInventoryIdentity = Get-InventoryIdentity $protectedCaptureBefore
    ProtectedClosedInventory = $closedBefore
    ReviewedDiagramNotes = $diagramNotes
    MixedTextFixture = $mixedText
    ImageFixture = $imageFixture
    ExpectedGuiBuilds = $ctx.ExpectedGuiBuilds
    ExpectedEngineCount = $ctx.ExpectedEngineCount
    ExpectedHiddenHostStarts = $ctx.ExpectedHiddenHostStarts
}
Write-JsonFile (Join-Path $evidenceRoot 'baseline-result.json') $result 16
$result | ConvertTo-Json -Depth 8
