param(
    [Parameter(Mandatory = $true)][string]$CandidateDirectory,
    [Parameter(Mandatory = $true)][string]$EvidenceDirectory,
    [Parameter(Mandatory = $true)][string]$FinalStage2FilesManifest,
    [Parameter(Mandatory = $true)][string]$FinalStage2EvidenceDirectory,
    [Parameter(Mandatory = $true)][string[]]$FinalStage2GatePaths,
    [string]$Version = 'native-worker.20261110.19',
    [switch]$ExecuteDelivery
)

$ErrorActionPreference = 'Stop'
$preparationRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$repositoryRoot = [IO.Path]::GetFullPath((Join-Path $preparationRoot '..\..\..'))
. (Join-Path $preparationRoot 'delivery-lib.ps1')

$candidate = Assert-AbsentAbsolutePath $CandidateDirectory 'candidate directory'
$evidence = Assert-AbsentAbsolutePath $EvidenceDirectory 'evidence directory'
Assert-PathsDisjoint $candidate $evidence 'candidate and evidence'
if ($Version -cne 'native-worker.20261110.19') {
    throw "Version must be native-worker.20261110.19, found $Version"
}
if ((Split-Path $candidate -Leaf) -cne 'levnote-native-worker-20261110.19') {
    throw "Candidate directory leaf must be levnote-native-worker-20261110.19: $candidate"
}

$nativeRoot = Assert-AbsoluteDirectory (Join-Path $repositoryRoot 'native') 'native root'
$stage2ManifestPath = Assert-AbsolutePath $FinalStage2FilesManifest 'final Stage2 file manifest'
if (-not (Test-Path -LiteralPath $stage2ManifestPath -PathType Leaf)) {
    throw "Final Stage2 file manifest is absent: $stage2ManifestPath"
}
$stage2Evidence = Assert-AbsoluteDirectory $FinalStage2EvidenceDirectory 'final Stage2 evidence directory'
$protectedRelease = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'bin/levnote-native-worker-20261110.18') 'protected .18 package'
$protectedEvidence = Assert-AbsoluteDirectory `
    (Join-Path $nativeRoot 'artifacts/acceptance/drawio-svg-budget-release18-20261110-011852') `
    'protected .18 evidence'

$requiredDirectories = @(
    'handoffs/drawio-worker-stage1-evidence',
    'handoffs/drawio-worker-stage1-warning-correction-evidence',
    'handoffs/drawio-worker-view-contract-evidence',
    'handoffs/drawio-worker-view-contract-reentry-correction-evidence',
    'handoffs/drawio-worker-view-warning-followup-evidence',
    'handoffs/drawio-worker-real-smoke-compile-evidence',
    'handoffs/drawio-worker-portable-docs-evidence'
)
$acceptedEvidenceDirectories = @(
    foreach ($relative in $requiredDirectories) {
        Assert-AbsoluteDirectory (Join-Path $nativeRoot $relative) $relative
    }
    $stage2Evidence
)

$requiredGateFiles = @(
    'handoffs/drawio-worker-stage1-root-acceptance.md',
    'handoffs/drawio-worker-stage1-result.md',
    'handoffs/drawio-worker-stage1-independent-review.md',
    'handoffs/drawio-worker-stage1-warning-correction-result.md',
    'handoffs/drawio-worker-view-contract-root-acceptance.md',
    'handoffs/drawio-worker-view-contract-result.md',
    'handoffs/drawio-worker-view-contract-independent-review.md',
    'handoffs/drawio-worker-view-contract-reentry-correction-result.md',
    'handoffs/drawio-worker-view-warning-followup-result.md',
    'handoffs/drawio-worker-real-smoke-preparation-result.md',
    'handoffs/drawio-worker-real-smoke-compile-result.md',
    'handoffs/drawio-worker-real-smoke-root-review.md',
    'handoffs/drawio-worker-portable-docs-result.md',
    'handoffs/drawio-worker-portable-wording-result.md',
    'handoffs/drawio-worker-release19-delivery-task.md'
)
$acceptedGatePaths = @(
    foreach ($relative in $requiredGateFiles) {
        $path = Join-Path $nativeRoot $relative
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Accepted gate file is absent: $path"
        }
        [IO.Path]::GetFullPath($path)
    }
)
foreach ($gatePath in $FinalStage2GatePaths) {
    $full = Assert-AbsolutePath $gatePath 'final Stage2 gate path'
    if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
        throw "Final Stage2 gate file is absent: $full"
    }
    $acceptedGatePaths += $full
}
$acceptedGatePaths = @($acceptedGatePaths | Sort-Object -Unique -CaseSensitive)
if ($acceptedGatePaths.Count -lt ($requiredGateFiles.Count + 1)) {
    throw 'Final Stage2 delivery context must name at least one frozen Stage2 gate file.'
}

. (Join-Path $preparationRoot 'source-scope.ps1')
$sourcePaths = @(Get-DeliverySourceInputPaths $repositoryRoot $nativeRoot)
$sourceSet = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($path in $sourcePaths) {
    $sourceSet.Add([IO.Path]::GetFullPath($path)) | Out-Null
}

