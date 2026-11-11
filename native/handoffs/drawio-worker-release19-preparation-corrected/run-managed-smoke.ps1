$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not (Test-Path -LiteralPath (Join-Path $evidenceRoot 'package-result.json') -PathType Leaf)) {
    throw 'Run the package stage first.'
}
$smokeRoot = Join-Path $evidenceRoot 'managed-smoke'
$bundleCopy = Join-Path $smokeRoot 'bundle-copy'
$configRoot = Join-Path $smokeRoot 'config'
$recordPath = Join-Path $smokeRoot 'record.json'
$compilePath = Join-Path $smokeRoot 'compile.ps1'
$testExe = Join-Path $bundleCopy 'engineruntime.test.exe'
foreach ($path in @($smokeRoot, $bundleCopy, $configRoot, $recordPath)) {
    if (Test-Path -LiteralPath $path) {
        throw "Managed smoke path must be fresh and absent: $path"
    }
}

$candidateBefore = @(Get-DirectoryInventory $ctx.candidateDirectory)
if ($candidateBefore.Count -ne [int]$ctx.ExpectedCandidateFiles) {
    throw "Managed smoke requires the complete candidate, found $($candidateBefore.Count) files."
}
New-Item -ItemType Directory -Path $bundleCopy -ErrorAction Stop | Out-Null
foreach ($source in Get-ChildItem -LiteralPath $ctx.candidateDirectory -File -Recurse) {
    $relative = [IO.Path]::GetRelativePath($ctx.candidateDirectory, $source.FullName)
    $destination = Join-Path $bundleCopy $relative
    $parent = Split-Path $destination -Parent
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null
    }
    Copy-Item -LiteralPath $source.FullName -Destination $destination -ErrorAction Stop
}
$bundleBeforeCompile = @(Get-DirectoryInventory $bundleCopy)
Assert-InventoryContainsExact $candidateBefore $bundleBeforeCompile 'Managed smoke initial bundle copy' $false

$compileScript = @"
`$ErrorActionPreference = 'Stop'
`$env:GOOS = 'windows'
`$env:GOARCH = 'amd64'
`$env:GOAMD64 = 'v1'
& $(Quote-PowerShellLiteral $ctx.GoWrapper) -GoArguments @('test','-c','-o',$(Quote-PowerShellLiteral $testExe),'./core/engineruntime')
exit `$LASTEXITCODE
"@
[IO.File]::WriteAllText($compilePath, $compileScript, [Text.UTF8Encoding]::new($false))

$compileRecord = Invoke-CapturedPowerShellScript 'managed-tester-compile' $compilePath $ctx.nativeRoot $smokeRoot
$compileRecord.CompileCount = 1
$compileRecord.OutputPath = $testExe
Write-JsonFile (Join-Path $smokeRoot 'compile-invocation.json') $compileRecord 8
if ($compileRecord.ExitCode -ne 0) {
    throw "The single managed tester compile failed with exit $($compileRecord.ExitCode); raw streams are preserved."
}
if (-not (Test-Path -LiteralPath $testExe -PathType Leaf)) {
    throw 'Managed tester compile exited zero without the expected executable.'
}

$oldEnable = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE', 'Process')
$oldCandidate = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE', 'Process')
$oldEvidence = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE', 'Process')
$oldConfig = [Environment]::GetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG', 'Process')
$runStdout = Join-Path $smokeRoot 'run.stdout.log'
$runStderr = Join-Path $smokeRoot 'run.stderr.log'
$runStarted = [DateTime]::UtcNow
$watch = [Diagnostics.Stopwatch]::StartNew()
try {
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE', '1', 'Process')
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE', $ctx.candidateDirectory, 'Process')
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE', $recordPath, 'Process')
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG', $configRoot, 'Process')
    $process = Start-Process -FilePath $testExe -ArgumentList @(
        '-test.run', $ctx.SmokeTestName, '-test.v'
    ) -WorkingDirectory $bundleCopy -WindowStyle Hidden `
        -RedirectStandardOutput $runStdout -RedirectStandardError $runStderr -PassThru -Wait
    $runExit = $process.ExitCode
    $process.Dispose()
} finally {
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE', $oldEnable, 'Process')
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE', $oldCandidate, 'Process')
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE', $oldEvidence, 'Process')
    [Environment]::SetEnvironmentVariable('LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG', $oldConfig, 'Process')
}
$watch.Stop()

