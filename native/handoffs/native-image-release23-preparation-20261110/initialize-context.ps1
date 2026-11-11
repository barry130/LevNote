param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('native-worker.20261110.23')]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [ValidateScript({
        [IO.Path]::IsPathFullyQualified($_) -and
        -not (Test-Path -LiteralPath ([IO.Path]::GetFullPath($_)))
    })]
    [string]$CandidateDirectory,

    [Parameter(Mandatory = $true)]
    [string[]]$RootSourceAcceptancePath,

    [Parameter(Mandatory = $true)]
    [string]$CompositionAcceptancePath,

    [Parameter(Mandatory = $true)]
    [string]$AppAcceptancePath,

    [Parameter(Mandatory = $true)]
    [string]$MixedTextFixturePath,

    [Parameter(Mandatory = $true)]
    [string]$ImageFixturePath,

    [Parameter(Mandatory = $true)]
    [ValidateScript({
        [IO.Path]::IsPathFullyQualified($_) -and
        (Test-Path -LiteralPath ([IO.Path]::GetFullPath($_)) -PathType Container)
    })]
    [string]$EvidenceDirectory,

    [switch]$ExecuteDelivery
)

$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')

if ($Version -cne 'native-worker.20261110.23') {
    throw 'Invalid .23 version.'
}
$evidence = Assert-AbsoluteDirectory $EvidenceDirectory 'evidence directory'
if (-not [string]::Equals($evidence, $evidenceRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw "EvidenceDirectory must be this harness root: $evidence"
}
if (Test-Path -LiteralPath (Join-Path $evidence 'context.json') -PathType Leaf) {
    throw 'Delivery context already exists.'
}

$repositoryRoot = [IO.Path]::GetFullPath('F:\AIproject\LevNote')
$nativeRoot = Assert-AbsoluteDirectory (Join-Path $repositoryRoot 'native') 'native root'
$candidate = Assert-AbsentAbsolutePath $CandidateDirectory 'candidate directory'
$expectedCandidate = [IO.Path]::GetFullPath(
    (Join-Path $nativeRoot 'bin/levnote-native-worker-20261110.23'))
if (-not [string]::Equals($candidate, $expectedCandidate, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Candidate path must be the fresh .23 directory: $candidate"
}
if (-not [string]::Equals(
        [IO.Path]::GetPathRoot($candidate),
        'F:\',
        [StringComparison]::OrdinalIgnoreCase)) {
    throw "The fresh .23 candidate must be on F:, resolved to $candidate."
}

$requiredAcceptanceNames = @(
    'raster-admission-root-acceptance.md',
    'raster-decode-root-acceptance.md',
    'binary-core-root-acceptance.md',
    'binary-copy-receipt-root-acceptance.md',
    'binary-session-root-acceptance.md',
    'image-bitmap-lifecycle-root-acceptance.md',
    'native-image-view-root-acceptance.md',
    'native-image-view-narrow-toolbar-root-acceptance.md'
)
$sourceGates = @(
    foreach ($path in @($RootSourceAcceptancePath)) {
        $full = Assert-AbsolutePath $path 'accepted image source gate'
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
            throw "Accepted image source gate is absent: $full"
        }
        $full
    }
)
if ($sourceGates.Count -ne $requiredAcceptanceNames.Count) {
    throw "Exactly eight accepted image source gates are required; found $($sourceGates.Count)."
}
$actualNames = @($sourceGates | ForEach-Object { [IO.Path]::GetFileName($_) })
foreach ($name in $requiredAcceptanceNames) {
    if (@($actualNames | Where-Object { $_ -ceq $name }).Count -ne 1) {
        throw "Required accepted image source gate is missing or duplicated: $name"
    }
}

# Composition and final application integration are separate accepted gates.
# Initialization binds both explicit root acceptances together with the eight
# component/source gates so the eventual package cannot be authorized from
# frozen component source alone.
$compositionAcceptance = Assert-AbsolutePath $CompositionAcceptancePath 'composition acceptance'
$appAcceptance = Assert-AbsolutePath $AppAcceptancePath 'app integration acceptance'
foreach ($path in @($compositionAcceptance, $appAcceptance)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required composition or final-app integration acceptance is absent: $path"
    }
    if ([IO.Path]::GetFileName($path) -notmatch 'root-acceptance\.md$') {
        throw "Integration acceptance must be an explicit root-acceptance file: $path"
    }
}

