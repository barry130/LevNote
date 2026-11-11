$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not (Test-Path -LiteralPath (Join-Path $evidenceRoot 'baseline-result.json') -PathType Leaf)) {
    throw 'Run the hidden baseline stage first.'
}
if (Test-Path -LiteralPath $ctx.candidateDirectory) {
    throw "Fresh .19 candidate directory already exists: $($ctx.candidateDirectory)"
}
$packageEvidence = Join-Path $evidenceRoot 'package'
$packageCopy = Join-Path $evidenceRoot 'package-copy'
if (Test-Path -LiteralPath $packageEvidence) {
    throw "Fresh package evidence directory already exists: $packageEvidence"
}
if (Test-Path -LiteralPath $packageCopy) {
    throw "Fresh package copy already exists: $packageCopy"
}

$invokePath = Join-Path $evidenceRoot 'package-invoke.ps1'
$invokeScript = @"
`$ErrorActionPreference = 'Stop'
& $(Quote-PowerShellLiteral $ctx.packageScript) -OutputDirectory $(Quote-PowerShellLiteral $ctx.candidateDirectory) -EvidenceDirectory $(Quote-PowerShellLiteral $packageEvidence) -Version $(Quote-PowerShellLiteral $ctx.version)
exit `$LASTEXITCODE
"@
[IO.File]::WriteAllText($invokePath, $invokeScript, [Text.UTF8Encoding]::new($false))

$command = Invoke-CapturedPowerShellScript 'package' $invokePath $ctx.nativeRoot $evidenceRoot
$command.InvocationCount = 1
$command.PackageScript = $ctx.packageScript
$command.PackageArguments = [ordered]@{
    OutputDirectory = $ctx.candidateDirectory
    EvidenceDirectory = $packageEvidence
    Version = $ctx.version
}
$command.ExpectedGuiBuilds = $ctx.ExpectedGuiBuilds
Write-JsonFile (Join-Path $evidenceRoot 'package-command.json') $command
if ($command.ExitCode -ne 0) {
    throw "The single package invocation failed with exit $($command.ExitCode); its raw streams are preserved."
}

$packageResultPath = Join-Path $packageEvidence 'package-result.json'
if (-not (Test-Path -LiteralPath $packageResultPath -PathType Leaf)) {
    throw 'Package invocation exited zero without package-result.json.'
}
$package = Read-JsonFile $packageResultPath
$buildNames = @('worker-markdown', 'worker-mindmap', 'worker-mindmap-preview', 'worker-drawio', 'host')
$builds = @(
    foreach ($name in $buildNames) {
        $path = Join-Path $packageEvidence ($name + '-build.json')
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Expected GUI build record is absent: $name"
        }
        Read-JsonFile $path
    }
)
$buildBad = @(
    $builds | Where-Object {
        $_.ExitCode -ne 0 -or $_.StdoutBytes -ne 0 -or $_.StderrBytes -ne 0 -or
        (Get-Item -LiteralPath $_.StdoutPath).Length -ne 0 -or
        (Get-Item -LiteralPath $_.StderrPath).Length -ne 0
    }
)
if ($builds.Count -ne 5 -or $buildBad.Count -ne 0) {
    throw "Expected five successful GUI builds with empty streams; found $($builds.Count) builds and $($buildBad.Count) failures."
}