$runRecord = [ordered]@{
    Schema = 1
    Name = 'managed-tester-run'
    Command = $testExe
    Arguments = @('-test.run', $ctx.SmokeTestName, '-test.v')
    WorkingDirectory = $bundleCopy
    StartedAtUtc = $runStarted.ToString('o')
    FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
    WallSeconds = $watch.Elapsed.TotalSeconds
    ExitCode = $runExit
    StdoutPath = $runStdout
    StderrPath = $runStderr
    StdoutBytes = [Int64](Get-Item -LiteralPath $runStdout).Length
    StderrBytes = [Int64](Get-Item -LiteralPath $runStderr).Length
    WindowStyle = 'Hidden'
    RunCount = 1
    Environment = [ordered]@{
        LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE = '1'
        LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE = $ctx.candidateDirectory
        LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE = $recordPath
        LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG = $configRoot
    }
}
Write-JsonFile (Join-Path $smokeRoot 'run-invocation.json') $runRecord 8
if ($runExit -ne 0) {
    throw "The single enabled managed smoke failed with exit $runExit; raw compile/run streams are preserved."
}
if (-not (Test-Path -LiteralPath $recordPath -PathType Leaf)) {
    throw 'Managed tester exited zero without its structured record.'
}
$structured = Read-JsonFile $recordPath
$rolled = Get-ManagedRecord $runStdout
if (($structured | ConvertTo-Json -Depth 16 -Compress) -cne ($rolled | ConvertTo-Json -Depth 16 -Compress)) {
    throw 'Managed structured record file and logged record differ.'
}

$expectedViolations = [Collections.Generic.List[string]]::new()
if ($structured.schema -ne 1 -or $structured.status -cne 'PASS') {
    $expectedViolations.Add('schema/status')
}
if ([IO.Path]::GetFullPath([string]$structured.candidateRelease) -ne [IO.Path]::GetFullPath($ctx.candidateDirectory) -or
    [IO.Path]::GetFullPath([string]$structured.bundleCopy) -ne [IO.Path]::GetFullPath($bundleCopy) -or
    [IO.Path]::GetFullPath([string]$structured.evidencePath) -ne [IO.Path]::GetFullPath($recordPath) -or
    [IO.Path]::GetFullPath([string]$structured.configRoot) -ne [IO.Path]::GetFullPath($configRoot)) {
    $expectedViolations.Add('paths')
}
if ($structured.candidateVersion -cne $ctx.version -or
    $structured.candidateExeSHA256Before -cne $structured.candidateExeSHA256After -or
    $structured.bundledFourthSelection -cne 'bundled' -or
    $structured.importFourthSelection -cne 'bundled' -or
    $structured.activatedFourthSelection -cne ('installed:' + $structured.importDigest) -or
    $structured.initialSelections.markdown -cne 'bundled' -or
    $structured.initialSelections.mindmapSVG -cne 'bundled' -or
    $structured.initialSelections.mindmapPreview -cne 'bundled' -or
    $structured.afterImportSelections.markdown -cne 'bundled' -or
    $structured.afterImportSelections.mindmapSVG -cne 'bundled' -or
    $structured.afterImportSelections.mindmapPreview -cne 'bundled' -or
    $structured.afterActivateSelections.markdown -cne 'bundled' -or
    $structured.afterActivateSelections.mindmapSVG -cne 'bundled' -or
    $structured.afterActivateSelections.mindmapPreview -cne 'bundled') {
    $expectedViolations.Add('selection/version')
}
if ([int]$structured.sourceBytes -le 0 -or
    $structured.sourceSHA256Before -cne $structured.sourceSHA256After -or
    [string]::IsNullOrWhiteSpace([string]$structured.payloadSHA256) -or
    [int]$structured.export.newManagedCalls -ne 1 -or
    [int]$structured.export.exportCalls -ne 1 -or
    [int]$structured.export.admissionCalls -ne 1 -or
    [int]$structured.export.verifyCalls -ne 1 -or
    [int]$structured.export.startCalls -ne 1 -or
    [int]$structured.export.waitCalls -ne 1 -or
    -not [bool]$structured.export.returned -or
    [int]$structured.export.pageIndex -ne 1 -or
    [string]$structured.export.pageID -cne 'second') {
    $expectedViolations.Add('export/operation counts')
}
if (-not [bool]$structured.process.startObserved -or
    -not [bool]$structured.process.duplicateHandleObserved -or
    -not [bool]$structured.process.imagePathObserved -or
    -not [bool]$structured.process.creationObserved -or
    [uint64]$structured.process.creationFiletime100ns -eq 0 -or
    -not [bool]$structured.process.waitObserved -or
    $null -eq $structured.process.exitCode -or [int]$structured.process.exitCode -ne 0 -or
    -not [bool]$structured.process.primaryClosed -or
    [bool]$structured.process.fallbackCloseAttempted -or
    -not [bool]$structured.runner.closeCalled -or
    [string]::IsNullOrWhiteSpace([string]$structured.process.imagePath) -or
    [string]::IsNullOrWhiteSpace([string]$structured.process.creationUTC)) {
    $expectedViolations.Add('process lifecycle')
}
$warnings = @($structured.warnings)
if ($warnings.Count -ne 2 -or
    $warnings[0].code -cne 'UNSUPPORTED_ELEMENT' -or
    $warnings[0].message -cne 'page 1 cell 0: mxCell is neither a vertex nor an edge and is not rendered' -or
    $warnings[1].code -cne 'UNSUPPORTED_ELEMENT' -or
    $warnings[1].message -cne 'page 1 cell 1: mxCell is neither a vertex nor an edge and is not rendered' -or
    [uint32]$warnings[0].start -ne 0 -or [uint32]$warnings[1].start -ne 0 -or
    [uint32]$warnings[0].end -ne [uint32]$structured.sourceBytes -or
    [uint32]$warnings[1].end -ne [uint32]$structured.sourceBytes) {
    $expectedViolations.Add('warning metadata/order')
}
if ($expectedViolations.Count -ne 0) {
    throw "Managed smoke record validation failed: $($expectedViolations -join ', ')"
}

