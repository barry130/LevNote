$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not [string]::Equals(
        [IO.Path]::GetPathRoot($ctx.candidateDirectory),
        'F:\',
        [StringComparison]::OrdinalIgnoreCase)) {
    throw "The fresh .21 candidate directory must be on F:, resolved to $($ctx.candidateDirectory)."
}
if (Test-Path -LiteralPath $ctx.candidateDirectory) {
    throw "Fresh .21 candidate directory already exists: $($ctx.candidateDirectory)"
}
if (Test-Path -LiteralPath (Join-Path $evidenceRoot 'baseline-result.json')) {
    throw 'Baseline already completed in this evidence directory.'
}

$sourcePaths = @(Get-DeliverySourceInputPaths $ctx.repositoryRoot $ctx.nativeRoot)
$sourceBefore = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
$formatsPrefix = (Join-Path $ctx.nativeRoot 'formats').TrimEnd('\') + '\'
$formatsCount = @($sourceBefore | Where-Object {
    ([IO.Path]::GetFullPath([string]$_.Path)).StartsWith(
        $formatsPrefix, [StringComparison]::OrdinalIgnoreCase)
}).Count
$noticePath = [IO.Path]::GetFullPath(
    (Join-Path $ctx.nativeRoot 'portable/NOTICE-PREAMBLE.md'))
if ($formatsCount -eq 0 -or @($sourceBefore | Where-Object {
    [IO.Path]::GetFullPath([string]$_.Path) -eq $noticePath
}).Count -ne 1) {
    throw 'Scoped build inputs must include native/formats and portable/NOTICE-PREAMBLE.md.'
}

$portableReadme = Get-FileFingerprint $ctx.portableSourceReadme
if ($portableReadme.Bytes -ne [Int64]$ctx.PortableSourceReadmeBytes -or
    $portableReadme.SHA256 -cne [string]$ctx.PortableSourceReadmeSHA256) {
    throw 'Portable README differs from the accepted .20 source.'
}
$readmeText = [IO.File]::ReadAllText(
    $ctx.portableSourceReadme, [Text.Encoding]::UTF8)
$jsonRoutes = @(
    foreach ($match in [regex]::Matches(
            $readmeText, '(?m)^\{"schema":3,"id":"[^\r\n]+\}$')) {
        try {
            ([string]($match.Value | ConvertFrom-Json).nativeRoute)
        } catch {
            throw "Portable README contains invalid inline schema3 JSON: $($match.Value)"
        }
    }
)
if (([regex]::Matches($readmeText, '\{\{VERSION\}\}')).Count -ne 1 -or
    $jsonRoutes.Count -ne 3 -or
    (@($jsonRoutes | Sort-Object) -join ',') -cne
        (@('builtin.drawio', 'builtin.markdown', 'builtin.mindmap') -join ',') -or
    -not $readmeText.Contains('本包')) {
    throw 'Portable README must retain one version marker and the three accepted schema3 routes.'
}

$protectedBefore = @(Get-DirectoryInventory $ctx.protectedReleaseDirectory)
if ($protectedBefore.Count -ne [int]$ctx.protectedReleaseFileCount -or
    (Get-InventoryTotalBytes $protectedBefore) -ne [Int64]$ctx.protectedReleaseTotalBytes -or
    (Get-InventoryIdentity $protectedBefore) -cne [string]$ctx.protectedReleaseInventoryIdentity) {
    throw 'Protected .20 package does not match its accepted complete inventory.'
}
$protectedManifest = Read-JsonFile (
    Join-Path $ctx.protectedReleaseDirectory 'release-manifest.json')
$protectedHost = Get-FileFingerprint (
    Join-Path $ctx.protectedReleaseDirectory 'levnote-native.exe')
$protectedManifestFile = Get-FileFingerprint (
    Join-Path $ctx.protectedReleaseDirectory 'release-manifest.json')
if ($protectedManifest.version -cne $ctx.protectedReleaseVersion -or
    $protectedHost.SHA256 -cne $ctx.protectedReleaseHostSHA256 -or
    $protectedManifestFile.SHA256 -cne $ctx.protectedReleaseManifestSHA256) {
    throw 'Protected .20 version, host or release manifest differs from acceptance.'
}

$protectedEvidenceBefore = @(Get-DirectoryInventory $ctx.protectedEvidenceDirectory)
if ($protectedEvidenceBefore.Count -ne [int]$ctx.protectedEvidenceFileCount -or
    (Get-InventoryTotalBytes $protectedEvidenceBefore) -ne [Int64]$ctx.protectedEvidenceTotalBytes -or
    (Get-InventoryIdentity $protectedEvidenceBefore) -cne [string]$ctx.protectedEvidenceInventoryIdentity) {
    throw 'Protected .20 direct evidence does not match its accepted complete inventory.'
}
$protectedAggregate = Get-FileFingerprint (
    Join-Path $ctx.protectedEvidenceDirectory 'aggregate.json')
if ($protectedAggregate.SHA256 -cne [string]$ctx.protectedEvidenceAggregateSHA256) {
    throw 'Protected .20 aggregate hash differs from acceptance.'
}

$sourceGatesBefore = @(
    foreach ($path in @($ctx.SourceGatePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
if ($sourceGatesBefore.Count -ne @($ctx.SourceGateFingerprints).Count -or
    @($sourceGatesBefore | Where-Object {
        (Split-Path $_.Path -Leaf) -match '(?i)warning|structural'
    }).Count -eq 0 -or
    @($sourceGatesBefore | Where-Object {
        (Split-Path $_.Path -Leaf) -match '(?i)retention|efficiency'
    }).Count -eq 0) {
    throw 'Accepted warning and retention gate records are incomplete.'
}
for ($index = 0; $index -lt $sourceGatesBefore.Count; $index++) {
    $expected = $ctx.SourceGateFingerprints[$index]
    if ($sourceGatesBefore[$index].Path -cne [string]$expected.Path -or
        $sourceGatesBefore[$index].Bytes -ne [Int64]$expected.Bytes -or
        $sourceGatesBefore[$index].SHA256 -cne [string]$expected.SHA256 -or
        $sourceGatesBefore[$index].LastWriteUtcTicks -ne [Int64]$expected.LastWriteUtcTicks) {
        throw "Accepted gate file changed after initialization: $($sourceGatesBefore[$index].Path)"
    }
}

$benchmarkFixturesBefore = @(
    foreach ($path in @($ctx.BenchmarkFixturePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
if ($benchmarkFixturesBefore.Count -ne @($ctx.BenchmarkFixtureFingerprints).Count) {
    throw 'Accepted benchmark fixture set changed after initialization.'
}
for ($index = 0; $index -lt $benchmarkFixturesBefore.Count; $index++) {
    $expected = $ctx.BenchmarkFixtureFingerprints[$index]
    if ($benchmarkFixturesBefore[$index].Path -cne [string]$expected.Path -or
        $benchmarkFixturesBefore[$index].Bytes -ne [Int64]$expected.Bytes -or
        $benchmarkFixturesBefore[$index].SHA256 -cne [string]$expected.SHA256 -or
        $benchmarkFixturesBefore[$index].LastWriteUtcTicks -ne [Int64]$expected.LastWriteUtcTicks) {
        throw "Accepted benchmark fixture changed after initialization: $($benchmarkFixturesBefore[$index].Path)"
    }
}

$fixture = Get-FileFingerprint $ctx.SmokeFixturePath
if ($fixture.Bytes -ne [Int64]$ctx.SmokeFixture.Bytes -or
    $fixture.SHA256 -cne [string]$ctx.SmokeFixture.SHA256 -or
    $fixture.LastWriteUtcTicks -ne [Int64]$ctx.SmokeFixture.LastWriteUtcTicks) {
    throw 'Reviewed managed smoke tester source differs from its initialization identity.'
}
$diagramNotes = Get-FileFingerprint $ctx.ReviewedDiagramNotesPath
if ($diagramNotes.Bytes -ne [Int64]$ctx.ReviewedDiagramNotesBytes -or
    $diagramNotes.SHA256 -cne [string]$ctx.ReviewedDiagramNotesSHA256) {
    throw 'Reviewed diagram-notes manifest differs from the accepted example.'
}

Write-JsonFile (Join-Path $evidenceRoot 'inputs-before.json') $sourceBefore
Write-JsonFile (Join-Path $evidenceRoot 'protected-before.json') $protectedBefore
Write-JsonFile (Join-Path $evidenceRoot 'protected-evidence-before.json') $protectedEvidenceBefore
Write-JsonFile (Join-Path $evidenceRoot 'source-gates-before.json') $sourceGatesBefore
Write-JsonFile (Join-Path $evidenceRoot 'benchmark-fixtures-before.json') $benchmarkFixturesBefore

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
    PortableReadme = $portableReadme
    PortableReadmeMarkerCount = 1
    PortableReadmeInlineSchema3JSONCount = $jsonRoutes.Count
    PortableReadmeRoutes = $jsonRoutes
    ProtectedReleaseFileCount = $protectedBefore.Count
    ProtectedReleaseTotalBytes = Get-InventoryTotalBytes $protectedBefore
    ProtectedReleaseInventoryIdentity = Get-InventoryIdentity $protectedBefore
    ProtectedEvidenceFileCount = $protectedEvidenceBefore.Count
    ProtectedEvidenceTotalBytes = Get-InventoryTotalBytes $protectedEvidenceBefore
    ProtectedEvidenceInventoryIdentity = Get-InventoryIdentity $protectedEvidenceBefore
    ProtectedEvidenceAggregateSHA256 = $protectedAggregate.SHA256
    SourceGateFileCount = $sourceGatesBefore.Count
    BenchmarkFixtureCount = $benchmarkFixturesBefore.Count
    SmokeFixture = $fixture
    ReviewedDiagramNotes = $diagramNotes
    ExpectedGuiBuilds = $ctx.ExpectedGuiBuilds
    ExpectedCandidateFiles = $ctx.ExpectedCandidateFiles
}
Write-JsonFile (Join-Path $evidenceRoot 'baseline-result.json') $result
$result | ConvertTo-Json -Depth 8
