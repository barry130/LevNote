$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (Test-Path -LiteralPath (Join-Path $evidenceRoot 'aggregate.json') -PathType Leaf) {
    throw 'Aggregate already completed in this evidence directory.'
}

$checks = [Collections.Generic.List[object]]::new()
function Check([string]$Name, [bool]$Pass, [string]$Detail = '') {
    $checks.Add([ordered]@{ Name = $Name; Pass = $Pass; Detail = $Detail })
}

$baseline = Read-JsonFile (Join-Path $evidenceRoot 'baseline-result.json')
$packageCommand = Read-JsonFile (Join-Path $evidenceRoot 'package-command.json')
$packageResult = Read-JsonFile (Join-Path $evidenceRoot 'package-result.json')
$hostResult = Read-JsonFile (Join-Path $evidenceRoot 'host-smoke/result.json')
$hostSession = Read-JsonFile (Join-Path $evidenceRoot 'host-smoke/session-after-close.json')
$packageInternal = Read-JsonFile (Join-Path $evidenceRoot 'package/package-result.json')

$buildNames = @('worker-markdown', 'worker-mindmap', 'worker-mindmap-preview', 'worker-drawio', 'host')
$builds = @(
    foreach ($name in $buildNames) {
        Read-JsonFile (Join-Path $evidenceRoot ("package/{0}-build.json" -f $name))
    }
)
$buildBad = @(
    $builds | Where-Object {
        $_.ExitCode -ne 0 -or $_.StdoutBytes -ne 0 -or $_.StderrBytes -ne 0 -or
        (Get-Item -LiteralPath $_.StdoutPath).Length -ne 0 -or
        (Get-Item -LiteralPath $_.StderrPath).Length -ne 0
    }
)
Check 'one package-script invocation made the required five Windows GUI builds with exit 0 and empty build streams' (
    $baseline.CandidateAbsent -and
    $packageCommand.InvocationCount -eq 1 -and
    $packageCommand.ExitCode -eq 0 -and
    $packageCommand.ExpectedGuiBuilds -eq 5 -and
    $packageResult.PackageInvocationCount -eq 1 -and
    $packageResult.GuiBuildCount -eq 5 -and
    $builds.Count -eq 5 -and $buildBad.Count -eq 0 -and
    $packageCommand.PackageArguments.OutputDirectory -ceq $ctx.candidateDirectory -and
    $packageCommand.PackageArguments.EvidenceDirectory -ceq (Join-Path $evidenceRoot 'package') -and
    $packageCommand.PackageArguments.Version -ceq $ctx.version
) "builds=$($builds.Count)"

$candidate = @(Get-DirectoryInventory $ctx.candidateDirectory)
$packageCopy = @(Get-DirectoryInventory (Join-Path $evidenceRoot 'package-copy'))
$candidateIdentity = Get-InventoryIdentity $candidate
$copyMatches = $true
try {
    Assert-InventoryContainsExact $candidate $packageCopy 'Complete package evidence copy' $false
} catch {
    $copyMatches = $false
}
Check 'actual fresh candidate inventory matches the fixed four-engine plus host/asset artifact set and its complete package copy' (
    $candidate.Count -eq 25 -and $candidate.Count -eq $packageResult.CandidateFileCount -and
    $packageResult.ExpectedCandidateFileCount -eq 25 -and
    $packageResult.ExpectedCandidateFileSetPass -and
    $candidateIdentity -ceq $packageResult.CandidateInventoryIdentity -and
    $copyMatches -and $packageCopy.Count -eq $candidate.Count
) "actual=$($candidate.Count), expectedAssertion=25, bytes=$(Get-InventoryTotalBytes $candidate)"

$releasePath = Join-Path $ctx.candidateDirectory 'release-manifest.json'
$release = Read-JsonFile $releasePath
$releaseEntries = @($release.files)
$releaseBad = @(
    foreach ($entry in $releaseEntries) {
        $actual = @($candidate | Where-Object { $_.RelativePath -ceq [string]$entry.path })
        if ($actual.Count -ne 1 -or
            [Int64]$actual[0].Bytes -ne [Int64]$entry.size -or
            $actual[0].SHA256.ToLowerInvariant() -cne ([string]$entry.sha256).ToLowerInvariant()) {
            [string]$entry.path
        }
    }
)
$hostFingerprint = Get-FileFingerprint (Join-Path $ctx.candidateDirectory 'levnote-native.exe')
$manifestFingerprint = Get-FileFingerprint $releasePath
Check 'release manifest is schema1 for .22 and all 24 non-manifest payload entries match actual size and SHA-256' (
    $release.schema -eq 1 -and $release.version -ceq $ctx.version -and
    $release.goos -ceq 'windows' -and $release.goarch -ceq 'amd64' -and
    $release.goamd64 -ceq 'v1' -and $releaseEntries.Count -eq 24 -and
    @($release.excludes).Count -eq 1 -and @($release.excludes)[0] -ceq 'release-manifest.json' -and
    $releaseBad.Count -eq 0 -and
    $packageInternal.ReleaseManifestSHA256 -ceq $manifestFingerprint.SHA256 -and
    $packageInternal.HostSHA256 -ceq $hostFingerprint.SHA256
) "entries=$($releaseEntries.Count)"