$candidateAfter = @(Get-DirectoryInventory $ctx.candidateDirectory)
Assert-InventoryContainsExact $candidateBefore $candidateAfter 'Candidate during managed smoke' $false
$bundleAfter = @(Get-DirectoryInventory $bundleCopy)
if ($bundleAfter.Count -ne ($candidateBefore.Count + 1)) {
    throw "Managed bundle copy contains $($bundleAfter.Count) files; expected candidate plus tester."
}
$bundleWithoutTester = @($bundleAfter | Where-Object { $_.RelativePath -cne 'engineruntime.test.exe' })
Assert-InventoryContainsExact $candidateBefore $bundleWithoutTester 'Managed bundle payload after smoke' $false
$testFingerprint = Get-FileFingerprint $testExe
$recordFingerprint = Get-FileFingerprint $recordPath
$warningBytes = [Text.Encoding]::UTF8.GetBytes(($warnings | ConvertTo-Json -Depth 6 -Compress))
$result = [ordered]@{
    Schema = 1
    Status = 'PASS'
    CandidateDirectory = $ctx.candidateDirectory
    BundleCopy = $bundleCopy
    ConfigRoot = $configRoot
    StructuredRecord = $recordFingerprint
    TesterExecutable = $testFingerprint
    CompileInvocation = Join-Path $smokeRoot 'compile-invocation.json'
    RunInvocation = Join-Path $smokeRoot 'run-invocation.json'
    RawCompileStdout = $compileRecord.StdoutPath
    RawCompileStderr = $compileRecord.StderrPath
    RawRunStdout = $runStdout
    RawRunStderr = $runStderr
    CompileExit = $compileRecord.ExitCode
    RunExit = $runExit
    ManagedTesterCompiles = 1
    ManagedTesterRuns = 1
    TestName = $ctx.SmokeTestName
    SourceBytes = $structured.sourceBytes
    SourceSHA256Before = $structured.sourceSHA256Before
    SourceSHA256After = $structured.sourceSHA256After
    PayloadBytes = $structured.payloadBytes
    PayloadSHA256 = $structured.payloadSHA256
    WarningsSHA256 = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($warningBytes))
    Warnings = $warnings
    Export = $structured.export
    Process = $structured.process
    Runner = $structured.runner
    CandidateFileCount = $candidateAfter.Count
    CandidateInventoryIdentity = Get-InventoryIdentity $candidateAfter
    BundlePayloadFileCount = $bundleWithoutTester.Count
    BundleAfterFileCount = $bundleAfter.Count
}
Write-JsonFile (Join-Path $smokeRoot 'result.json') $result 16
$result | ConvertTo-Json -Depth 7
