param(
    [ValidateRange(1, 2)]
    [int]$AggregateAttempt = 1
)

$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
. (Join-Path $evidenceRoot 'source-scope.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (Test-Path -LiteralPath (Join-Path $evidenceRoot 'aggregate.json') -PathType Leaf) {
    throw 'Aggregate already completed in this evidence directory.'
}
$attemptRoot = Join-Path $evidenceRoot 'aggregate-attempts'
$attemptDirectory = Join-Path $attemptRoot ('attempt-{0}' -f $AggregateAttempt)
$aggregateCaptureDirectory = Join-Path (Split-Path $evidenceRoot -Parent) (
    (Split-Path $evidenceRoot -Leaf) + '-aggregate-capture-attempt-' + $AggregateAttempt)
if (-not (Test-Path -LiteralPath $aggregateCaptureDirectory -PathType Container)) {
    throw "Aggregate must run through run-stage.ps1 so captured output has a distinct attempt directory: $aggregateCaptureDirectory"
}
if (Test-Path -LiteralPath $attemptDirectory) {
    throw "Aggregate attempt already exists: $attemptDirectory"
}
if ($AggregateAttempt -eq 1) {
    if ((Test-Path -LiteralPath $attemptRoot -PathType Container) -and
        @(Get-ChildItem -LiteralPath $attemptRoot -Force).Count -ne 0) {
        throw 'Aggregate attempt 1 cannot be reused; choose one metadata-only retry attempt 2 or stop.'
    }
} else {
    $priorAttemptDirectory = Join-Path $attemptRoot 'attempt-1'
    $priorAggregatePath = Join-Path $priorAttemptDirectory 'aggregate.json'
    if (-not (Test-Path -LiteralPath $priorAttemptDirectory -PathType Container)) {
        throw 'Aggregate attempt 2 requires a preserved failed attempt 1.'
    }
    if (Test-Path -LiteralPath $priorAggregatePath -PathType Leaf) {
        $priorAggregate = Read-JsonFile $priorAggregatePath
        if ([bool]$priorAggregate.Pass) {
            throw 'Aggregate attempt 1 passed; the metadata-only retry is not allowed.'
        }
    } else {
        $priorCaptureDirectory = Join-Path (Split-Path $evidenceRoot -Parent) (
            (Split-Path $evidenceRoot -Leaf) + '-aggregate-capture-attempt-1')
        $priorInvocationPaths = @(
            (Join-Path $priorCaptureDirectory 'outer-aggregate-attempt-1-invocation.json')
            (Join-Path $priorCaptureDirectory 'stage-aggregate-attempt-1-invocation.json')
        )
        $recordedFailedInvocation = $false
        foreach ($path in $priorInvocationPaths) {
            if (Test-Path -LiteralPath $path -PathType Leaf) {
                $recorded = Read-JsonFile $path
                if ([int]$recorded.ExitCode -ne 0) {
                    $recordedFailedInvocation = $true
                    break
                }
            }
        }
        if (-not $recordedFailedInvocation) {
            throw 'Attempt 2 requires a preserved failed invocation for attempt 1.'
        }
    }
}
if (-not (Test-Path -LiteralPath $attemptRoot -PathType Container)) {
    New-Item -ItemType Directory -Path $attemptRoot -ErrorAction Stop | Out-Null
}
New-Item -ItemType Directory -Path $attemptDirectory -ErrorAction Stop | Out-Null
$checks = [Collections.Generic.List[object]]::new()
function Check([string]$Name, [bool]$Pass, [string]$Detail = '') {
    $checks.Add([ordered]@{ Name = $Name; Pass = $Pass; Detail = $Detail })
}

$baseline = Read-JsonFile (Join-Path $evidenceRoot 'baseline-result.json')
$packageCommand = Read-JsonFile (Join-Path $evidenceRoot 'package-command.json')
$packageResult = Read-JsonFile (Join-Path $evidenceRoot 'package-result.json')
$packageInternal = Read-JsonFile (Join-Path $evidenceRoot 'package/package-result.json')
$hostSummary = Read-JsonFile (Join-Path $evidenceRoot 'host-smoke/result.json')
$run1 = Read-JsonFile (Join-Path $evidenceRoot 'host-smoke/run-1-mixed-cli-open/result.json')
$run2 = Read-JsonFile (Join-Path $evidenceRoot 'host-smoke/run-2-path-only-restore/result.json')
Assert-RequiredProperties $baseline @(
    'CandidateAbsent', 'SourceInputCount') 'Baseline result'
Assert-RequiredProperties $packageCommand @(
    'InvocationCount', 'ExitCode', 'ExpectedGuiBuilds',
    'PackageArguments') 'Package command'
Assert-RequiredProperties $packageCommand.PackageArguments @(
    'OutputDirectory', 'EvidenceDirectory', 'Version') 'Package arguments'