$fixedEngineNames = @(
    'levnote.markdown-export',
    'levnote.mindmap-svg',
    'levnote.mindmap-preview',
    'levnote.drawio-svg'
)
$engineNames = @($packageResult.EngineChecks | ForEach-Object { $_.Engine } | Sort-Object)
Check 'four fixed catalog engine manifests match the .22 version and each worker payload identity' (
    $packageResult.EngineChecks.Count -eq 4 -and
    @(Compare-Object ($fixedEngineNames | Sort-Object) $engineNames).Count -eq 0 -and
    @($packageResult.EngineChecks | Where-Object { -not $_.Pass }).Count -eq 0
)

$peFacts = @($packageResult.PEFacts)
Check 'five executable payloads are Windows amd64 PE32+ GUI images' (
    $peFacts.Count -eq 5 -and @($peFacts | Where-Object { -not $_.Pass }).Count -eq 0
)
Check 'all portable copied assets and authored local package links were checked successfully' (
    $packageResult.SourceCopyCount -gt 0 -and
    @($packageResult.SourceCopies | Where-Object { -not $_.Pass }).Count -eq 0 -and
    $packageResult.LocalAuthoredLinksChecked -and $packageInternal.LocalAuthoredLinksChecked
)
$packageReadmePath = Join-Path $ctx.candidateDirectory 'README.md'
$packageReadmeText = [IO.File]::ReadAllText(
    $packageReadmePath, [Text.Encoding]::UTF8)
$portableReadme = Get-FileFingerprint $ctx.portableSourceReadme
$packageReadme = Get-FileFingerprint $packageReadmePath
$portableReadmeBefore = [pscustomobject]$ctx.PortableSourceReadmeFingerprint
Check 'portable README has one version marker, three exact schema3 routes and a rendered .22 version' (
    (Test-FingerprintEqual $portableReadmeBefore $portableReadme) -and
    $packageResult.PortableReadmeSource.Bytes -eq $portableReadme.Bytes -and
    $packageResult.PortableReadmeSource.SHA256 -ceq $portableReadme.SHA256 -and
    $packageResult.PortableReadmePackage.SHA256 -ceq $packageReadme.SHA256 -and
    $packageResult.PortableReadmeMarkerCount -eq 1 -and
    $packageResult.PortableReadmeInlineSchema3JSONCount -eq 3 -and
    (@($packageResult.PortableReadmeRoutes | Sort-Object) -join ',') -ceq
        (@('builtin.drawio', 'builtin.markdown', 'builtin.mindmap') -join ',') -and
    $portableReadme.SHA256 -cne $packageReadme.SHA256 -and
    -not $packageReadmeText.Contains('{{VERSION}}') -and
    $packageReadmeText.Contains($ctx.version)
) "source=$($portableReadme.Bytes) bytes, package=$($packageReadme.Bytes) bytes"

$protectedBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-before.json'))
$protectedAfter = @(Get-DirectoryInventory $ctx.protectedReleaseDirectory)
$protectedEvidenceBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-evidence-before.json'))
$protectedEvidenceAfter = @(Get-DirectoryInventory $ctx.protectedEvidenceDirectory)
$protectedOK = $true
try {
    Assert-InventoriesEqual $protectedBefore $protectedAfter 'Protected .21 package'
    Assert-InventoriesEqual $protectedEvidenceBefore $protectedEvidenceAfter 'Protected .21 direct evidence'
} catch {
    $protectedOK = $false
}
Check 'accepted .21 package and direct delivery evidence remain byte/time identical to their complete accepted inventories' (
    $protectedOK -and
    $protectedAfter.Count -eq [int]$ctx.protectedReleaseFileCount -and
    (Get-InventoryTotalBytes $protectedAfter) -eq [Int64]$ctx.protectedReleaseTotalBytes -and
    (Get-InventoryIdentity $protectedAfter) -ceq [string]$ctx.protectedReleaseInventoryIdentity -and
    $protectedEvidenceAfter.Count -eq [int]$ctx.protectedEvidenceFileCount -and
    (Get-InventoryTotalBytes $protectedEvidenceAfter) -eq [Int64]$ctx.protectedEvidenceTotalBytes -and
    (Get-InventoryIdentity $protectedEvidenceAfter) -ceq [string]$ctx.protectedEvidenceInventoryIdentity
) "release=$($protectedAfter.Count) files, evidence=$($protectedEvidenceAfter.Count) files"