$sourcePaths = @(Get-DeliverySourceInputPaths $repositoryRoot $nativeRoot)
$sourceInputs = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
$sourceIdentity = Get-InventoryIdentity $sourceInputs
$frozenFilesPath = [IO.Path]::GetFullPath(
    (Join-Path $nativeRoot 'handoffs/image-bitmap-lifecycle-frozen-files.json'))
$frozenFacts = Get-FrozenBitmapFacts $nativeRoot $frozenFilesPath
$patchFacts = Get-PinnedPatchFacts $nativeRoot $frozenFacts

$protectedRelease = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'bin/levnote-native-worker-20261110.22') `
    'protected .22 package'
$protectedEvidence = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'artifacts/acceptance/tab-workflow-release22-20261110') `
    'protected .22 direct evidence'
$protectedCapture = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'artifacts/acceptance/tab-workflow-release22-20261110-aggregate-capture') `
    'protected .22 aggregate capture'
$protectedClosed = Get-FileFingerprint `
    (Join-Path $nativeRoot 'artifacts/acceptance/tab-workflow-release22-20261110-closed-inventory.json')
foreach ($protected in @(
        @{ Directory = $protectedRelease; Count = 25; Bytes = [Int64]26863393;
           Identity = 'BA43209FF7F6CFBEDD8287DBEA7B642008AFA839D133FECD8CC5C9D065FD9CF6' },
        @{ Directory = $protectedEvidence; Count = 134; Bytes = [Int64]54340909;
           Identity = 'B862D886E30F00D4364972DC8521AA75EB2C266DF81B1250DF1F42A2D1B1CC5C' },
        @{ Directory = $protectedCapture; Count = 6; Bytes = [Int64]2684;
           Identity = 'E6D61DCF9B02D9D32F3DC4957DA4747506E0A7E942DD62C55C91B9107011DFE3' })) {
    $inventory = @(Get-DirectoryInventory $protected.Directory)
    if ($inventory.Count -ne [int]$protected.Count -or
        (Get-InventoryTotalBytes $inventory) -ne [Int64]$protected.Bytes -or
        (Get-InventoryIdentity $inventory) -cne $protected.Identity) {
        throw "Protected .22 inventory differs: $($protected.Directory)"
    }
}
if ($protectedClosed.Bytes -ne 57245 -or
    $protectedClosed.SHA256 -cne
        'E61FB97AA24C5DAD6673F365D32EB77398C0D98EE32FE456A13F9B45A75373E2') {
    throw 'Protected .22 closed inventory identity differs.'
}
$protectedReleaseManifest = Read-JsonFile (
    Join-Path $protectedRelease 'release-manifest.json')
$protectedReleaseHost = Get-FileFingerprint (
    Join-Path $protectedRelease 'levnote-native.exe')
$protectedReleaseManifestFile = Get-FileFingerprint (
    Join-Path $protectedRelease 'release-manifest.json')
if ($protectedReleaseManifest.version -cne 'native-worker.20261110.22' -or
    $protectedReleaseHost.Bytes -ne 14101504 -or
    $protectedReleaseHost.SHA256 -cne
        '085E65F7BA1386A01E58CFCB26F0C069AACBFFD47AA063974CCDF0CC99C0C19F' -or
    $protectedReleaseManifestFile.Bytes -ne 4490 -or
    $protectedReleaseManifestFile.SHA256 -cne
        '4F66B7DC96E7213DF728E44A44B3DA77AE2C293BC5C08E10315156DB4793671F') {
    throw 'Protected .22 version, host, or release manifest differs from acceptance.'
}

$exampleDirectory = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'examples/plugins') 'plugin examples'
$diagramNotesPath = Join-Path $exampleDirectory 'diagram-notes.json'
$diagramNotes = Get-FileFingerprint $diagramNotesPath
if ($diagramNotes.Bytes -ne 179 -or
    $diagramNotes.SHA256 -cne
        '317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07') {
    throw 'Reviewed diagram-notes JSON differs from the accepted identity.'
}