Assert-RequiredProperties $packageResult @(
    'PackageInvocationCount', 'GuiBuildCount', 'CandidateFileCount',
    'CandidateInventoryIdentity', 'ExpectedCandidateFileCount',
    'ExpectedCandidateFileSetPass', 'ExpectedReleaseManifestPayloadEntries',
    'EngineChecks', 'PEFacts', 'SourceCopyCount', 'SourceCopies',
    'LocalAuthoredLinksChecked', 'PortableReadmeSource',
    'PortableReadmePackage', 'PortableReadmeMarkerCount',
    'PortableReadmeInlineSchema3JSONCount', 'PortableReadmeRoutes'
) 'Package result'
Assert-RequiredProperties $packageInternal @(
    'ReleaseManifestSHA256', 'HostSHA256', 'LocalAuthoredLinksChecked'
) 'Internal package result'
Assert-RequiredProperties $hostSummary @(
    'RunCount', 'SameCopiedBundle', 'SameConfigRoot',
    'DistinctSequentialRunLifetimes', 'Run1CompletedBeforeRun2Started',
    'BothRunsSettledAliveForBoundedPeriod',
    'BothRunsRecordedOptionalPreCloseSessionObservation',
    'BothRunsPublishedFreshSessionAfterStart',
    'BothRunsClosedOnlyAfterSettling', 'CandidatePreserved',
    'CopiedBundlePreserved', 'FixturesPreserved',
    'InstalledPluginPreserved'
) 'Host summary'
$runContract = @(
    'Status', 'Kind', 'StartedAtUtc', 'FinishedAtUtc',
    'AppStarts', 'StartAttemptCount', 'StartedHidden',
    'LaunchMethod', 'AppArguments', 'App', 'WindowFound', 'WindowVisible',
    'SessionSettlingMillisecondsRequested', 'SessionSettlingMillisecondsActual',
    'SessionSettlingAliveChecks', 'SessionSettlingCompleted',
    'ProcessAliveAfterSettling', 'CloseIssuedAfterSettling',
    'PreCloseSessionObservationAttempted',
    'PreCloseSessionStrictShapeObserved', 'PreCloseSessionObservationError',
    'GracefulWMClosePosted',
    'WMClosePostCount', 'GracefulCloseTimedOut', 'ForcedAppCleanup',
    'AppExitCode', 'OwnedAppStillLive', 'ProcessDisposed', 'JobAfterAssignment',
    'JobAfterExit', 'RequestedWorkerOperations', 'SessionObserved',
    'SessionBeforeRun', 'SessionBeforeRunRecordPath',
    'SessionBeforeRunCopyPath', 'SessionFactsBeforeClose',
    'SessionBeforeCloseFingerprint', 'SessionFactsAfterClose',
    'SessionAfterCloseFingerprint', 'FreshSessionPublication',
    'ClosePublicationAfterProcessStart', 'SessionCopyPath', 'AppStdoutBytes',
    'AppStderrBytes', 'ExecutedByteIdenticalBundle',
    'RunOwnedConfigRoot', 'ExecutablePath'
)
foreach ($run in @($run1, $run2)) {
    Assert-RequiredProperties $run $runContract "Host run $($run.Name)"
    Assert-RequiredProperties $run.App @(
        'Id', 'Path', 'StartTimeUtc', 'HeldHandle',
        'PathMethod') "Host run $($run.Name) process"
    foreach ($jobName in @('JobAfterAssignment', 'JobAfterExit')) {
        $job = $run.$jobName
        Assert-RequiredProperties $job @(
            'TotalProcesses', 'ActiveProcesses') "Host run $($run.Name) $jobName"
    }
    Assert-RequiredProperties $run.SessionBeforeRun @(
        'Present', 'Path', 'Bytes', 'SHA256', 'LastWriteUtcTicks'
    ) "Host run $($run.Name) pre-run session"
    if ($null -ne $run.SessionFactsBeforeClose) {
        Assert-RequiredProperties $run.SessionFactsBeforeClose @(
            'ActiveIndex', 'TabCount') "Host run $($run.Name) pre-close session"
    }
    if ($null -ne $run.SessionBeforeCloseFingerprint) {
        Assert-RequiredProperties $run.SessionBeforeCloseFingerprint @(
            'Present', 'Path', 'Bytes', 'SHA256', 'LastWriteUtcTicks'
        ) "Host run $($run.Name) pre-close session fingerprint"
    }
    Assert-RequiredProperties $run.SessionFactsAfterClose @(
        'Path', 'SessionBytes', 'SessionSHA256', 'ActiveIndex',
        'TabCount') "Host run $($run.Name) post-close session"
    Assert-RequiredProperties $run.SessionAfterCloseFingerprint @(
        'Present', 'Path', 'Bytes', 'SHA256', 'LastWriteUtcTicks'
    ) "Host run $($run.Name) post-close session fingerprint"
}
$run1Session = Get-ReleaseSessionFacts `
    (Join-Path $evidenceRoot 'host-smoke/run-1-mixed-cli-open/session-after-close.json') `
    (Join-Path $evidenceRoot 'host-smoke/workspace/route-gate.diagramnote') `
    (Join-Path $evidenceRoot 'host-smoke/workspace/image-fixture.png')
$run2Session = Get-ReleaseSessionFacts `
    (Join-Path $evidenceRoot 'host-smoke/run-2-path-only-restore/session-after-close.json') `
    (Join-Path $evidenceRoot 'host-smoke/workspace/route-gate.diagramnote') `
    (Join-Path $evidenceRoot 'host-smoke/workspace/image-fixture.png')

