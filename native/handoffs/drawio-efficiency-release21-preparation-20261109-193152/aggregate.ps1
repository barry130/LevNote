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
$managed = Read-JsonFile (Join-Path $evidenceRoot 'managed-smoke/result.json')
$managedRecord = Read-JsonFile (Join-Path $evidenceRoot 'managed-smoke/record.json')
$managedCompile = Read-JsonFile (Join-Path $evidenceRoot 'managed-smoke/compile-invocation.json')
$managedRun = Read-JsonFile (Join-Path $evidenceRoot 'managed-smoke/run-invocation.json')
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
Check 'release manifest is schema1 for .21 and all 24 non-manifest payload entries match actual size and SHA-256' (
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
Check 'four fixed catalog engine manifests match the .21 version and each worker payload identity' (
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
Check 'portable README has one version marker, three exact schema3 routes and a rendered .21 version' (
    $portableReadme.Bytes -eq [Int64]$ctx.PortableSourceReadmeBytes -and
    $portableReadme.SHA256 -ceq [string]$ctx.PortableSourceReadmeSHA256 -and
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
    Assert-InventoriesEqual $protectedBefore $protectedAfter 'Protected .20 package'
    Assert-InventoriesEqual $protectedEvidenceBefore $protectedEvidenceAfter 'Protected .20 direct evidence'
} catch {
    $protectedOK = $false
}
Check 'accepted .20 package and direct delivery evidence remain byte/time identical to their complete accepted inventories' (
    $protectedOK -and
    $protectedAfter.Count -eq [int]$ctx.protectedReleaseFileCount -and
    (Get-InventoryTotalBytes $protectedAfter) -eq [Int64]$ctx.protectedReleaseTotalBytes -and
    (Get-InventoryIdentity $protectedAfter) -ceq [string]$ctx.protectedReleaseInventoryIdentity -and
    $protectedEvidenceAfter.Count -eq [int]$ctx.protectedEvidenceFileCount -and
    (Get-InventoryTotalBytes $protectedEvidenceAfter) -eq [Int64]$ctx.protectedEvidenceTotalBytes -and
    (Get-InventoryIdentity $protectedEvidenceAfter) -ceq [string]$ctx.protectedEvidenceInventoryIdentity
) "release=$($protectedAfter.Count) files, evidence=$($protectedEvidenceAfter.Count) files"

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
    foreach ($path in @($ctx.SourceGatePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$sourceGatesOK = $true
try {
    Assert-InventoriesEqual $sourceGatesBefore $sourceGatesAfter `
        'Accepted warning/retention source gates'
} catch {
    $sourceGatesOK = $false
}
Check 'accepted warning and retention result/review/acceptance gates remain byte/time unchanged' (
    $sourceGatesOK -and
    $sourceGatesAfter.Count -eq $sourceGatesBefore.Count -and
    @($sourceGatesAfter | Where-Object {
        (Split-Path $_.Path -Leaf) -match '(?i)warning|structural'
    }).Count -gt 0 -and
    @($sourceGatesAfter | Where-Object {
        (Split-Path $_.Path -Leaf) -match '(?i)retention|efficiency'
    }).Count -gt 0
) "gates=$($sourceGatesAfter.Count)"

$benchmarkFixturesBefore = @(
    Read-JsonFile (Join-Path $evidenceRoot 'benchmark-fixtures-before.json'))
$benchmarkFixturesAfter = @(
    foreach ($path in @($ctx.BenchmarkFixturePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$benchmarkFixturesOK = $true
try {
    Assert-InventoriesEqual $benchmarkFixturesBefore $benchmarkFixturesAfter `
        'Accepted source-retention benchmark fixtures'
} catch {
    $benchmarkFixturesOK = $false
}
Check 'accepted source-retention benchmark fixture files remain byte/time unchanged' (
    $benchmarkFixturesOK -and
    $benchmarkFixturesAfter.Count -eq $benchmarkFixturesBefore.Count
) "fixtures=$($benchmarkFixturesAfter.Count)"

$smokeFixture = Get-FileFingerprint $ctx.SmokeFixturePath
$smokeFixtureInSource = @($sourceAfter | Where-Object {
    $_.Path -ceq $smokeFixture.Path
})
Check 'managed installed-drawio tester source was captured while the .21 source/package inputs were frozen' (
    $smokeFixtureInSource.Count -eq 1 -and
    $smokeFixtureInSource[0].Bytes -eq $smokeFixture.Bytes -and
    $smokeFixtureInSource[0].SHA256 -ceq $smokeFixture.SHA256 -and
    $smokeFixtureInSource[0].LastWriteUtcTicks -eq $smokeFixture.LastWriteUtcTicks -and
    $smokeFixture.Bytes -eq [Int64]$ctx.SmokeFixture.Bytes -and
    $smokeFixture.SHA256 -ceq [string]$ctx.SmokeFixture.SHA256
) "bytes=$($smokeFixture.Bytes), sha256=$($smokeFixture.SHA256)"

$managedRecordPath = Join-Path $evidenceRoot 'managed-smoke/record.json'
$managedResultPath = Join-Path $evidenceRoot 'managed-smoke/result.json'
$rawWarningState = Get-JsonArrayProperty $managedRecordPath 'Warnings'
$managedWarningState = Get-JsonArrayProperty $managedResultPath 'Warnings'
$warnings = [object[]]@($rawWarningState.Elements)
$normalizedWarnings = [object[]]@($managedWarningState.Elements)
$rawWarningJson = ConvertTo-Json -InputObject $warnings -Depth 6 -Compress
$normalizedWarningJson = ConvertTo-Json -InputObject $normalizedWarnings -Depth 6 -Compress
$warningHash = [Convert]::ToHexString(
    [Security.Cryptography.SHA256]::HashData(
        [Text.Encoding]::UTF8.GetBytes($normalizedWarningJson)))
$warningPropertyKinds = @(
    'WarningPropertyKind',
    'WarningCount',
    'WarningsJson',
    'WarningsSHA256'
)
$missingWarningProperties = @(
    $warningPropertyKinds | Where-Object {
        @($managed.PSObject.Properties.Name) -cnotcontains $_
    }
)
$warningStateOK = $managedWarningState.Kind -ceq 'Array' -and
    $rawWarningJson -ceq $normalizedWarningJson -and
    $missingWarningProperties.Count -eq 0 -and
    [string]$managed.WarningPropertyKind -ceq $rawWarningState.Kind -and
    [int]$managed.WarningCount -eq 0 -and
    [string]$managed.WarningsJson -ceq '[]' -and
    [string]$managed.WarningsSHA256 -ceq $warningHash
if ($warnings.Count -ne 0) {
    $warningStateOK = $false
}
$managedOK = $managed.Status -eq 'PASS' -and
    $managed.CompileExit -eq 0 -and $managed.RunExit -eq 0 -and
    $managed.ManagedTesterCompiles -eq 1 -and $managed.ManagedTesterRuns -eq 1 -and
    $managedCompile.CompileCount -eq 1 -and $managedCompile.ExitCode -eq 0 -and
    $managedRun.RunCount -eq 1 -and $managedRun.ExitCode -eq 0 -and
    $managedRun.WindowStyle -eq 'Hidden' -and
    $managedRecord.schema -eq 1 -and $managedRecord.status -eq 'PASS' -and
    $managedRecord.export.newManagedCalls -eq 1 -and $managedRecord.export.exportCalls -eq 1 -and
    $managedRecord.export.admissionCalls -eq 1 -and $managedRecord.export.verifyCalls -eq 1 -and
    $managedRecord.export.startCalls -eq 1 -and $managedRecord.export.waitCalls -eq 1 -and
    $managedRecord.export.pageIndex -eq 1 -and $managedRecord.export.pageID -ceq 'second' -and
    $managedRecord.sourceBytes -eq [int]$ctx.ExpectedSmokeSourceBytes -and
    $managedRecord.sourceSHA256Before -ceq $managedRecord.sourceSHA256After -and
    $managedRecord.candidateExeSHA256Before -ceq $managedRecord.candidateExeSHA256After -and
    $managedRecord.payloadBytes -eq [int]$ctx.ExpectedSmokePayloadBytes -and
    $managedRecord.payloadSHA256 -ceq [string]$ctx.ExpectedSmokePayloadSHA256 -and
    $managedRecord.process.startObserved -and $managedRecord.process.duplicateHandleObserved -and
    $managedRecord.process.imagePathObserved -and $managedRecord.process.creationObserved -and
    $managedRecord.process.waitObserved -and $managedRecord.process.exitCode -eq 0 -and
    $managedRecord.process.primaryClosed -and -not $managedRecord.process.fallbackCloseAttempted -and
    $managedRecord.runner.closeCalled -and
    $rawWarningState.Kind -in @('Null', 'Array') -and
    $warningStateOK -and
    $warnings.Count -eq [int]$ctx.ExpectedSmokeWarningCount -and
    $managed.CandidateFileCount -eq 25 -and $managed.BundlePayloadFileCount -eq 25 -and
    $managed.BundleAfterFileCount -eq 26
Check 'one actual managed draw.io second-page request used the reviewed fixture from a complete bundle copy and joined one exact worker lifecycle' $managedOK
Check 'the unchanged managed fixture produced exactly 516 source bytes, the 598-byte SVG oracle and zero warnings' (
    $managed.SourceBytes -eq 516 -and $managed.PayloadBytes -eq 598 -and
    $managed.PayloadSHA256 -ceq
        '3f990a4aac3209bafad94f44026bf14f3cec27ea44bdbd3169555947936472bf' -and
    $warnings.Count -eq 0 -and $warningStateOK
) "source=$($managed.SourceBytes), payload=$($managed.PayloadBytes), warnings=$($warnings.Count), raw=$($rawWarningState.Kind), normalized=$($managedWarningState.Kind), sha256=$warningHash"

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
    $hostResult.SessionObserved -and $hostSessionOK -and $hostResult.SourcePreserved -and
    $hostResult.InstalledPluginPreserved -and
    $hostResult.BundlePreserved -and $hostResult.AppStdoutBytes -eq 0 -and $hostResult.AppStderrBytes -eq 0 -and
    $hostResult.JobProcessesAfterAssignment -eq 1 -and
    $hostResult.JobTotalProcessesAfterExit -eq 1 -and
    $hostResult.RequestedWorkerOperations -eq 0 -and $configOK -and $hostBundleOK
Check 'one separate byte-identical copied .21 host started hidden, requested no worker, and exited normally after one owned WM_CLOSE' $hostOK
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
    SourceInputCount = $sourceAfter.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceAfter
    SourceGateFileCount = $sourceGatesAfter.Count
    BenchmarkFixtureCount = $benchmarkFixturesAfter.Count
    SmokeFixture = $smokeFixture
    ManagedTesterCompiles = 1
    ManagedTesterRuns = 1
    ManagedWorkerNewRequests = $managedRecord.export.newManagedCalls
    ManagedWorkerExports = $managedRecord.export.exportCalls
    ManagedWorkerPID = $managedRecord.process.pid
    ManagedWorkerHandle = $managedRecord.process.duplicateHandle
    ManagedWorkerCreationUTC = $managedRecord.process.creationUTC
    ManagedPayloadSHA256 = $managedRecord.payloadSHA256
    ManagedWarningsPropertyKind = $rawWarningState.Kind
    ManagedWarningsCount = $warnings.Count
    ManagedWarningsSHA256 = $managed.WarningsSHA256
    HiddenAppStarts = $hostResult.AppStarts
    HiddenAppPID = $hostResult.App.Id
    HiddenAppHandle = $hostResult.App.HeldHandle
    HiddenHostConfigRoot = $configRoot
    HiddenHostConfigFiles = @($hostResult.ConfigFilesAfterClose)
    Scope = 'Exact local .21 package selection only: one package-script invocation/five Windows GUI builds, actual 25-file candidate and schema1 manifest, four fixed engine manifests, immutable complete .20 package/evidence and accepted warning/retention fixture protection, one enabled managed draw.io second-page request, and one separate hidden custom-suffix host startup/WM_CLOSE. The managed request uses the unchanged 516-byte fixture and 598-byte literal SVG oracle with zero warnings; inherited source tests establish broader behavior. No physical IME/UI, dirty Save, benchmark replay, process-memory, CPU or packaged interactive SVG claim.'
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
    ManagedTesterCompiles = 1
    ManagedTesterRuns = 1
    ManagedWorkerExports = $managedRecord.export.exportCalls
    ManagedSourceBytes = $managedRecord.sourceBytes
    ManagedPayloadBytes = $managedRecord.payloadBytes
    ManagedPayloadSHA256 = $managedRecord.payloadSHA256
    ManagedWarningsSHA256 = $managed.WarningsSHA256
    HiddenAppStarts = $hostResult.AppStarts
    HostWorkerOperations = $hostResult.RequestedWorkerOperations
    SourceGateFileCount = $sourceGatesAfter.Count
    BenchmarkFixtureCount = $benchmarkFixturesAfter.Count
    Scope = $validator.Scope
}
$aggregatePath = Join-Path $evidenceRoot 'aggregate.json'
$records = @(
    Get-ChildItem -LiteralPath $evidenceRoot -File |
        Where-Object { $_.FullName -cne $aggregatePath } |
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