$mixedText = Get-FileFingerprint (Assert-AbsolutePath $MixedTextFixturePath 'mixed text fixture')
$mixedBytes = [IO.File]::ReadAllBytes($mixedText.Path)
if ($mixedText.Bytes -ne 893 -or
    $mixedText.SHA256 -cne
        '7A166D51274D6868463F6D2CF671D2B8680D05047A24D2A85156E28843E4A1FB' -or
    -not ($mixedBytes.Length -ge 3 -and
        $mixedBytes[0] -eq 0xEF -and $mixedBytes[1] -eq 0xBB -and $mixedBytes[2] -eq 0xBF)) {
    throw 'Mixed text fixture must be the accepted 893-byte UTF-8/BOM file.'
}
$mixedTextValue = [Text.Encoding]::UTF8.GetString($mixedBytes)
if (-not $mixedTextValue.Contains("`r`n") -or
    [regex]::IsMatch($mixedTextValue, "(?<!`r)`n")) {
    throw 'Mixed text fixture must use CRLF exclusively.'
}

$imageFixture = Get-FileFingerprint (Assert-AbsolutePath $ImageFixturePath 'PNG fixture')
if ($imageFixture.Bytes -ne 17635 -or
    $imageFixture.SHA256 -cne
        'C00AA0141252F585691226D0AC4092CE67C2425B5175FBD0536BA78E633BA92F') {
    throw 'PNG fixture must be the accepted corrected light imageview capture.'
}

$portableReadmePath = Join-Path $nativeRoot 'portable/README.md'
$portableReadme = Get-FileFingerprint $portableReadmePath
$readmeText = [IO.File]::ReadAllText($portableReadmePath, [Text.Encoding]::UTF8)
$imageReadmeContract = Get-ImageReadmeContract
foreach ($needle in @($imageReadmeContract.Required)) {
    Assert-ExpectedText $readmeText $needle 'Portable README'
}
foreach ($needle in @($imageReadmeContract.Forbidden)) {
    Assert-ForbiddenText $readmeText $needle 'Portable README'
}
$jsonMatches = [regex]::Matches($readmeText, '(?m)^\{"schema":3,"id":"[^\r\n]+\}$')
$jsonRoutes = @(
    foreach ($match in $jsonMatches) {
        [string](($match.Value | ConvertFrom-Json).nativeRoute)
    }
)
if (([regex]::Matches($readmeText, '\{\{VERSION\}\}')).Count -ne 1 -or
    $jsonMatches.Count -ne 3 -or
    (@($jsonRoutes | Sort-Object) -join ',') -cne
        (@('builtin.drawio', 'builtin.markdown', 'builtin.mindmap') -join ',')) {
    throw 'Portable README must retain one version marker and the three accepted schema3 routes.'
}