$buildNames = @('worker-markdown', 'worker-mindmap', 'worker-mindmap-preview', 'worker-drawio', 'host')
$builds = @(
    foreach ($name in $buildNames) {
        Read-JsonFile (Join-Path $evidenceRoot ("package/{0}-build.json" -f $name))
    }
)
$buildBad = @($builds | Where-Object {
    $_.ExitCode -ne 0 -or $_.StdoutBytes -ne 0 -or $_.StderrBytes -ne 0 -or
    (Get-Item -LiteralPath $_.StdoutPath).Length -ne 0 -or
    (Get-Item -LiteralPath $_.StderrPath).Length -ne 0
})
Check 'one package-script invocation made five Windows GUI builds with exit 0 and empty streams' (
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
. (Join-Path $ctx.nativeRoot 'scripts/portable-package-lib.ps1')
$portableCopies = @(Get-PortableAssetCopies $ctx.nativeRoot)
$expectedCandidateFiles = [Collections.Generic.HashSet[string]]::new(
    [StringComparer]::Ordinal)
foreach ($name in @(
        'levnote-native.exe',
        'release-manifest.json',
        'README.md',
        'LICENSE',
        'THIRD_PARTY_NOTICES.md'
    )) {
    $expectedCandidateFiles.Add($name) | Out-Null
}
foreach ($relativePath in @(
        'engines/markdown-export/levnote-markdown-export.exe',
        'engines/markdown-export/manifest.json',
        'engines/mindmap-svg/levnote-mindmap-svg.exe',
        'engines/mindmap-svg/manifest.json',
        'engines/mindmap-preview/levnote-mindmap-preview.exe',
        'engines/mindmap-preview/manifest.json',
        'engines/drawio-svg/levnote-drawio-svg.exe',
        'engines/drawio-svg/manifest.json'
    )) {
    $expectedCandidateFiles.Add($relativePath) | Out-Null
}
foreach ($copy in $portableCopies) {
    $expectedCandidateFiles.Add([string]$copy.Relative) | Out-Null
}
$fixedArtifactCount = $expectedCandidateFiles.Count
Check 'actual candidate count is derived from host, four engine pairs, and actual portable SourceCopies' (
    $candidate.Count -eq $fixedArtifactCount -and
    $candidate.Count -eq [int]$packageResult.CandidateFileCount -and
    [int]$packageResult.ExpectedCandidateFileCount -eq $fixedArtifactCount -and
    $packageResult.ExpectedCandidateFileSetPass -and
    $candidateIdentity -ceq [string]$packageResult.CandidateInventoryIdentity -and
    $copyMatches -and $packageCopy.Count -eq $candidate.Count
) "actual=$($candidate.Count), derived=$fixedArtifactCount, bytes=$(Get-InventoryTotalBytes $candidate)"

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
Check 'release manifest is schema1 for .23 and every non-manifest payload matches actual bytes and SHA-256' (
    $release.schema -eq 1 -and $release.version -ceq $ctx.version -and
    $release.goos -ceq 'windows' -and $release.goarch -ceq 'amd64' -and
    $release.goamd64 -ceq 'v1' -and
    $releaseEntries.Count -eq $candidate.Count - 1 -and
    $releaseEntries.Count -eq [int]$packageResult.ExpectedReleaseManifestPayloadEntries -and
    @($release.excludes).Count -eq 1 -and
    @($release.excludes)[0] -ceq 'release-manifest.json' -and
    $releaseBad.Count -eq 0 -and
    $packageInternal.ReleaseManifestSHA256 -ceq $manifestFingerprint.SHA256 -and
    $packageInternal.HostSHA256 -ceq $hostFingerprint.SHA256
) "entries=$($releaseEntries.Count), candidate=$($candidate.Count)"

$fixedEngineNames = @(
    'levnote.markdown-export',
    'levnote.mindmap-svg',
    'levnote.mindmap-preview',
    'levnote.drawio-svg'
)
$engineNames = @($packageResult.EngineChecks | ForEach-Object { $_.Engine } | Sort-Object)
Check 'four fixed engine manifests match the .23 version and each worker payload identity' (
    $packageResult.EngineChecks.Count -eq 4 -and
    @(Compare-Object ($fixedEngineNames | Sort-Object) $engineNames).Count -eq 0 -and
    @($packageResult.EngineChecks | Where-Object { -not $_.Pass }).Count -eq 0
)

$peFacts = @($packageResult.PEFacts)
Check 'five executable payloads are Windows amd64 PE32+ GUI images' (
    $peFacts.Count -eq 5 -and
    @($peFacts | Where-Object { -not $_.Pass }).Count -eq 0
)

$bitmapAsset = @($packageResult.SourceCopies | Where-Object {
    $_.RelativePath -ceq 'attribution/mygo-v0.8.3-bitmap-dispose.patch'
})
$startupAsset = @($packageResult.SourceCopies | Where-Object {
    $_.RelativePath -ceq 'attribution/mygo-v0.8.3-no-startup-cleanup.patch'
})
$noticeText = [IO.File]::ReadAllText(
    (Join-Path $ctx.candidateDirectory 'THIRD_PARTY_NOTICES.md'), [Text.Encoding]::UTF8)
Check 'actual portable assets, local links, both dependency patches, and bitmap attribution are shipped' (
    $packageResult.SourceCopyCount -eq $portableCopies.Count -and
    @($packageResult.SourceCopies | Where-Object { -not $_.Pass }).Count -eq 0 -and
    $packageResult.LocalAuthoredLinksChecked -and
    $packageInternal.LocalAuthoredLinksChecked -and
    $bitmapAsset.Count -eq 1 -and $bitmapAsset[0].Shipped.Bytes -eq 47176 -and
    $bitmapAsset[0].Shipped.SHA256 -ceq
        '995D5505C3B5B1BD869547DF3F1AD5954C2DD57F1C9F32E53554851B5DB0D4D1' -and
    $startupAsset.Count -eq 1 -and $startupAsset[0].Shipped.Bytes -eq 516 -and
    $startupAsset[0].Shipped.SHA256 -ceq
        '1DD7D35B0E6CFD66510F9CF0D82F703AD7656C740BC682CCF1D0C21E0F8B110A' -and
    $noticeText.Contains('attribution/mygo-v0.8.3-bitmap-dispose.patch', [StringComparison]::Ordinal) -and
    $noticeText.Contains('位图', [StringComparison]::Ordinal)
) "sourceCopies=$($packageResult.SourceCopyCount)"

$packageReadmePath = Join-Path $ctx.candidateDirectory 'README.md'
$packageReadmeText = [IO.File]::ReadAllText($packageReadmePath, [Text.Encoding]::UTF8)
$portableReadme = Get-FileFingerprint $ctx.portableSourceReadme
$packageReadme = Get-FileFingerprint $packageReadmePath
$portableReadmeBefore = [pscustomobject]$ctx.PortableSourceReadmeFingerprint
$imageReadmeContract = Get-ImageReadmeContract
$missingProse = @($imageReadmeContract.Required | Where-Object {
    -not $packageReadmeText.Contains($_, [StringComparison]::Ordinal)
})
$forbiddenProse = @($imageReadmeContract.Forbidden | Where-Object {
    $packageReadmeText.Contains($_, [StringComparison]::Ordinal)
})
Check 'portable README retains one marker, three schema3 routes, and the complete native image contract' (
    (Test-FingerprintEqual $portableReadmeBefore $portableReadme) -and
    $packageResult.PortableReadmeSource.Bytes -eq $portableReadme.Bytes -and
    $packageResult.PortableReadmeSource.SHA256 -ceq $portableReadme.SHA256 -and
    $packageResult.PortableReadmePackage.SHA256 -ceq $packageReadme.SHA256 -and
    $packageResult.PortableReadmeMarkerCount -eq 1 -and
    $packageResult.PortableReadmeInlineSchema3JSONCount -eq 3 -and
    (@($packageResult.PortableReadmeRoutes | Sort-Object) -join ',') -ceq
        (@('builtin.drawio', 'builtin.markdown', 'builtin.mindmap') -join ',') -and
    -not $packageReadmeText.Contains('{{VERSION}}') -and
    $packageReadmeText.Contains($ctx.version) -and
    $missingProse.Count -eq 0 -and $forbiddenProse.Count -eq 0
) "source=$($portableReadme.Bytes) bytes, missingProse=$($missingProse -join ','), forbiddenProse=$($forbiddenProse -join ',')"

$frozenAfter = Get-FrozenBitmapFacts $ctx.nativeRoot $ctx.FrozenBitmapFilesPath
$patchAfter = Get-PinnedPatchFacts $ctx.nativeRoot $frozenAfter
$baselinePatches = Read-JsonFile (Join-Path $evidenceRoot 'pinned-patches-before.json')
Check 'frozen bitmap lifecycle identity and pinned patch-manifest guarded files remain exact' (
    $frozenAfter.FileCount -eq 35 -and
    $frozenAfter.FrozenFiles.SHA256 -ceq
        '2B1D185CB3302427E692B0FE955CE2D0AAC0BAB8F559775B8E522B56BCA10E8A' -and
    $frozenAfter.ChangedOrAdded -eq 33 -and $frozenAfter.UnchangedReferences -eq 2 -and
    @($patchAfter.Entries).Count -eq 25 -and
    $patchAfter.Manifest.SHA256 -ceq
        '9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A' -and
    $patchAfter.BitmapPatch.SHA256 -ceq
        '995D5505C3B5B1BD869547DF3F1AD5954C2DD57F1C9F32E53554851B5DB0D4D1' -and
    $patchAfter.StartupPatch.SHA256 -ceq
        '1DD7D35B0E6CFD66510F9CF0D82F703AD7656C740BC682CCF1D0C21E0F8B110A' -and
    (Test-FingerprintEqual ([pscustomobject]$baselinePatches.Manifest) $patchAfter.Manifest) -and
    $patchAfter.GuardedFilesMatchFrozenList
)

$protectedBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-before.json'))
$protectedAfter = @(Get-DirectoryInventory $ctx.protectedReleaseDirectory)
$protectedEvidenceBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-evidence-before.json'))
$protectedEvidenceAfter = @(Get-DirectoryInventory $ctx.protectedEvidenceDirectory)
$protectedCaptureBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'protected-capture-before.json'))
$protectedCaptureAfter = @(Get-DirectoryInventory $ctx.protectedCaptureDirectory)
$protectedOK = $true
try {
    Assert-InventoriesEqual $protectedBefore $protectedAfter 'Protected .22 package'
    Assert-InventoriesEqual $protectedEvidenceBefore $protectedEvidenceAfter 'Protected .22 direct evidence'
    Assert-InventoriesEqual $protectedCaptureBefore $protectedCaptureAfter 'Protected .22 aggregate capture'
} catch {
    $protectedOK = $false
}
$closedAfter = Get-FileFingerprint (
    Join-Path $ctx.nativeRoot 'artifacts/acceptance/tab-workflow-release22-20261110-closed-inventory.json')
