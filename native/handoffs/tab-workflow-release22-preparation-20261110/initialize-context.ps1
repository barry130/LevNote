param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('native-worker.20261110.22')]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [ValidateScript({
        [IO.Path]::IsPathFullyQualified($_) -and
        -not (Test-Path -LiteralPath ([IO.Path]::GetFullPath($_)))
    })]
    [string]$CandidateDirectory,

    [Parameter(Mandatory = $true)]
    [ValidateScript({
        [IO.Path]::IsPathFullyQualified($_) -and
        (Test-Path -LiteralPath ([IO.Path]::GetFullPath($_)) -PathType Leaf)
    })]
    [string[]]$TabGatePath,

    [Parameter(Mandatory = $true)]
    [ValidateScript({
        [IO.Path]::IsPathFullyQualified($_) -and
        (Test-Path -LiteralPath ([IO.Path]::GetFullPath($_)) -PathType Container)
    })]
    [string]$EvidenceDirectory,

    [switch]$ExecuteDelivery
)

<#
# TabSourceIdentity below is the complete inventory identity of the scoped
# product/build inputs. It is calculated from actual files and recorded as a
# hash string; it is never interpreted as a filesystem path.
#>
process {
    if ($Version -cne 'native-worker.20261110.22') {
        throw 'Invalid .22 version.'
    }
    $ErrorActionPreference = 'Stop'
    $evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
    . (Join-Path $evidenceRoot 'delivery-lib.ps1')
    $evidence = Assert-AbsoluteDirectory $EvidenceDirectory 'evidence directory'
    if (-not [string]::Equals($evidence, $evidenceRoot, [StringComparison]::OrdinalIgnoreCase)) {
        throw "EvidenceDirectory must be this harness root: $evidence"
    }
    if (Test-Path -LiteralPath (Join-Path $evidence 'context.json') -PathType Leaf) {
        throw 'Delivery context already exists.'
    }
    . (Join-Path $evidenceRoot 'source-scope.ps1')
    $repositoryRoot = [IO.Path]::GetFullPath('F:\AIproject\LevNote')
    $nativeRoot = Assert-AbsoluteDirectory (Join-Path $repositoryRoot 'native') 'native root'
    $sourcePaths = @(Get-DeliverySourceInputPaths $repositoryRoot $nativeRoot)
    $sourceInputs = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
    $tabSourceIdentity = Get-InventoryIdentity $sourceInputs

    $candidate = Assert-AbsentAbsolutePath $CandidateDirectory 'candidate directory'
    $expectedCandidate = [IO.Path]::GetFullPath(
        (Join-Path $nativeRoot 'bin/levnote-native-worker-20261110.22'))
    if (-not [string]::Equals($candidate, $expectedCandidate, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Candidate path must be the fresh .22 directory: $candidate"
    }

    $gates = @(
        foreach ($path in @($TabGatePath)) {
            $full = Assert-AbsolutePath $path 'accepted tab gate'
            if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
                throw "Accepted tab gate file is absent: $full"
            }
            $full
        }
    )
    if ($gates.Count -lt 1) { throw 'At least one accepted tab gate file is required.' }

$protectedRelease = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'bin/levnote-native-worker-20261110.21') `
    'protected .21 package'
$protectedEvidence = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'artifacts/acceptance/drawio-efficiency-release21-20261110') `
    'protected .21 direct evidence'