$context = [ordered]@{
    Schema = 1
    PreparedAtUtc = [DateTime]::UtcNow.ToString('o')
    ExecutionAuthorized = [bool]$ExecuteDelivery
    RepositoryRoot = $repositoryRoot
    NativeRoot = $nativeRoot
    EvidenceDirectory = $evidence
    CandidateDirectory = $candidate
    Version = $Version
    PackageScript = [IO.Path]::GetFullPath(
        (Join-Path $nativeRoot 'scripts/package-markdown-worker.ps1'))
    ExpectedGuiBuilds = 5
    ExpectedEngineCount = 4
    ExpectedHiddenHostStarts = 2
    ExpectedReleaseManifestSchema = 1
    PortableSourceReadme = $portableReadmePath
    PortableSourceReadmeFingerprint = $portableReadme
    RootSourceAcceptancePaths = @($sourceGates)
    RootSourceAcceptanceFingerprints = @($sourceGates | ForEach-Object { Get-FileFingerprint $_ })
    CompositionAcceptancePath = $compositionAcceptance
    CompositionAcceptanceFingerprint = Get-FileFingerprint $compositionAcceptance
    AppAcceptancePath = $appAcceptance
    AppAcceptanceFingerprint = Get-FileFingerprint $appAcceptance
    FrozenBitmapFilesPath = $frozenFilesPath
    FrozenBitmapFiles = $frozenFacts.FrozenFiles
    FrozenBitmapFileCount = $frozenFacts.FileCount
    FrozenBitmapChangedOrAdded = $frozenFacts.ChangedOrAdded
    FrozenBitmapUnchangedReferences = $frozenFacts.UnchangedReferences
    FrozenBitmapEntries = $frozenFacts.Entries
    PatchManifestPath = $patchFacts.ManifestPath
    PatchManifest = $patchFacts.Manifest
    PatchManifestEntryCount = @($patchFacts.Entries).Count
    BitmapPatch = $patchFacts.BitmapPatch
    StartupPatch = $patchFacts.StartupPatch
    ProtectedReleaseDirectory = $protectedRelease
    ProtectedReleaseFileCount = 25
    ProtectedReleaseTotalBytes = [Int64]26863393
    ProtectedReleaseInventoryIdentity = 'BA43209FF7F6CFBEDD8287DBEA7B642008AFA839D133FECD8CC5C9D065FD9CF6'
    ProtectedReleaseVersion = 'native-worker.20261110.22'
    ProtectedReleaseHostSHA256 = '085E65F7BA1386A01E58CFCB26F0C069AACBFFD47AA063974CCDF0CC99C0C19F'
    ProtectedReleaseManifestSHA256 = '4F66B7DC96E7213DF728E44A44B3DA77AE2C293BC5C08E10315156DB4793671F'
    ProtectedEvidenceDirectory = $protectedEvidence
    ProtectedEvidenceFileCount = 134
    ProtectedEvidenceTotalBytes = [Int64]54340909
    ProtectedEvidenceInventoryIdentity = 'B862D886E30F00D4364972DC8521AA75EB2C266DF81B1250DF1F42A2D1B1CC5C'
    ProtectedCaptureDirectory = $protectedCapture
    ProtectedCaptureFileCount = 6
    ProtectedCaptureTotalBytes = [Int64]2684
    ProtectedCaptureInventoryIdentity = 'E6D61DCF9B02D9D32F3DC4957DA4747506E0A7E942DD62C55C91B9107011DFE3'
    ProtectedClosedInventory = $protectedClosed
    ReviewedDiagramNotesPath = $diagramNotesPath
    ReviewedDiagramNotes = $diagramNotes
    MixedTextFixturePath = $mixedText.Path
    MixedTextFixture = $mixedText
    ImageFixturePath = $imageFixture.Path
    ImageFixture = $imageFixture
    SourceInputCount = $sourceInputs.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceInputs
    SourceInputIdentity = $sourceIdentity
    Scope = 'Fresh .23 image package/evidence; eight accepted image source gates including binary-copy receipts and narrow-toolbar layout plus composition/final-app root acceptances; scoped app/core/formats/modules/cmd/ui, portable build inputs, both MyGo patches and guarded frozen files; reviewed plugin/text/PNG fixtures; immutable complete .22 package/direct evidence/capture/closed inventory.'
}
Write-JsonFile (Join-Path $evidenceRoot 'context.json') $context 18
Write-JsonFile (Join-Path $evidenceRoot 'commands.json') ([ordered]@{
    Schema = 1
    Schedule = @(
        'baseline.ps1',
        'run-package.ps1',
        'run-host-smoke.ps1',
        'aggregate.ps1'
    )
    Wrapper = 'run-stage.ps1 -Name baseline|package|host-smoke'
    PackageInvocationCount = 1
    GuiBuildCount = 5
    HiddenHostStartCount = 2
    HostRunOrder = @('mixed-cli-open', 'path-only-session-restore')
    WorkerRequestCount = 0
    TestExecutableCompileCount = 0
    AggregateFirstAttempt = 'run-stage.ps1 -Name aggregate -AggregateAttempt 1'
    AggregateRetryLimit = 2
    AggregateRetry = 'run-stage.ps1 -Name aggregate -AggregateAttempt 2 (only after preserved attempt 1 failure; no baseline/package/host rerun)'
})
[ordered]@{
    Status = $(if ($ExecuteDelivery) { 'AUTHORIZED' } else { 'PREPARED-ONLY' })
    EvidenceDirectory = $evidenceRoot
    CandidateDirectory = $candidate
    Version = $Version
    RootSourceAcceptanceCount = $sourceGates.Count
    CompositionAcceptance = $compositionAcceptance
    AppAcceptance = $appAcceptance
    FrozenBitmapFileCount = $frozenFacts.FileCount
    PatchManifestEntryCount = @($patchFacts.Entries).Count
    SourceInputCount = $sourceInputs.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceInputs
    SourceInputIdentity = $sourceIdentity
} | ConvertTo-Json -Depth 6