Check 'accepted .22 package, direct evidence, capture sibling, and closed inventory remain byte/time identical' (
    $protectedOK -and
    $protectedAfter.Count -eq 25 -and
    (Get-InventoryTotalBytes $protectedAfter) -eq 26863393 -and
    (Get-InventoryIdentity $protectedAfter) -ceq
        'BA43209FF7F6CFBEDD8287DBEA7B642008AFA839D133FECD8CC5C9D065FD9CF6' -and
    $protectedEvidenceAfter.Count -eq 134 -and
    (Get-InventoryTotalBytes $protectedEvidenceAfter) -eq 54340909 -and
    (Get-InventoryIdentity $protectedEvidenceAfter) -ceq
        'B862D886E30F00D4364972DC8521AA75EB2C266DF81B1250DF1F42A2D1B1CC5C' -and
    $protectedCaptureAfter.Count -eq 6 -and
    (Get-InventoryTotalBytes $protectedCaptureAfter) -eq 2684 -and
    (Get-InventoryIdentity $protectedCaptureAfter) -ceq
        'E6D61DCF9B02D9D32F3DC4957DA4747506E0A7E942DD62C55C91B9107011DFE3' -and
    $closedAfter.Bytes -eq 57245 -and
    $closedAfter.SHA256 -ceq
        'E61FB97AA24C5DAD6673F365D32EB77398C0D98EE32FE456A13F9B45A75373E2'
) "release=$($protectedAfter.Count), evidence=$($protectedEvidenceAfter.Count), capture=$($protectedCaptureAfter.Count)"

$sourceBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'inputs-before.json'))
$sourcePaths = @(Get-DeliverySourceInputPaths $ctx.repositoryRoot $ctx.nativeRoot)
$sourceAfter = @($sourcePaths | ForEach-Object { Get-FileFingerprint $_ })
$sourceOK = $true
try {
    Assert-InventoriesEqual $sourceBefore $sourceAfter 'Scoped source/build input'
} catch {
    $sourceOK = $false
}
$sourceGatesBefore = @(Read-JsonFile (Join-Path $evidenceRoot 'source-gates-before.json'))
$sourceGatesAfter = @(
    foreach ($path in @($ctx.RootSourceAcceptancePaths)) {
        Get-FileFingerprint ([string]$path)
    }
)
$sourceGatesOK = $true
try {
    Assert-InventoriesEqual $sourceGatesBefore $sourceGatesAfter 'Root image source acceptance'
} catch {
    $sourceGatesOK = $false
}
$architectureBefore = @(Read-JsonFile (
    Join-Path $evidenceRoot 'architecture-acceptances-before.json'))
foreach ($entry in $architectureBefore) {
    $entry.Actual = Get-FileFingerprint $entry.Path
}
$architectureOK = @($architectureBefore | Where-Object {
    -not (Test-FingerprintEqual ([pscustomobject]$_.Expected) $_.Actual)
}).Count -eq 0
Check 'scoped source/build inputs, frozen source gates, and composition/app acceptances remain unchanged' (
    $sourceOK -and
    $sourceAfter.Count -eq $baseline.SourceInputCount -and
    (Get-InventoryIdentity $sourceAfter) -ceq [string]$ctx.SourceInputIdentity -and
    $sourceGatesOK -and $sourceGatesAfter.Count -eq 8 -and $architectureOK
) "sourceFiles=$($sourceAfter.Count), gates=$($sourceGatesAfter.Count)"