$candidate = @(Get-DirectoryInventory $ctx.candidateDirectory)
$expectedFiles = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
$expectedFiles.Add('levnote-native.exe') | Out-Null
$expectedFiles.Add('release-manifest.json') | Out-Null
$fixedEngines = @(
    [pscustomobject]@{ Engine = 'levnote.markdown-export'; Format = 'html'; RelativePath = 'engines/markdown-export'; Filename = 'levnote-markdown-export.exe' },
    [pscustomobject]@{ Engine = 'levnote.mindmap-svg'; Format = 'svg'; RelativePath = 'engines/mindmap-svg'; Filename = 'levnote-mindmap-svg.exe' },
    [pscustomobject]@{ Engine = 'levnote.mindmap-preview'; Format = 'mindmap-scene-v1'; RelativePath = 'engines/mindmap-preview'; Filename = 'levnote-mindmap-preview.exe' },
    [pscustomobject]@{ Engine = 'levnote.drawio-svg'; Format = 'svg'; RelativePath = 'engines/drawio-svg'; Filename = 'levnote-drawio-svg.exe' }
)
foreach ($engine in $fixedEngines) {
    $expectedFiles.Add($engine.RelativePath + '/' + $engine.Filename) | Out-Null
    $expectedFiles.Add($engine.RelativePath + '/manifest.json') | Out-Null
}
if (@($package.Engines).Count -ne $fixedEngines.Count) {
    throw "Package result contains $(@($package.Engines).Count) engines, expected four."
}
$packageEngineBad = @(
    foreach ($fixed in $fixedEngines) {
        $engineMatches = @($package.Engines | Where-Object {
            $_.Engine -ceq $fixed.Engine -and $_.Format -ceq $fixed.Format -and
            $_.RelativePath -ceq $fixed.RelativePath
        })
        if ($engineMatches.Count -ne 1) { $fixed.Engine }
    }
)
if ($packageEngineBad.Count -ne 0) {
    throw "Fixed catalog engine identities are incomplete or duplicated: $($packageEngineBad -join ', ')"
}
foreach ($copy in @($package.SourceCopies)) {
    $expectedFiles.Add([string]$copy.RelativePath) | Out-Null
}
$expectedFiles.Add('README.md') | Out-Null
$expectedFiles.Add('THIRD_PARTY_NOTICES.md') | Out-Null
$actualNames = @($candidate | ForEach-Object { $_.RelativePath })
$missing = @($expectedFiles | Where-Object { $actualNames -notcontains $_ } | Sort-Object)
$unexpected = @($actualNames | Where-Object { -not $expectedFiles.Contains($_) } | Sort-Object)
if ($candidate.Count -ne [int]$ctx.ExpectedCandidateFiles) {
    throw "Actual .19 candidate inventory is $($candidate.Count); expected assertion is $($ctx.ExpectedCandidateFiles)."
}
if ($missing.Count -ne 0 -or $unexpected.Count -ne 0 -or
    $expectedFiles.Count -ne [int]$ctx.ExpectedCandidateFiles) {
    throw "Candidate artifact set differs: missing=$($missing -join ','); unexpected=$($unexpected -join ',')"
}

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
if ($release.schema -ne 1 -or $release.version -cne $ctx.version -or
    $release.goos -cne 'windows' -or $release.goarch -cne 'amd64' -or
    $release.goamd64 -cne 'v1' -or
    $releaseEntries.Count -ne ($candidate.Count - 1) -or
    @($release.excludes).Count -ne 1 -or @($release.excludes)[0] -cne 'release-manifest.json' -or
    $releaseBad.Count -ne 0) {
    throw "Release manifest schema1 or payload identities differ: bad=$($releaseBad -join ',')"
}

$engineChecks = @(
    foreach ($fixed in $fixedEngines) {
        $manifestPath = Join-Path $ctx.candidateDirectory ($fixed.RelativePath + '/manifest.json')
        $manifest = Read-JsonFile $manifestPath
        $workerPath = Join-Path $ctx.candidateDirectory ($fixed.RelativePath + '/' + $fixed.Filename)
        $worker = Get-FileFingerprint $workerPath
        $manifestFile = Get-FileFingerprint $manifestPath
        $pass = $manifest.schema -eq 1 -and $manifest.engine -ceq $fixed.Engine -and
            $manifest.format -ceq $fixed.Format -and $manifest.engineVersion -ceq $ctx.version -and
            $manifest.filename -ceq $fixed.Filename -and
            [Int64]$manifest.size -eq $worker.Bytes -and
            ([string]$manifest.sha256).ToLowerInvariant() -cne $null -and
            ([string]$manifest.sha256).ToLowerInvariant() -ceq $worker.SHA256.ToLowerInvariant()
        [ordered]@{
            Engine = $fixed.Engine
            Format = $fixed.Format
            RelativePath = $fixed.RelativePath
            Worker = $worker
            Manifest = $manifestFile
            Pass = $pass
        }
    }
)
if (@($engineChecks | Where-Object { -not $_.Pass }).Count -ne 0) {
    throw 'One or more fixed engine manifests do not match the .19 payload identity.'
}