$protectedRetry = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'artifacts/acceptance/drawio-efficiency-release21-20261110-aggregate-retry') `
    'protected .21 retry sibling'
foreach ($protected in @(
        @{ Directory = $protectedRelease; Count = 25; Bytes = 26859809;
           Identity = '3126E7898073CEBBC02EAB907EEF541C8A3D5538F0B77A45D58B6CE9DBE72ED7' },
        @{ Directory = $protectedEvidence; Count = 202; Bytes = 91722328;
           Identity = '55B150C3A423BC3D213E7CF7282F1D87F9247AD35768236144D9217E7F4CE79B' },
        @{ Directory = $protectedRetry; Count = 4; Bytes = 1635;
           Identity = 'FA8CDD03BA1AC32257A97403BE861DC82E9EE3EF2443866D7860B6BC1EF6AB16' })) {
    $inventory = @(Get-DirectoryInventory $protected.Directory)
    if ($inventory.Count -ne [int]$protected.Count -or
        (Get-InventoryTotalBytes $inventory) -ne [Int64]$protected.Bytes -or
        (Get-InventoryIdentity $inventory) -cne $protected.Identity) {
        throw "Protected .21 inventory differs: $($protected.Directory)"
    }
}
$protectedReleaseManifest = Read-JsonFile (
    Join-Path $protectedRelease 'release-manifest.json')
$protectedReleaseHost = Get-FileFingerprint (
    Join-Path $protectedRelease 'levnote-native.exe')
$protectedReleaseManifestFile = Get-FileFingerprint (
    Join-Path $protectedRelease 'release-manifest.json')
if ($protectedReleaseManifest.version -cne 'native-worker.20261110.21' -or
    $protectedReleaseHost.SHA256 -cne '2C400D3730AEE64FC380CA0312B44BAAB1C0803AA70BA96525804BB4B436B329' -or
    $protectedReleaseManifestFile.SHA256 -cne 'C2556B0E372466B5F67FEBF7C9FBACBA0D44050B00F9892C64837ED16CE9A824') {
    throw 'Protected .21 version, host, or release manifest differs from acceptance.'
}
$exampleDirectory = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'examples/plugins') 'plugin examples'
$portableReadme = Get-FileFingerprint (
    Join-Path $nativeRoot 'portable/README.md')
$diagramNotesPath = Join-Path $exampleDirectory 'diagram-notes.json'
$diagramNotes = Get-FileFingerprint $diagramNotesPath
if ($diagramNotes.Bytes -ne 179 -or
    $diagramNotes.SHA256 -cne '317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07') {
    throw 'Reviewed diagram-notes JSON differs from the accepted identity.'
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
    ExpectedCandidateFiles = 25
    ExpectedEngineCount = 4
    ExpectedReleaseManifestSchema = 1
    ExpectedReleaseManifestEntries = 24
    PortableSourceReadme = [IO.Path]::GetFullPath(
        (Join-Path $nativeRoot 'portable/README.md'))
    PortableSourceReadmeFingerprint = $portableReadme
    TabGatePaths = @($gates)
    TabSourceIdentity = $tabSourceIdentity
    TabGateFingerprints = @($gates | ForEach-Object { Get-FileFingerprint $_ })
    ProtectedReleaseDirectory = $protectedRelease
    ProtectedReleaseFileCount = 25
    ProtectedReleaseTotalBytes = 26859809
    ProtectedReleaseInventoryIdentity = '3126E7898073CEBBC02EAB907EEF541C8A3D5538F0B77A45D58B6CE9DBE72ED7'
    ProtectedReleaseVersion = 'native-worker.20261110.21'
    ProtectedReleaseHostSHA256 = '2C400D3730AEE64FC380CA0312B44BAAB1C0803AA70BA96525804BB4B436B329'
    ProtectedReleaseManifestSHA256 = 'C2556B0E372466B5F67FEBF7C9FBACBA0D44050B00F9892C64837ED16CE9A824'
    ProtectedEvidenceDirectory = $protectedEvidence
    ProtectedEvidenceFileCount = 202
    ProtectedEvidenceTotalBytes = 91722328
    ProtectedEvidenceInventoryIdentity = '55B150C3A423BC3D213E7CF7282F1D87F9247AD35768236144D9217E7F4CE79B'
    ProtectedRetryDirectory = $protectedRetry
    ProtectedRetryFileCount = 4
    ProtectedRetryTotalBytes = 1635
    ProtectedRetryInventoryIdentity = 'FA8CDD03BA1AC32257A97403BE861DC82E9EE3EF2443866D7860B6BC1EF6AB16'
    ReviewedDiagramNotesPath = $diagramNotesPath
    ReviewedDiagramNotes = $diagramNotes
    SourceInputCount = $sourceInputs.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceInputs
    Scope = 'Fresh .22 package/evidence; accepted tab gate files, scoped product/build inputs, reviewed diagram-notes example, and immutable complete .21 package plus separate direct/retry evidence only.'
}
Write-JsonFile (Join-Path $evidenceRoot 'context.json') $context 16
Write-JsonFile (Join-Path $evidenceRoot 'commands.json') ([ordered]@{
    Schema = 1
    Schedule = @('baseline.ps1', 'run-package.ps1', 'run-host-smoke.ps1', 'aggregate.ps1')
    Wrapper = 'run-stage.ps1 -Name baseline|package|host-smoke|aggregate'
    PackageInvocationCount = 1
    TestExecutableCompileCount = 0
    WorkerRequestCount = 0
    HiddenHostStartCount = 1
}) 8
[ordered]@{
    Status = $(if ($ExecuteDelivery) { 'AUTHORIZED' } else { 'PREPARED-ONLY' })
    EvidenceDirectory = $evidenceRoot
    CandidateDirectory = $candidate
    Version = $Version
    TabGateCount = $gates.Count
    TabSourceIdentity = $tabSourceIdentity
    ProtectedRetryFileCount = 4
    ProtectedRetryTotalBytes = 1635
    ProtectedRetryInventoryIdentity = 'FA8CDD03BA1AC32257A97403BE861DC82E9EE3EF2443866D7860B6BC1EF6AB16'
} | ConvertTo-Json
}