function Test-HostRun([object]$Run, [object]$Session, [string]$Kind, [int]$ArgumentCount) {
    $bundleExe = Join-Path $evidenceRoot 'host-smoke/bundle/levnote-native.exe'
    $preRunCopyOK = if ([bool]$Run.SessionBeforeRun.Present) {
        -not [string]::IsNullOrWhiteSpace([string]$Run.SessionBeforeRunCopyPath) -and
            (Test-Path -LiteralPath ([string]$Run.SessionBeforeRunCopyPath) -PathType Leaf) -and
            (Test-FileStateContentEqual $Run.SessionBeforeRun (
                Get-OptionalFileState ([string]$Run.SessionBeforeRunCopyPath)))
    } else {
        [string]::IsNullOrWhiteSpace([string]$Run.SessionBeforeRunCopyPath)
    }
    $postCloseStateMatches = [bool]$Run.SessionAfterCloseFingerprint.Present -and
        [string]$Run.SessionAfterCloseFingerprint.Path -ceq
            [string]$Run.SessionFactsAfterClose.Path -and
        [Int64]$Run.SessionAfterCloseFingerprint.Bytes -eq
            [Int64]$Run.SessionFactsAfterClose.SessionBytes -and
        [string]$Run.SessionAfterCloseFingerprint.SHA256 -ceq
            [string]$Run.SessionFactsAfterClose.SessionSHA256
    return $Run.Status -eq 'PASS' -and $Run.Kind -eq $Kind -and
        $Run.AppStarts -eq 1 -and $Run.StartAttemptCount -eq 1 -and
        $Run.StartedHidden -and
        $Run.LaunchMethod -eq 'Start-Process -WindowStyle Hidden' -and
        @($Run.AppArguments).Count -eq $ArgumentCount -and
        [string]$Run.App.Path -eq $bundleExe -and
        $Run.App.PathMethod -eq 'QueryFullProcessImageName(exact held handle)' -and
        $Run.WindowFound -and -not [bool]$Run.WindowVisible -and
        $Run.SessionSettlingMillisecondsRequested -eq 5000 -and
        [double]$Run.SessionSettlingMillisecondsActual -ge 4900 -and
        $Run.SessionSettlingAliveChecks -gt 0 -and
        $Run.SessionSettlingCompleted -and $Run.ProcessAliveAfterSettling -and
        $Run.CloseIssuedAfterSettling -and
        $Run.PreCloseSessionObservationAttempted -and
        $Run.GracefulWMClosePosted -and $Run.WMClosePostCount -eq 1 -and
        -not $Run.GracefulCloseTimedOut -and -not $Run.ForcedAppCleanup -and
        $Run.AppExitCode -eq 0 -and -not $Run.OwnedAppStillLive -and
        $Run.ProcessDisposed -and
        $Run.JobAfterAssignment.TotalProcesses -eq 1 -and
        $Run.JobAfterAssignment.ActiveProcesses -eq 1 -and
        $Run.JobAfterExit.TotalProcesses -eq 1 -and
        $Run.JobAfterExit.ActiveProcesses -eq 0 -and
        $Run.RequestedWorkerOperations -eq 0 -and
        $Run.SessionObserved -and
        $preRunCopyOK -and
        $postCloseStateMatches -and
        $Run.FreshSessionPublication -and
        $Run.ClosePublicationAfterProcessStart -and
        $Run.SessionFactsAfterClose.ActiveIndex -eq 1 -and
        $Run.SessionFactsAfterClose.TabCount -eq 2 -and
        [string]$Run.SessionCopyPath -eq [string]$Session.Path -and
        $Session.ActiveIndex -eq 1 -and $Session.TabCount -eq 2 -and
        $Session.Text.Encoding -eq 'UTF-8' -and $Session.Text.BOM -and
        $Session.Text.PluginID -eq 'diagram-notes' -and
        $Session.Image.Record -eq 'binary' -and $Session.Image.Kind -eq 'png' -and
        $Session.Image.PluginID -eq 'image' -and
        $Session.Image.View -eq 'preview' -and
        $Session.Image.ForbiddenPayloadAbsent -and
        $Run.AppStdoutBytes -eq 0 -and $Run.AppStderrBytes -eq 0
}