$peFacts = @(
    foreach ($entry in $candidate | Where-Object { $_.RelativePath.EndsWith('.exe') }) {
        Get-PEInfo $entry.Path $entry.SHA256
    }
)
if ($peFacts.Count -ne 5 -or @($peFacts | Where-Object { -not $_.Pass }).Count -ne 0) {
    throw 'Expected five Windows amd64 GUI PE executables.'
}

$assetChecks = @(
    foreach ($copy in @($package.SourceCopies)) {
        $source = Get-FileFingerprint ([string]$copy.Source)
        $shipped = Get-FileFingerprint (Join-Path $ctx.candidateDirectory ([string]$copy.RelativePath))
        [ordered]@{
            RelativePath = [string]$copy.RelativePath
            Source = $source
            Shipped = $shipped
            Pass = $source.SHA256 -ceq ([string]$copy.SHA256) -and $source.SHA256 -ceq $shipped.SHA256
        }
    }
)
if ($assetChecks.Count -eq 0 -or @($assetChecks | Where-Object { -not $_.Pass }).Count -ne 0) {
    throw 'Portable source-copy assets do not retain their exact hashes.'
}

$sourcePaths = @(Get-ChildItem -LiteralPath $ctx.candidateDirectory -File -Recurse)
New-Item -ItemType Directory -Path $packageCopy -ErrorAction Stop | Out-Null
foreach ($source in $sourcePaths) {
    $relative = [IO.Path]::GetRelativePath($ctx.candidateDirectory, $source.FullName)
    $destination = Join-Path $packageCopy $relative
    $parent = Split-Path $destination -Parent
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null
    }
    Copy-Item -LiteralPath $source.FullName -Destination $destination -ErrorAction Stop
}
$copyInventory = @(Get-DirectoryInventory $packageCopy)
Assert-InventoryContainsExact $candidate $copyInventory 'Complete package copy' $false

$packageHost = Get-FileFingerprint (Join-Path $ctx.candidateDirectory 'levnote-native.exe')
$manifestFingerprint = Get-FileFingerprint $releasePath
$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    Status = 'PASS'
    PackageInvocationCount = 1
    GuiBuildCount = $builds.Count
    Builds = $builds
    Version = $ctx.version
    CandidateDirectory = $ctx.candidateDirectory
    CandidateFileCount = $candidate.Count
    CandidateTotalBytes = Get-InventoryTotalBytes $candidate
    CandidateInventoryIdentity = Get-InventoryIdentity $candidate
    ExpectedCandidateFileCount = $ctx.ExpectedCandidateFiles
    ExpectedCandidateFileSetPass = $true
    PackageCopyDirectory = $packageCopy
    PackageCopyFileCount = $copyInventory.Count
    PackageCopyTotalBytes = Get-InventoryTotalBytes $copyInventory
    Host = $packageHost
    ReleaseManifest = $manifestFingerprint
    ReleaseManifestPayloadEntries = $releaseEntries.Count
    ReleaseManifestSchema = $release.schema
    EngineChecks = $engineChecks
    PEFacts = $peFacts
    SourceCopyCount = $assetChecks.Count
    SourceCopies = $assetChecks
    LocalAuthoredLinksChecked = [bool]$package.LocalAuthoredLinksChecked
}
Write-JsonFile (Join-Path $evidenceRoot 'package-result.json') $result 16
$result | ConvertTo-Json -Depth 7
