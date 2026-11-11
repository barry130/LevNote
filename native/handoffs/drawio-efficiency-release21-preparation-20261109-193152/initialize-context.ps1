param(
    [Parameter(Mandatory = $true)][string]$CandidateDirectory,
    [Parameter(Mandatory = $true)][string]$EvidenceDirectory,
    [Parameter(Mandatory = $true)][string[]]$SourceGatePaths,
    [Parameter(Mandatory = $true)][string[]]$BenchmarkFixturePaths,
    [switch]$ExecuteDelivery
)

$ErrorActionPreference = 'Stop'
$preparationRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$repositoryRoot = [IO.Path]::GetFullPath((Join-Path $preparationRoot '..\..\..'))
. (Join-Path $preparationRoot 'delivery-lib.ps1')
. (Join-Path $preparationRoot 'source-scope.ps1')

$candidate = Assert-AbsentAbsolutePath $CandidateDirectory 'candidate directory'
$evidence = Assert-AbsentAbsolutePath $EvidenceDirectory 'evidence directory'
Assert-PathsDisjoint $candidate $evidence 'candidate and evidence'
$version = 'native-worker.20261110.21'
$expectedCandidate = [IO.Path]::GetFullPath(
    (Join-Path $repositoryRoot 'native/bin/levnote-native-worker-20261110.21'))
if (-not [string]::Equals(
        $candidate, $expectedCandidate, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Candidate path must be the fresh .21 directory: $candidate"
}
if (-not [string]::Equals(
        $repositoryRoot, 'F:\AIproject\LevNote',
        [StringComparison]::OrdinalIgnoreCase)) {
    throw "Preparation must run from F:\AIproject\LevNote; resolved to $repositoryRoot"
}

$nativeRoot = Assert-AbsoluteDirectory (Join-Path $repositoryRoot 'native') 'native root'
$protectedRelease = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'bin/levnote-native-worker-20261110.20') `
    'protected .20 package'
$protectedEvidence = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'artifacts/acceptance/drawio-route-release20-20261110-031456') `
    'protected .20 direct evidence'

$sourceGateFull = @(
    foreach ($path in $SourceGatePaths) {
        $full = Assert-AbsolutePath $path 'accepted source gate'
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
            throw "Accepted source gate is absent: $full"
        }
        $full
    }
) | Sort-Object -Unique -CaseSensitive
if ($sourceGateFull.Count -lt 6) {
    throw 'Supply at least the frozen warning and retention result/review/acceptance gate files.'
}
if (@($sourceGateFull | Where-Object {
    (Split-Path $_ -Leaf) -match '(?i)warning|structural'
}).Count -eq 0) {
    throw 'SourceGatePaths must include the accepted structural-warning gate evidence.'
}
if (@($sourceGateFull | Where-Object {
    (Split-Path $_ -Leaf) -notmatch '(?i)warning|structural'
}).Count -gt 3) {
    throw 'Supply only the accepted structural-warning result, review and acceptance gates.'
}
if (@($sourceGateFull | Where-Object {
    (Split-Path $_ -Leaf) -match '(?i)retention|efficiency'
}).Count -eq 0) {
    throw 'SourceGatePaths must include the accepted source-retention gate evidence.'
}
if (@($sourceGateFull | Where-Object {
    (Split-Path $_ -Leaf) -notmatch '(?i)retention|efficiency'
}).Count -gt 3) {
    throw 'Supply only the accepted source-retention result, review and acceptance gates.'
}

$benchmarkFull = @(
    foreach ($path in $BenchmarkFixturePaths) {
        $full = Assert-AbsolutePath $path 'benchmark fixture'
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
            throw "Benchmark fixture is absent: $full"
        }
        $full
    }
) | Sort-Object -Unique -CaseSensitive
if ($benchmarkFull.Count -eq 0) {
    throw 'BenchmarkFixturePaths must name the accepted source-retention benchmark fixture.'
}
$sourceSet = [Collections.Generic.HashSet[string]]::new(
    [StringComparer]::OrdinalIgnoreCase)
foreach ($path in @(Get-DeliverySourceInputPaths $repositoryRoot $nativeRoot)) {
    $sourceSet.Add([IO.Path]::GetFullPath($path)) | Out-Null
}
foreach ($path in $benchmarkFull) {
    if (-not $sourceSet.Contains($path)) {
        throw "Benchmark fixture is outside the scoped source/build inputs: $path"
    }
}

$smokeFixturePath = [IO.Path]::GetFullPath(
    (Join-Path $nativeRoot 'core/engineruntime/drawio_managed_bundle_test.go'))
if (-not $sourceSet.Contains($smokeFixturePath)) {
    throw 'Managed smoke fixture is outside the scoped source/build inputs.'
}
$smokeFixture = Get-FileFingerprint $smokeFixturePath

$sourceGateFingerprints = @(
    foreach ($path in $sourceGateFull) {
        Get-FileFingerprint $path
    }
)
$benchmarkFingerprints = @(
    foreach ($path in $benchmarkFull) {
        Get-FileFingerprint $path
    }
)