$workspaceMixed = Join-Path $evidenceRoot 'host-smoke/workspace/route-gate.diagramnote'
$workspaceImage = Join-Path $evidenceRoot 'host-smoke/workspace/image-fixture.png'
$run1Paths = @($run1.AppArguments)
$run1OK = Test-HostRun $run1 $run1Session 'mixed-cli-open' 2
if ($run1OK) {
    $run1OK = $run1Paths[0] -ceq $workspaceMixed -and
        $run1Paths[1] -ceq $workspaceImage -and
        -not [bool]$run1.SessionBeforeRun.Present
}
Check 'run 1 opened mixed UTF-8/BOM text then PNG in one hidden exact host and closed normally' $run1OK
$run2OK = Test-HostRun $run2 $run2Session 'path-only-session-restore' 0
if ($run2OK) {
    $run2OK = [bool]$run2.SessionBeforeRun.Present -and
        (Test-FileStateContentEqual $run2.SessionBeforeRun $run1.SessionAfterCloseFingerprint) -and
        -not (Test-FileStateContentEqual $run2.SessionBeforeRun $run2.SessionAfterCloseFingerprint)
}
Check 'run 2 restored the same ordered mixed paths and active PNG from session with no CLI arguments' $run2OK
Check 'both runs used one byte-identical bundle and the same isolated C: config, with one process and zero workers each' (
    [string]$run1.ExecutedByteIdenticalBundle -ceq [string]$run2.ExecutedByteIdenticalBundle -and
    [string]$run1.RunOwnedConfigRoot -ceq [string]$run2.RunOwnedConfigRoot -and
    $hostSummary.DistinctSequentialRunLifetimes -and
    $hostSummary.Run1CompletedBeforeRun2Started -and
    $hostSummary.BothRunsSettledAliveForBoundedPeriod -and
    $hostSummary.BothRunsRecordedOptionalPreCloseSessionObservation -and
    $hostSummary.BothRunsPublishedFreshSessionAfterStart -and
    $hostSummary.BothRunsClosedOnlyAfterSettling -and
    [IO.Path]::GetPathRoot([string]$run1.ExecutablePath) -eq 'F:\' -and
    [IO.Path]::GetPathRoot([string]$run1.RunOwnedConfigRoot) -eq 'C:\' -and
    $hostSummary.RunCount -eq 2 -and $hostSummary.SameCopiedBundle -and
    $hostSummary.SameConfigRoot -and $hostSummary.CandidatePreserved -and
    $hostSummary.CopiedBundlePreserved -and $hostSummary.FixturesPreserved -and
    $hostSummary.InstalledPluginPreserved
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
    FrozenBitmapFileCount = $frozenAfter.FileCount
    FrozenBitmapIdentity = $frozenAfter.FrozenFiles.SHA256
    PatchManifestEntryCount = @($patchAfter.Entries).Count
    ProtectedReleaseFileCount = $protectedAfter.Count
    ProtectedReleaseTotalBytes = Get-InventoryTotalBytes $protectedAfter
    ProtectedReleaseInventoryIdentity = Get-InventoryIdentity $protectedAfter
    ProtectedEvidenceFileCount = $protectedEvidenceAfter.Count
    ProtectedEvidenceTotalBytes = Get-InventoryTotalBytes $protectedEvidenceAfter
    ProtectedEvidenceInventoryIdentity = Get-InventoryIdentity $protectedEvidenceAfter
    ProtectedCaptureFileCount = $protectedCaptureAfter.Count
    ProtectedCaptureTotalBytes = Get-InventoryTotalBytes $protectedCaptureAfter
    ProtectedCaptureInventoryIdentity = Get-InventoryIdentity $protectedCaptureAfter
    SourceInputCount = $sourceAfter.Count
    SourceInputTotalBytes = Get-InventoryTotalBytes $sourceAfter
    SourceInputIdentity = Get-InventoryIdentity $sourceAfter
    RootSourceAcceptanceFileCount = $sourceGatesAfter.Count
    HiddenAppStarts = $run1.AppStarts + $run2.AppStarts
    HiddenAppPIDs = @($run1.App.Id, $run2.App.Id)
    HiddenAppHandles = @($run1.App.HeldHandle, $run2.App.HeldHandle)
    HiddenHostConfigRoot = $run1.RunOwnedConfigRoot
    Run1Session = $run1Session
    Run2Session = $run2Session
    AggregateAttempt = $AggregateAttempt
    AggregateAttemptDirectory = $attemptDirectory
    AggregateCaptureDirectory = $aggregateCaptureDirectory
    Scope = 'Exact local .23 image package selection only: one package invocation/five Windows GUI builds, actual candidate/manifest counts, four engine manifests, five PE GUI targets, portable notice/patch attribution and frozen guarded files, immutable .22 protection, and two hidden host lifecycle runs sharing one copied bundle/config. Session/window evidence does not prove successful PNG decode or rendering. No physical input/IME, resource/RSS/VRAM, animation, SVG-image route, or dirty-recovery claim.'
}
$validatorPath = Join-Path $attemptDirectory 'validator.json'
Write-JsonFile $validatorPath $validator 18
Write-JsonFile (Join-Path $attemptDirectory 'protected-after.json') $protectedAfter
Write-JsonFile (Join-Path $attemptDirectory 'protected-evidence-after.json') $protectedEvidenceAfter
Write-JsonFile (Join-Path $attemptDirectory 'protected-capture-after.json') $protectedCaptureAfter
Write-JsonFile (Join-Path $attemptDirectory 'inputs-after.json') $sourceAfter
Write-JsonFile (Join-Path $attemptDirectory 'frozen-bitmap-after.json') $frozenAfter
Write-JsonFile (Join-Path $attemptDirectory 'pinned-patches-after.json') $patchAfter