$protectedRetryBefore = @(
    Read-JsonFile (Join-Path $evidenceRoot 'protected-retry-before.json'))
$protectedRetryAfter = @(Get-DirectoryInventory $ctx.protectedRetryDirectory)
$protectedRetryOK = $true
try {
    Assert-InventoriesEqual $protectedRetryBefore $protectedRetryAfter 'Protected .21 retry sibling'
} catch {
    $protectedRetryOK = $false
}
Check 'accepted separate .21 aggregate-retry sibling remains byte/time identical to its complete accepted inventory' (
    $protectedRetryOK -and
    $protectedRetryAfter.Count -eq [int]$ctx.protectedRetryFileCount -and
    (Get-InventoryTotalBytes $protectedRetryAfter) -eq [Int64]$ctx.protectedRetryTotalBytes -and
    (Get-InventoryIdentity $protectedRetryAfter) -ceq [string]$ctx.protectedRetryInventoryIdentity
) "files=$($protectedRetryAfter.Count), bytes=$(Get-InventoryTotalBytes $protectedRetryAfter)"

$sourceBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'inputs-before.json'))
$sourcePaths = @(Get-DeliverySourceInputPaths $ctx.repositoryRoot $ctx.nativeRoot)
$sourceAfter = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
$sourceOK = $true
try {
    Assert-InventoriesEqual $sourceBefore $sourceAfter 'Scoped source/build input'
} catch {
    $sourceOK = $false
}
$noticePath = [IO.Path]::GetFullPath((Join-Path $ctx.nativeRoot 'portable/NOTICE-PREAMBLE.md'))
$formatsPrefix = (Join-Path $ctx.nativeRoot 'formats').TrimEnd('\') + '\'
$formatsCount = @($sourceAfter | Where-Object {
    ([IO.Path]::GetFullPath([string]$_.Path)).StartsWith($formatsPrefix, [StringComparison]::OrdinalIgnoreCase)
}).Count
Check 'scoped source/build inputs, native/formats and portable NOTICE-PREAMBLE.md remain byte/time unchanged' (
    $sourceOK -and $sourceAfter.Count -eq $baseline.SourceInputCount -and
    $formatsCount -gt 0 -and @($sourceAfter | Where-Object {
        [IO.Path]::GetFullPath([string]$_.Path) -eq $noticePath
    }).Count -eq 1
) "files=$($sourceAfter.Count), formats=$formatsCount"
Write-JsonFile (Join-Path $evidenceRoot 'inputs-after.json') $sourceAfter

$sourceGatesBefore = @(
    Read-JsonFile (Join-Path $evidenceRoot 'source-gates-before.json'))
$sourceGatesAfter = @(
    foreach ($path in @($ctx.TabGatePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$sourceGatesOK = $true
try {
    Assert-InventoriesEqual $sourceGatesBefore $sourceGatesAfter `
        'Accepted tab workflow source gates'
} catch {
    $sourceGatesOK = $false
}
Check 'accepted tab workflow result/review/acceptance gate files remain byte/time unchanged' (
    $sourceGatesOK -and
    $sourceGatesAfter.Count -eq $sourceGatesBefore.Count -and
    $sourceGatesAfter.Count -eq @($ctx.TabGateFingerprints).Count
) "gates=$($sourceGatesAfter.Count)"
Check 'scoped source inventory identity remained the accepted tab workflow source identity throughout package and host execution' (
    (Get-InventoryIdentity $sourceAfter) -ceq [string]$ctx.TabSourceIdentity
) "identity=$($ctx.TabSourceIdentity)"

$hostTabs = @($hostSession.tabs)
$hostSourcePath = Join-Path $evidenceRoot 'host-smoke/workspace/route-gate.diagramnote'
$hostTab = @($hostTabs | Where-Object { $_.path -eq $hostSourcePath })
$hostSessionOK = $hostTabs.Count -eq 1 -and $hostTab.Count -eq 1 -and
    $hostTab[0].pluginId -ceq 'diagram-notes' -and
    $hostTab[0].view -ceq 'preview' -and
    $hostTab[0].encoding -ceq 'UTF-8' -and
    $hostTab[0].bom -eq $true -and
    -not ($hostTab[0].PSObject.Properties.Name -contains 'draft') -and
    -not [bool]$hostResult.SessionHasTerminalNewline
$configRoot = [IO.Path]::GetFullPath([string]$hostResult.RunOwnedConfigRoot)
$configPrefix = $configRoot.TrimEnd('\') + '\'
$sessionPath = [IO.Path]::GetFullPath([string]$hostResult.ConfigSessionPath)
$installedPluginPath = [IO.Path]::GetFullPath([string]$hostResult.InstalledPluginPath)
$configPaths = @($hostResult.ConfigFilesAfterClose | ForEach-Object {
    [IO.Path]::GetFullPath([string]$_.Path)
})
$configOK = $hostResult.ConfigRootRetained -and
    [IO.Path]::GetPathRoot($configRoot) -eq 'C:\' -and
    [IO.Path]::GetPathRoot([string]$hostResult.ExecutablePath) -eq 'F:\' -and
    $configPaths.Count -ge 1 -and
    $configPaths -contains $sessionPath -and
    $configPaths -contains $installedPluginPath -and
    @($configPaths | Where-Object {
        -not $_.StartsWith($configPrefix, [StringComparison]::OrdinalIgnoreCase)
    }).Count -eq 0
$hostBundle = @(Get-DirectoryInventory (Join-Path $evidenceRoot 'host-smoke/bundle'))
$hostBundleOK = $true
try {
    Assert-InventoryContainsExact $candidate $hostBundle 'Hidden host bundle' $false
} catch {
    $hostBundleOK = $false
}
$hostOK = $hostResult.Status -eq 'PASS' -and
    $hostResult.AppStarts -eq 1 -and $hostResult.StartAttemptCount -eq 1 -and $hostResult.StartedHidden -and
    $hostResult.LaunchMethod -eq 'Start-Process -WindowStyle Hidden' -and
    @($hostResult.AppArguments).Count -eq 1 -and @($hostResult.AppArguments)[0] -ceq $hostSourcePath -and
    $hostResult.App.Path -eq (Join-Path $evidenceRoot 'host-smoke/bundle/levnote-native.exe') -and
    $hostResult.App.PathMethod -eq 'QueryFullProcessImageName(exact held handle)' -and
    $hostResult.WindowFound -and -not $hostResult.WindowVisible -and
    $hostResult.GracefulWMClosePosted -and $hostResult.WMClosePostCount -eq 1 -and -not $hostResult.GracefulCloseTimedOut -and
    -not $hostResult.ForcedAppCleanup -and $hostResult.AppExitCode -eq 0 -and -not $hostResult.OwnedAppStillLive -and
    $hostResult.SessionObserved -and $hostSessionOK -and $hostResult.FixturePreserved -and
    $hostResult.InstalledPluginPreserved -and
    $hostResult.BundlePreserved -and $hostResult.AppStdoutBytes -eq 0 -and $hostResult.AppStderrBytes -eq 0 -and
    $hostResult.JobProcessesAfterAssignment -eq 1 -and
    $hostResult.JobTotalProcessesAfterExit -eq 1 -and
    $hostResult.RequestedWorkerOperations -eq 0 -and $configOK -and $hostBundleOK
Check 'one byte-identical copied .22 host started hidden, requested no worker, and exited normally after one owned WM_CLOSE' $hostOK
Check 'the hidden host fixture, installed plugin JSON, candidate and copied bundle remained byte/time unchanged' (
    $hostResult.FixturePreserved -and $hostResult.InstalledPluginPreserved -and
    $hostResult.BundlePreserved -and
    $hostResult.FixtureBefore.SHA256 -ceq $hostResult.FixtureAfter.SHA256 -and
    $hostResult.InstalledPluginBefore.SHA256 -ceq
        '317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07' -and
    $hostResult.InstalledPluginAfter.SHA256 -ceq
        '317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07'
)

$pass = @($checks | Where-Object { -not $_.Pass }).Count -eq 0
$validator = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    Pass = $pass
    Passed = @($checks | Where-Object { $_.Pass }).Count
    Total = $checks.Count
    Checks = $checks
    Version = $ctx.version
    CandidateDirectory = $ctx.candidateDirectory
    CandidateFileCount = $candidate.Count
    CandidateTotalBytes = Get-InventoryTotalBytes $candidate
    CandidateInventoryIdentity = $candidateIdentity
    Host = $hostFingerprint
    ReleaseManifest = $manifestFingerprint
    ReleaseManifestPayloadEntries = $releaseEntries.Count
    EngineCount = $packageResult.EngineChecks.Count
    GuiBuildCount = $builds.Count
    SourceCopyCount = $packageResult.SourceCopyCount
    ProtectedReleaseFileCount = $protectedAfter.Count
    ProtectedReleaseTotalBytes = Get-InventoryTotalBytes $protectedAfter
    ProtectedReleaseInventoryIdentity = Get-InventoryIdentity $protectedAfter
    ProtectedEvidenceFileCount = $protectedEvidenceAfter.Count
    ProtectedEvidenceTotalBytes = Get-InventoryTotalBytes $protectedEvidenceAfter
    ProtectedEvidenceInventoryIdentity = Get-InventoryIdentity $protectedEvidenceAfter
    ProtectedRetryFileCount = $protectedRetryAfter.Count
    ProtectedRetryTotalBytes = Get-InventoryTotalBytes $protectedRetryAfter
    ProtectedRetryInventoryIdentity = Get-InventoryIdentity $protectedRetryAfter
    SourceInputCount = $sourceAfter.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceAfter
    SourceIdentity = Get-InventoryIdentity $sourceAfter
    TabGateFileCount = $sourceGatesAfter.Count
    HiddenAppStarts = $hostResult.AppStarts
    HiddenAppPID = $hostResult.App.Id
    HiddenAppHandle = $hostResult.App.HeldHandle
    HiddenHostConfigRoot = $configRoot
    HiddenHostConfigFiles = @($hostResult.ConfigFilesAfterClose)
    Scope = 'Exact local .22 package selection only: one package-script invocation/five Windows GUI builds, actual 25-file candidate and schema1 manifest, four fixed engine manifests, immutable complete .21 package plus separate direct/retry evidence, accepted tab gate protection, and one separate hidden custom-suffix host startup/WM_CLOSE. No managed worker request, app suite, tester compile, benchmark, physical IME/UI, dirty Save, process-memory, CPU, or packaged interactive SVG claim.'
}
$validatorPath = Join-Path $evidenceRoot 'validator.json'
Write-JsonFile $validatorPath $validator 16
Write-JsonFile (Join-Path $evidenceRoot 'protected-after.json') $protectedAfter
Write-JsonFile (Join-Path $evidenceRoot 'protected-evidence-after.json') $protectedEvidenceAfter

$aggregate = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    Pass = $pass
    Passed = $validator.Passed
    Total = $validator.Total
    Version = $ctx.version
    CandidateDirectory = $ctx.candidateDirectory
    FileCount = $candidate.Count
    TotalBytes = Get-InventoryTotalBytes $candidate
    CandidateInventoryIdentity = $candidateIdentity
    Host = $hostFingerprint
    ReleaseManifest = $manifestFingerprint
    GuiBuildCount = $builds.Count
    EngineCount = $packageResult.EngineChecks.Count
    HiddenAppStarts = $hostResult.AppStarts
    HostWorkerOperations = $hostResult.RequestedWorkerOperations
    SourceIdentity = Get-InventoryIdentity $sourceAfter
    TabGateFileCount = $sourceGatesAfter.Count
    ProtectedRetryFileCount = $protectedRetryAfter.Count
    ProtectedRetryTotalBytes = Get-InventoryTotalBytes $protectedRetryAfter
    ProtectedRetryInventoryIdentity = Get-InventoryIdentity $protectedRetryAfter
    AggregateCaptureDirectory = Join-Path (Split-Path $evidenceRoot -Parent) (
        (Split-Path $evidenceRoot -Leaf) + '-aggregate-capture')
    Scope = $validator.Scope
}
$aggregatePath = Join-Path $evidenceRoot 'aggregate.json'
$records = @(
    Get-ChildItem -LiteralPath $evidenceRoot -File -Recurse |
        Sort-Object FullName |
        ForEach-Object { Get-FileFingerprint $_.FullName }
)
$aggregate.Evidence = $records
Write-JsonFile $aggregatePath $aggregate 16
[ordered]@{
    Pass = $pass
    Passed = $validator.Passed
    Total = $validator.Total
    CandidateDirectory = $ctx.candidateDirectory
    FileCount = $candidate.Count
    TotalBytes = Get-InventoryTotalBytes $candidate
    HostSHA256 = $hostFingerprint.SHA256
    ReleaseManifestSHA256 = $manifestFingerprint.SHA256
    ValidatorSHA256 = (Get-FileHash -LiteralPath $validatorPath -Algorithm SHA256).Hash
    AggregateSHA256 = (Get-FileHash -LiteralPath $aggregatePath -Algorithm SHA256).Hash
} | ConvertTo-Json
if (-not $pass) {
    exit 1
}