$stage2Manifest = Read-JsonFile $stage2ManifestPath
if ($stage2Manifest.schema -ne 1) {
    throw "Final Stage2 file manifest schema must be 1, found $($stage2Manifest.schema)"
}
$stage2Files = @($stage2Manifest.files)
if ($stage2Files.Count -eq 0) {
    throw 'Final Stage2 file manifest must contain at least one file.'
}
$stage2Normalized = @(
    foreach ($entry in $stage2Files) {
        $relative = ([string]$entry.relativePath).Replace('\', '/')
        if ([string]::IsNullOrWhiteSpace($relative) -or [IO.Path]::IsPathRooted($relative) -or
            @($relative.Split('/') | Where-Object { $_ -eq '..' }).Count -ne 0) {
            throw "Invalid final Stage2 relative path: $relative"
        }
        $absolute = [IO.Path]::GetFullPath((Join-Path $repositoryRoot $relative))
        if (-not $sourceSet.Contains($absolute)) {
            throw "Final Stage2 file is outside the scoped source inputs: $relative"
        }
        $actual = Get-FileFingerprint $absolute
        if ([Int64]$entry.bytes -ne $actual.Bytes -or
            ([string]$entry.sha256).ToUpperInvariant() -cne $actual.SHA256) {
            throw "Final Stage2 manifest identity differs from the current file: $relative"
        }
        [ordered]@{
            RelativePath = $relative
            Bytes = $actual.Bytes
            SHA256 = $actual.SHA256
        }
    }
)
if (@($stage2Normalized.RelativePath | Sort-Object -Unique -CaseSensitive).Count -ne $stage2Normalized.Count) {
    throw 'Final Stage2 file manifest contains duplicate paths.'
}

New-Item -ItemType Directory -Path $evidence -ErrorAction Stop | Out-Null
Copy-Item -LiteralPath $stage2ManifestPath -Destination (Join-Path $evidence 'final-stage2-input.json') -ErrorAction Stop
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
        'aggregate.ps1'
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
    Version = $Version
    PackageScript = [IO.Path]::GetFullPath((Join-Path $nativeRoot 'scripts/package-markdown-worker.ps1'))
    GoWrapper = [IO.Path]::GetFullPath((Join-Path $nativeRoot 'scripts/go.ps1'))
    ExpectedGuiBuilds = 5
    ExpectedCandidateFiles = 25
    ExpectedEngineCount = 4
    ExpectedReleaseManifestSchema = 1
    ProtectedReleaseDirectory = $protectedRelease
    ProtectedReleaseFileCount = 23
    ProtectedReleaseTotalBytes = 23699526
    ProtectedReleaseInventoryIdentity = '4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991'
    ProtectedReleaseVersion = 'native-worker.20261110.18'
    ProtectedReleaseHostSHA256 = 'BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A'
    ProtectedReleaseManifestSHA256 = '8CA2016C0064638BE965FDFE74468F8004A37143BE568A189FD7FEDAB2D12FDD'
    ProtectedEvidenceDirectory = $protectedEvidence
    ProtectedEvidenceFileCount = 117
    ProtectedEvidenceTotalBytes = 47816959
    ProtectedEvidenceInventoryIdentity = '8D73702A59E34583FB77E567AF7167A16A06698F751A16D623BDD469EE66C948'
    ProtectedEvidenceAggregateSHA256 = 'D1066FA721005A7F4DC97184BC8E1E802B8FE86013609B5A2F64975D81E14FF0'
    AcceptedEvidenceDirectories = $acceptedEvidenceDirectories
    AcceptedGatePaths = $acceptedGatePaths
    FinalStage2FilesManifest = $stage2ManifestPath
    FinalStage2EvidenceDirectory = $stage2Evidence
    FinalStage2Files = $stage2Normalized
    CopiedHarnessScripts = $copiedScripts
    SmokeFixturePath = [IO.Path]::GetFullPath((Join-Path $nativeRoot 'core/engineruntime/drawio_managed_bundle_test.go'))
    SmokeFixtureBytes = 21150
    SmokeFixtureSHA256 = '074C4894D1D05F0FEBE7B49F61BF958980BDDF6DFEF9151179A18C05A2FF4548'
    SmokeTestName = '^TestDrawioManagedRealBundleSmoke$'
    Scope = 'Fresh local .19 candidate and evidence roots only; no toolchain or complete historical-release inventory.'
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
}) 6

[ordered]@{
    Status = $(if ($ExecuteDelivery) { 'AUTHORIZED' } else { 'PREPARED-ONLY' })
    EvidenceDirectory = $evidence
    CandidateDirectory = $candidate
    Version = $Version
    FinalStage2FileCount = $stage2Normalized.Count
    CopiedHarnessScriptCount = $copiedScripts.Count
    SourceInputCount = $sourcePaths.Count
    AcceptedEvidenceDirectoryCount = $acceptedEvidenceDirectories.Count
    AcceptedGateFileCount = $acceptedGatePaths.Count
} | ConvertTo-Json