$aggregate = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    AggregateAttempt = $AggregateAttempt
    AttemptDirectory = $attemptDirectory
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
    HiddenAppStarts = 2
    HostWorkerOperations = $run1.RequestedWorkerOperations + $run2.RequestedWorkerOperations
    SourceIdentity = Get-InventoryIdentity $sourceAfter
    RootSourceAcceptanceFileCount = $sourceGatesAfter.Count
    FrozenBitmapFileCount = $frozenAfter.FileCount
    PatchManifestEntryCount = @($patchAfter.Entries).Count
    ProtectedEvidenceFileCount = $protectedEvidenceAfter.Count
    ProtectedCaptureFileCount = $protectedCaptureAfter.Count
    AggregateCaptureDirectory = $aggregateCaptureDirectory
    Scope = $validator.Scope
}
$aggregatePath = Join-Path $attemptDirectory 'aggregate.json'
$records = @(
    Get-ChildItem -LiteralPath $evidenceRoot -File -Recurse |
        Sort-Object FullName |
        ForEach-Object { Get-FileFingerprint $_.FullName }
)
$aggregate.Evidence = $records
Write-JsonFile $aggregatePath $aggregate 18
$attemptAggregateFingerprint = Get-FileFingerprint $aggregatePath
if (-not $pass) {
    [ordered]@{
        Pass = $false
        Passed = $validator.Passed
        Total = $validator.Total
        AggregateAttempt = $AggregateAttempt
        AttemptDirectory = $attemptDirectory
        AggregateCaptureDirectory = $aggregateCaptureDirectory
        ValidatorSHA256 = (Get-FileHash -LiteralPath $validatorPath -Algorithm SHA256).Hash
        AggregateSHA256 = $attemptAggregateFingerprint.SHA256
    } | ConvertTo-Json
    exit 1
}

$aggregate.SuccessfulAttempt = $AggregateAttempt
$aggregate.AttemptAggregate = $attemptAggregateFingerprint
$aggregate.Evidence = @(
    Get-ChildItem -LiteralPath $evidenceRoot -File -Recurse |
        Sort-Object FullName |
        ForEach-Object { Get-FileFingerprint $_.FullName }
)
$aggregatePath = Join-Path $evidenceRoot 'aggregate.json'
Write-JsonFile $aggregatePath $aggregate 18
[ordered]@{
    Pass = $pass
    Passed = $validator.Passed
    Total = $validator.Total
    AggregateAttempt = $AggregateAttempt
    AttemptDirectory = $attemptDirectory
    AggregateCaptureDirectory = $aggregateCaptureDirectory
    CandidateDirectory = $ctx.candidateDirectory
    FileCount = $candidate.Count
    TotalBytes = Get-InventoryTotalBytes $candidate
    HostSHA256 = $hostFingerprint.SHA256
    ReleaseManifestSHA256 = $manifestFingerprint.SHA256
    ValidatorSHA256 = (Get-FileHash -LiteralPath $validatorPath -Algorithm SHA256).Hash
    AttemptAggregateSHA256 = $attemptAggregateFingerprint.SHA256
    AggregateSHA256 = (Get-FileHash -LiteralPath $aggregatePath -Algorithm SHA256).Hash
} | ConvertTo-Json
