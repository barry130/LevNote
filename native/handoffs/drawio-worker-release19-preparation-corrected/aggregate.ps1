$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

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
Check 'release manifest is schema1 for .19 and all 24 non-manifest payload entries match actual size and SHA-256' (
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
Check 'four fixed catalog engine manifests match the .19 version and each worker payload identity' (
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

$protectedBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-before.json'))
$protectedAfter = @(Get-DirectoryInventory $ctx.protectedReleaseDirectory)
$protectedEvidenceBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-evidence-before.json'))
$protectedEvidenceAfter = @(Get-DirectoryInventory $ctx.protectedEvidenceDirectory)
$protectedOK = $true
try {
    Assert-InventoriesEqual $protectedBefore $protectedAfter 'Protected .18 package'
    Assert-InventoriesEqual $protectedEvidenceBefore $protectedEvidenceAfter 'Protected .18 direct evidence'
} catch {
    $protectedOK = $false
}
Check 'accepted .18 package and direct delivery evidence remain byte/time identical to their complete accepted inventories' (
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

$stage2Before = @(Read-JsonFile (Join-Path $evidenceRoot 'final-stage2-files.json'))
$stage2Bad = @(
    foreach ($entry in $stage2Before) {
        $actual = Get-FileFingerprint $entry.Path
        if ($actual.Bytes -ne [Int64]$entry.Bytes -or $actual.SHA256 -cne [string]$entry.SHA256 -or
            $actual.LastWriteUtcTicks -ne [Int64]$entry.LastWriteUtcTicks) {
            $entry.RelativePath
        }
    }
)
Check 'all final frozen Stage2 source/test files still match the activation-time accepted manifest' (
    $stage2Before.Count -eq @($ctx.FinalStage2Files).Count -and $stage2Bad.Count -eq 0
) "files=$($stage2Before.Count)"

$acceptedDirectoriesBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'accepted-directories-before.json'))
$acceptedDirsOK = $true
foreach ($record in $acceptedDirectoriesBefore) {
    $actual = @(Get-DirectoryInventory ([string]$record.Directory))
    try {
        Assert-InventoriesEqual @($record.Inventory) $actual ('Accepted evidence directory ' + $record.Directory)
    } catch {
        $acceptedDirsOK = $false
    }
}
$acceptedGatesBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'accepted-gates-before.json'))
$acceptedGatesAfter = @(
    foreach ($path in @($ctx.acceptedGatePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$acceptedGatesOK = $true
try {
    Assert-InventoriesEqual $acceptedGatesBefore $acceptedGatesAfter 'Accepted gate files'
} catch {
    $acceptedGatesOK = $false
}
Check 'accepted Stage1, correction, view, smoke and portable evidence/gate records remain unchanged' (
    $acceptedDirsOK -and $acceptedGatesOK -and
    $acceptedDirectoriesBefore.Count -eq @($ctx.AcceptedEvidenceDirectories).Count
) "directories=$($acceptedDirectoriesBefore.Count), gates=$($acceptedGatesAfter.Count)"

$warnings = @($managed.Warnings)
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
    $managedRecord.sourceSHA256Before -ceq $managedRecord.sourceSHA256After -and
    $managedRecord.candidateExeSHA256Before -ceq $managedRecord.candidateExeSHA256After -and
    $managedRecord.process.startObserved -and $managedRecord.process.duplicateHandleObserved -and
    $managedRecord.process.imagePathObserved -and $managedRecord.process.creationObserved -and
    $managedRecord.process.waitObserved -and $managedRecord.process.exitCode -eq 0 -and
    $managedRecord.process.primaryClosed -and -not $managedRecord.process.fallbackCloseAttempted -and
    $managedRecord.runner.closeCalled -and
    $warnings.Count -eq 2 -and
    $managed.CandidateFileCount -eq 25 -and $managed.BundlePayloadFileCount -eq 25 -and
    $managed.BundleAfterFileCount -eq 26
Check 'one actual managed draw.io second-page request used the reviewed fixture from a complete bundle copy and joined one exact worker lifecycle' $managedOK

$hostTabs = @($hostSession.tabs)
$hostSourcePath = Join-Path $evidenceRoot 'host-smoke/workspace/package-smoke.md'
$hostTab = @($hostTabs | Where-Object { $_.path -eq $hostSourcePath })
$hostSessionOK = $hostTabs.Count -eq 1 -and $hostTab.Count -eq 1 -and
    -not ($hostTab[0].PSObject.Properties.Name -contains 'draft') -and
    -not [bool]$hostResult.SessionHasTerminalNewline
$configRoot = [IO.Path]::GetFullPath([string]$hostResult.RunOwnedConfigRoot)
$configPrefix = $configRoot.TrimEnd('\') + '\'
$configOK = $hostResult.ConfigRootRetained -and
    [IO.Path]::GetPathRoot($configRoot) -eq 'C:\' -and
    [IO.Path]::GetPathRoot([string]$hostResult.ExecutablePath) -eq 'F:\' -and
    @($hostResult.ConfigFilesAfterClose).Count -ge 1 -and
    @($hostResult.ConfigFilesAfterClose | Where-Object {
        -not ([IO.Path]::GetFullPath([string]$_.Path)).StartsWith($configPrefix, [StringComparison]::OrdinalIgnoreCase)
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
    $hostResult.BundlePreserved -and $hostResult.AppStdoutBytes -eq 0 -and $hostResult.AppStderrBytes -eq 0 -and
    $hostResult.RequestedWorkerOperations -eq 0 -and $configOK -and $hostBundleOK
Check 'one separate byte-identical copied .19 host started hidden and exited normally after one owned WM_CLOSE without timeout or force cleanup' $hostOK

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
    FinalStage2FileCount = $stage2Before.Count
    AcceptedEvidenceDirectoryCount = $acceptedDirectoriesBefore.Count
    AcceptedGateFileCount = $acceptedGatesAfter.Count
    ManagedTesterCompiles = 1
    ManagedTesterRuns = 1
    ManagedWorkerNewRequests = $managedRecord.export.newManagedCalls
    ManagedWorkerExports = $managedRecord.export.exportCalls
    ManagedWorkerPID = $managedRecord.process.pid
    ManagedWorkerHandle = $managedRecord.process.duplicateHandle
    ManagedWorkerCreationUTC = $managedRecord.process.creationUTC
    ManagedPayloadSHA256 = $managedRecord.payloadSHA256
    ManagedWarningsSHA256 = $managed.WarningsSHA256
    HiddenAppStarts = $hostResult.AppStarts
    HiddenAppPID = $hostResult.App.PID
    HiddenAppHandle = $hostResult.App.HeldHandle
    HiddenHostConfigRoot = $configRoot
    HiddenHostConfigFiles = @($hostResult.ConfigFilesAfterClose)
    Scope = 'Exact local .19 package selection only: one package-script invocation/five Windows GUI builds, actual 25-file candidate and schema1 manifest, four fixed engine manifests, complete .18 and accepted-gate protection, one enabled managed draw.io second-page request, and one separate hidden host startup/WM_CLOSE. The managed smoke establishes one fixed installed fourth-worker request and joined lifecycle; inherited tests establish broader behavior. No physical IME/UI, dirty Save, process-memory, CPU or packaged interactive SVG claim.'
}
$validatorPath = Join-Path $evidenceRoot 'validator.json'
Write-JsonFile $validatorPath $validator 16
Write-JsonFile (Join-Path $evidenceRoot 'protected-after.json') $protectedAfter
Write-JsonFile (Join-Path $evidenceRoot 'protected-evidence-after.json') $protectedEvidenceAfter

$recordPaths = [Collections.Generic.List[string]]::new()
foreach ($relative in @(
    'context.json',
    'commands.json',
    'delivery-lib.ps1',
    'source-scope.ps1',
    'initialize-context.ps1',
    'invoke-stage.ps1',
    'run-stage.ps1',
    'baseline.ps1',
    'baseline-result.json',
    'inputs-before.json',
    'inputs-after.json',
    'protected-before.json',
    'protected-after.json',
    'protected-evidence-before.json',
    'protected-evidence-after.json',
    'accepted-directories-before.json',
    'accepted-gates-before.json',
    'final-stage2-input.json',
    'final-stage2-files.json',
    'run-package.ps1',
    'package-invoke.ps1',
    'package-command.json',
    'package.stdout.log',
    'package.stderr.log',
    'package-result.json',
    'package/package-result.json',
    'run-managed-smoke.ps1',
    'managed-smoke/compile.ps1',
    'managed-smoke/compile-invocation.json',
    'managed-smoke/managed-tester-compile.stdout.log',
    'managed-smoke/managed-tester-compile.stderr.log',
    'managed-smoke/run-invocation.json',
    'managed-smoke/run.stdout.log',
    'managed-smoke/run.stderr.log',
    'managed-smoke/record.json',
    'managed-smoke/result.json',
    'run-host-smoke.ps1',
    'host-smoke/result.json',
    'host-smoke/session-after-close.json',
    'host-smoke/app.stdout.log',
    'host-smoke/app.stderr.log',
    'aggregate.ps1',
    'validator.json'
)) {
    $recordPaths.Add($relative)
}
foreach ($build in $buildNames) {
    $recordPaths.Add("package/$build-build.json")
    $recordPaths.Add("package/$build-build.ps1")
    $recordPaths.Add("package/$build.stdout.log")
    $recordPaths.Add("package/$build.stderr.log")
}
foreach ($name in @('baseline', 'package', 'managed-smoke', 'host-smoke')) {
    foreach ($prefix in @('outer-', 'stage-')) {
        $recordPaths.Add($prefix + $name + '.stdout.log')
        $recordPaths.Add($prefix + $name + '.stderr.log')
        $recordPaths.Add($prefix + $name + '-invocation.json')
    }
}
$records = @(
    foreach ($relative in $recordPaths) {
        $path = Join-Path $evidenceRoot $relative
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Expected evidence record is absent: $relative"
        }
        Get-FileFingerprint $path
    }
)
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
    HiddenAppStarts = $hostResult.AppStarts
    HostWorkerOperations = $hostResult.RequestedWorkerOperations
    Stage2FileCount = $stage2Before.Count
    Evidence = $records
    Scope = $validator.Scope
}
$aggregatePath = Join-Path $evidenceRoot 'aggregate.json'
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