New-Item -ItemType Directory -Path $evidence -ErrorAction Stop | Out-Null
$copiedScripts = @(
    foreach ($script in @(
        'delivery-lib.ps1',
        'source-scope.ps1',
        'initialize-context.ps1',
        'invoke-stage.ps1',
        'run-stage.ps1',
        'baseline.ps1',
        'run-package.ps1',
        'run-managed-smoke.ps1',
        'run-host-smoke.ps1',
        'aggregate.ps1',
        'static-parse.ps1'
    )) {
        $source = Join-Path $preparationRoot $script
        $destination = Join-Path $evidence $script
        Copy-Item -LiteralPath $source -Destination $destination -ErrorAction Stop
        Get-FileFingerprint $destination
    }
)
Copy-Item -LiteralPath (Join-Path $preparationRoot 'README.md') `
    -Destination (Join-Path $evidence 'README.md') -ErrorAction Stop

$context = [ordered]@{
    Schema = 1
    PreparedAtUtc = [DateTime]::UtcNow.ToString('o')
    ExecutionAuthorized = [bool]$ExecuteDelivery
    RepositoryRoot = $repositoryRoot
    NativeRoot = $nativeRoot
    PreparationDirectory = $preparationRoot
    CandidateDirectory = $candidate
    EvidenceDirectory = $evidence
    Version = $version
    PackageScript = [IO.Path]::GetFullPath(
        (Join-Path $nativeRoot 'scripts/package-markdown-worker.ps1'))
    GoWrapper = [IO.Path]::GetFullPath((Join-Path $nativeRoot 'scripts/go.ps1'))
    ExpectedGuiBuilds = 5
    ExpectedCandidateFiles = 25
    ExpectedEngineCount = 4
    ExpectedReleaseManifestSchema = 1
    ExpectedReleaseManifestEntries = 24
    PortableSourceReadme = [IO.Path]::GetFullPath(
        (Join-Path $nativeRoot 'portable/README.md'))
    PortableSourceReadmeBytes = 12455
    PortableSourceReadmeSHA256 = '700243D30C97FD8B72110FBC5D96C4794CEAF44461A5197CBDE5E4EC5F0BA80E'
    ProtectedReleaseDirectory = $protectedRelease
    ProtectedReleaseFileCount = 25
    ProtectedReleaseTotalBytes = 26854177
    ProtectedReleaseInventoryIdentity = '6B899EFB2FB2A90CB2FB93CFE9BC48A3B94B4CFA2AB16C61C5AB005B05DF14ED'
    ProtectedReleaseVersion = 'native-worker.20261110.20'
    ProtectedReleaseHostSHA256 = '4B776F634A3626F2BDD15303E175FE38CC4908D566AC54A6D25F51D989B4F34E'
    ProtectedReleaseManifestSHA256 = '53C0B71C317734001596988F6AB2BF44201879CC322BA9DB3B9AB494E553DBFA'
    ProtectedEvidenceDirectory = $protectedEvidence
    ProtectedEvidenceFileCount = 139
    ProtectedEvidenceTotalBytes = 54304954
    ProtectedEvidenceInventoryIdentity = '5ED2F19531E178E33F355164546AB50C5C31F37E032F9E8B7CCBB6791C9C8621'
    ProtectedEvidenceAggregateSHA256 = 'C1CAEC5205F8A833A4DFF86B84485E4D84187CAC3BD6D3DA88DA1F47D7E59611'
    SourceGatePaths = @($sourceGateFull)
    SourceGateFingerprints = @($sourceGateFingerprints)
    BenchmarkFixturePaths = @($benchmarkFull)
    BenchmarkFixtureFingerprints = @($benchmarkFingerprints)
    ReviewedDiagramNotesPath = [IO.Path]::GetFullPath(
        (Join-Path $nativeRoot 'examples/plugins/diagram-notes.json'))
    ReviewedDiagramNotesBytes = 179
    ReviewedDiagramNotesSHA256 = '317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07'
    SmokeFixturePath = $smokeFixturePath
    SmokeFixture = $smokeFixture
    SmokeTestName = '^TestDrawioManagedRealBundleSmoke$'
    ExpectedSmokeSourceBytes = 516
    ExpectedSmokePayloadBytes = 598
    ExpectedSmokePayloadSHA256 = '3f990a4aac3209bafad94f44026bf14f3cec27ea44bdbd3169555947936472bf'
    ExpectedSmokeWarningCount = 0
    CopiedHarnessScripts = $copiedScripts
    Scope = 'Fresh .21 package/evidence; scoped current product/build inputs; accepted warning/retention gates and benchmark fixture; immutable complete .20 package/direct evidence only.'
}
Write-JsonFile (Join-Path $evidence 'context.json') $context 16
Write-JsonFile (Join-Path $evidence 'commands.json') ([ordered]@{
    Schema = 1
    Schedule = @(
        'baseline.ps1',
        'run-package.ps1',
        'run-managed-smoke.ps1',
        'run-host-smoke.ps1',
        'aggregate.ps1'
    )
    Wrapper = 'run-stage.ps1 -Name baseline|package|managed-smoke|host-smoke|aggregate'
    PackageInvocationCount = 1
    ManagedTesterCompileCount = 1
    ManagedTesterRunCount = 1
    HiddenHostStartCount = 1
}) 8

[ordered]@{
    Status = $(if ($ExecuteDelivery) { 'AUTHORIZED' } else { 'PREPARED-ONLY' })
    EvidenceDirectory = $evidence
    CandidateDirectory = $candidate
    Version = $version
    SourceGateFileCount = $sourceGateFull.Count
    BenchmarkFixtureCount = $benchmarkFull.Count
    CopiedHarnessScriptCount = $copiedScripts.Count
} | ConvertTo-Json
