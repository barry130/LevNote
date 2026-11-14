#requires -Version 7.0
param(
    [Parameter(Mandatory=$true)][string]$InputContractPath,
    [Parameter(Mandatory=$true)][ValidatePattern('^[0-9A-Fa-f]{64}$')][string]$InputContractSHA256,
    [switch]$ExecuteReviewed
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
if (-not $ExecuteReviewed) { throw 'Task20 draft is inactive; explicit later root-reviewed execution required.' }
if (-not $IsWindows -or [IntPtr]::Size -ne 8) { throw 'Native Windows x64 PowerShell7 required.' }
# Inline contract/hash preflight occurs before loading any adjacent executable helper.
$contractFull=[IO.Path]::GetFullPath($InputContractPath)
if ((Get-FileHash -LiteralPath $contractFull -Algorithm SHA256).Hash -cne $InputContractSHA256.ToUpperInvariant()) { throw 'Reviewed input contract hash mismatch.' }
$contract=[IO.File]::ReadAllText($contractFull,[Text.UTF8Encoding]::new($false,$true)) | ConvertFrom-Json
foreach ($key in @('executeAuthorized','sourceFrozen','helpersReviewed','ephemeralCheckout')) {
    if ($contract.$key -isnot [bool] -or -not $contract.$key) { throw "False/missing root-reviewed $key." }
}
if ($contract.schema -ne 1 -or $contract.target -cne 'native-windows-amd64') { throw 'Contract schema/target differs.' }
$libraryPath=Join-Path $PSScriptRoot 'ci-lib.ps1'
if ((Get-FileHash -LiteralPath $libraryPath -Algorithm SHA256).Hash -cne $contract.helpers.librarySHA256.ToUpperInvariant() -or
    (Get-FileHash -LiteralPath $PSCommandPath -Algorithm SHA256).Hash -cne $contract.helpers.runnerSHA256.ToUpperInvariant()) { throw 'Final reviewed runner/library hash differs.' }
. $libraryPath
$repo=PlainDirectory $contract.repositoryRoot; $native=Join-Path $repo 'native'
$ownedPath=Join-Path $repo 'native/handoffs/autonomous-local-windows-delivery-20261113/owned-host.cs'
$ownedFact=Pin $ownedPath '68806c747b79445a8eb3db4781c35e07aec69ad50a56a867e2dbc3af634d50fa'
$pwsh=PlainFile ([Diagnostics.Process]::GetCurrentProcess().MainModule.FileName)
$runtimeFact=Pin $pwsh $contract.runtime.sha256
if ($pwsh -cne (LocalPath $contract.runtime.path) -or $PSVersionTable.PSVersion.ToString() -cne $contract.runtime.version) { throw 'Reviewed PowerShell path/version differs.' }
$rootReview=Pin $contract.rootReview.path $contract.rootReview.sha256
$sourceManifestFact=Pin $contract.sourceInventory.path $contract.sourceInventory.sha256
$frozen=Json $contract.sourceInventory.path
if ($frozen.schema -ne 1 -or (LocalPath $frozen.repositoryRoot) -cne $repo) { throw 'Frozen source inventory header differs.' }
$sourceBefore=@(SourceInventory $repo); SameInventory @($frozen.files) $sourceBefore 'Frozen shipping-source/docs inventory'
$null=CheckVendor $native
$version=[string]$contract.packageVersion
if ($version -notmatch '^ci\.[A-Za-z0-9][A-Za-z0-9._+-]{0,50}$') { throw 'CI package version must be a fresh ci.* identity, never a historical runnable.' }
$evidence=LocalPath $contract.evidenceRoot
$parent=PlainDirectory ([IO.Path]::GetDirectoryName($evidence))
$expectedParent=LocalPath (Join-Path $native 'artifacts/ci')
if ($parent -cne $expectedParent -or [IO.Path]::GetFileName($evidence) -notmatch '^native-windows-[A-Za-z0-9._-]+$') { throw 'Evidence must be an absent direct native/artifacts/ci/native-windows-* directory.' }
$tools=Join-Path $native 'tools'; $buildCache=Join-Path $native 'build-cache'
if ((Test-Path -LiteralPath $tools) -or (Test-Path -LiteralPath $buildCache)) { throw 'Clean ephemeral checkout required: tools/build-cache must be absent; no reuse or deletion.' }
$archiveFact=Pin $contract.goArchive.path $script:GoArchiveSHA256
if ($archiveFact.bytes -ne $script:GoArchiveBytes -or $contract.goArchive.sha256 -cne $script:GoArchiveSHA256 -or $contract.goArchive.bytes -ne $script:GoArchiveBytes) { throw 'Official Go archive identity differs.' }
$defaultTemp=PlainDirectory ([IO.Path]::GetTempPath())
if (-not $defaultTemp.StartsWith('C:\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Ordinary default C:TEMP required; unsupported runner layout is not a pass.' }
$commandBound=[int]$contract.commandTimeoutSeconds
if ($commandBound -lt 60 -or $commandBound -gt 1200) { throw 'Command timeout outside60..1200s.' }
$errors=[Collections.Generic.List[object]]::new(); $commands=[Collections.Generic.List[object]]::new(); $guards=[Collections.Generic.List[object]]::new()
$result=[ordered]@{schema=1; status='FAIL'; scope='Native Windows source test/vet/build and exact fresh five-PE package only; not host/physicalGUI/IME/full acceptance'; startedAtUtc=[DateTime]::UtcNow.ToString('o'); repositoryRoot=$repo; packageVersion=$version; sourceFileCount=$sourceBefore.Count; commands=@(); errors=@(); postguards=@(); realWorkerStatus='NOT RUN; opt-in separate reviewed bundle test plan'; physicalGUIStatus='NOT RUN'; toolsRetained=$false; configRetained=$false}
$toolsBefore=$null; $bundleBefore=$null; $config=$null
function RecordError([string]$Phase,[object]$Failure) { $errors.Add([ordered]@{phase=$Phase; detail=$Failure.Exception.ToString(); scriptStackTrace=[string]$Failure.ScriptStackTrace}) }
function Guard([string]$Name,[scriptblock]$Action) {
    try { $fact=& $Action; $guards.Add([ordered]@{name=$Name; pass=$true; facts=$fact}) }
    catch { $guards.Add([ordered]@{name=$Name; pass=$false}); RecordError ('postguard:'+ $Name) $_ }
}
function ChildEnvironment {
    $environment=[Collections.Generic.Dictionary[string,string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in [Environment]::GetEnvironmentVariables('Process').GetEnumerator()) {
        $name=[string]$entry.Key
        # No inherited test opt-ins; real-worker/resource and fixture gates cannot silently alter this base run.
        if (-not $name.StartsWith('=') -and -not $name.StartsWith('LEVNOTE_', [StringComparison]::OrdinalIgnoreCase) -and -not $name.StartsWith('GO', [StringComparison]::OrdinalIgnoreCase) -and $name -ine 'TEST_TELEMETRY_DIR') { $environment[$name]=[string]$entry.Value }
    }
    $environment['PATH']=[IO.Path]::GetDirectoryName($pwsh)+';'+$environment['PATH']
    $environment['GOROOT']=Join-Path $tools 'go'; $environment['GOPATH']=Join-Path $buildCache 'gopath'
    $environment['GOMODCACHE']=Join-Path $buildCache 'gopath/pkg/mod'; $environment['GOCACHE']=Join-Path $buildCache 'build'
    $environment['GOTMPDIR']=Join-Path $buildCache 'tmp'; $environment['TEST_TELEMETRY_DIR']=Join-Path $buildCache 'telemetry'
    $environment['GOENV']='off'; $environment['GOWORK']='off'; $environment['GOFLAGS']='-mod=vendor'
    $environment['GOPROXY']='off'; $environment['GOSUMDB']='off'; $environment['GOVCS']='*:off'; $environment['GOTOOLCHAIN']='local'
    $environment['GOTELEMETRY']='off'; $environment['CGO_ENABLED']='0'; $environment['GOOS']='windows'; $environment['GOARCH']='amd64'; $environment['GOAMD64']='v1'
    $environment['APPDATA']=Join-Path $config 'appdata/Roaming'; $environment['LOCALAPPDATA']=Join-Path $config 'appdata/Local'
    $environment['USERPROFILE']=Join-Path $config 'profile'; $environment['HOME']=$environment['USERPROFILE']
    $environment['HOMEDRIVE']='C:'; $environment['HOMEPATH']=$environment['USERPROFILE'].Substring(2)
    $environment['TEMP']=Join-Path $config 'temp'; $environment['TMP']=$environment['TEMP']
    $environment['XDG_CONFIG_HOME']=$environment['APPDATA']; $environment['XDG_CACHE_HOME']=$environment['LOCALAPPDATA']
    return [string[]]@(foreach ($entry in $environment.GetEnumerator()) { $entry.Key+'='+$entry.Value })
}
function Command([string]$Name,[string]$Executable,[string[]]$Arguments,[int]$Timeout=$commandBound) {
    $directory=FreshDirectory (Join-Path $evidence $Name)
    $stdin=Join-Path $directory 'stdin.empty'; $output=Join-Path $directory 'stdout.log'; $errorOutput=Join-Path $directory 'stderr.log'
    $inputFile=[IO.File]::Open($stdin,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read); $inputFile.Dispose()
    $record=[ordered]@{name=$Name; executable=$Executable; executableFingerprint=(Fingerprint $Executable); arguments=$Arguments; cwd=$native; timeoutSeconds=$Timeout; startedAtUtc=[DateTime]::UtcNow.ToString('o'); exitCode=$null; timedOut=$false; completed=$false; cleanupForced=$false; jobEmptyObserved=$false; errors=@()}
    $failures=[Collections.Generic.List[object]]::new(); $ownedCommand=[LevNoteLocalAcceptance.OwnedHost]::new()
    try {
        $environmentRows=ChildEnvironment
        $record.selectedEnvironment=[ordered]@{}
        foreach ($entry in $environmentRows) {
            $parts=$entry.Split('=',2)
            if ($parts[0] -match '^(GO[A-Z0-9_]*|CGO_ENABLED|TEST_TELEMETRY_DIR|APPDATA|LOCALAPPDATA|USERPROFILE|HOME|HOMEDRIVE|HOMEPATH|TEMP|TMP|XDG_CONFIG_HOME|XDG_CACHE_HOME)$') { $record.selectedEnvironment[$parts[0]]=$parts[1] }
        }
        $ownedCommand.Start($Executable,$Arguments,$native,$environmentRows,$stdin,$output,$errorOutput)
        $record.pid=$ownedCommand.ProcessId; $record.imagePath=$ownedCommand.ImagePath; $record.processStartUtc=$ownedCommand.StartUtc.ToString('o')
        $record.createdSuspended=$ownedCommand.CreatedSuspended; $record.assignedBeforeResume=$ownedCommand.AssignedBeforeResume; $record.resumed=$ownedCommand.Resumed
        if (-not $ownedCommand.WaitExit($Timeout*1000)) { $record.timedOut=$true; throw 'Owned command exceeded wall-clock bound.' }
        $record.exitCode=$ownedCommand.ExitCode()
        $join=[Diagnostics.Stopwatch]::StartNew()
        do {
            $accounting=$ownedCommand.Accounting(); $record.jobAfterExit=$accounting
            if ($accounting.ActiveProcesses -eq 0) { $record.jobEmptyObserved=$true; break }
            Start-Sleep -Milliseconds 50
        } while ($join.ElapsedMilliseconds -lt 5000)
        if (-not $record.jobEmptyObserved) { throw 'Root command exited but Job descendants not observed empty within5s.' }
        if ($record.exitCode -ne 0) { throw "Owned command exited $($record.exitCode); no retry." }
        $record.completed=$true
    } catch { $failures.Add($_.Exception.ToString()) }
    finally {
        try {
            if ($ownedCommand.HasProcess) {
                $ownedCommand.ForceCleanup(10000); $record.cleanupForced=$ownedCommand.ForcedCleanup
                if ($ownedCommand.AssignedBeforeResume) {
                    $record.cleanupAccounting=$ownedCommand.Accounting()
                    if ($record.cleanupAccounting.ActiveProcesses -ne 0) { throw 'Cleanup active0 not observed; host join/handle close cannot upgrade FAIL.' }
                }
            }
        } catch { $failures.Add($_.Exception.ToString()) }
        $hadProcess=$ownedCommand.HasProcess
        try { $ownedCommand.Dispose() } catch { $failures.Add($_.Exception.ToString()) }
        $record.finishedAtUtc=[DateTime]::UtcNow.ToString('o')
        $record.streams=@(foreach ($path in @($stdin,$output,$errorOutput)) { if (Test-Path -LiteralPath $path -PathType Leaf) { Fingerprint $path ([IO.Path]::GetFileName($path)) } })
        if ($hadProcess -and $record.streams.Count -ne 3) { $failures.Add('Started command lacks complete stdin/stdout/stderr capture set.') }
        $record.errors=$failures.ToArray()
        $commands.Add($record); NewJson (Join-Path $directory 'command-result.json') $record
    }
    if ($failures.Count -or -not $record.completed -or $record.cleanupForced) { throw "Native CI command failed: $Name (original streams/cleanup errors retained)." }
}
$null=FreshDirectory $evidence
try {
    NewJson (Join-Path $evidence 'inputs-before.json') ([ordered]@{contract=(Pin $contractFull $InputContractSHA256); rootReview=$rootReview; sourceManifest=$sourceManifestFact; source=$sourceBefore; runtime=$runtimeFact; ownedHelper=$ownedFact; goArchive=$archiveFact; productRoots=$script:ProductRoots; cgo='0 (matching current local lane, not other-platform proof)'})
    $null=ExtractVerifiedGo $contract.goArchive.path $tools
    $toolsBefore=@(Inventory $tools); $result.toolsRetained=$true
    $null=FreshDirectory $buildCache
    foreach ($name in @('build','gopath','gopath/pkg/mod','tmp','telemetry')) { New-Item -ItemType Directory -Path (Join-Path $buildCache $name) -Force | Out-Null }
    [IO.File]::WriteAllText((Join-Path $buildCache 'telemetry/mode'),"off`n",[Text.Encoding]::ASCII)
    [IO.File]::WriteAllText((Join-Path $buildCache 'go.mod'),"module levnote.local/build-cache`n`ngo 1.27.3`n",[Text.UTF8Encoding]::new($false))
    $config=FreshDirectory (Join-Path $defaultTemp ('levnote-native-ci-'+[Guid]::NewGuid().ToString('N')))
    foreach ($name in @('appdata/Roaming','appdata/Local','profile','temp')) { New-Item -ItemType Directory -Path (Join-Path $config $name) -Force | Out-Null }
    $result.configRoot=$config; $result.configRetained=$true
    # Frozen Task16 is read-only dependency; compile only in this later explicitly activated run.
    Add-Type -TypeDefinition ([IO.File]::ReadAllText($ownedPath)) -Language CSharp
    $go=Join-Path $tools 'go/bin/go.exe'
    Command '00-version' $go @('version') 60
    $versionOutput=[IO.File]::ReadAllText((Join-Path $evidence '00-version/stdout.log')).Trim()
    if ($versionOutput -cne 'go version go1.27.3 windows/amd64') { throw 'Actual verified toolchain version/target differs.' }
    $null=CheckVendor $native
    Command '01-build-guard' $pwsh @('-NoLogo','-NoProfile','-NonInteractive','-File',(Join-Path $native 'scripts/check-build-guard.ps1'),'-EvidenceDirectory',(Join-Path $evidence 'guard-cases'),'-CaseSet','All')
    $guardCases=Json (Join-Path $evidence 'guard-cases/result.json')
    if ($guardCases.Status -cne 'PASS' -or $guardCases.CaseSet -cne 'All' -or $guardCases.Total -le 0 -or $guardCases.Passed -ne $guardCases.Total) { throw 'Existing negative/context guard cases not fully passed.' }
    $result.buildGuard=[ordered]@{passed=$guardCases.Passed; total=$guardCases.Total; fixtureScope='Existing tiny isolated modules, not product acceptance'}
    Command '02-test' $go (@('test','-json','-count=1','-timeout=15m')+$script:ProductRoots)
    # Test JSON is mandatory and skips are reported, never translated into feature acceptance.
    $testEvents=@([IO.File]::ReadLines((Join-Path $evidence '02-test/stdout.log')) | ForEach-Object { $_ | ConvertFrom-Json -ErrorAction Stop })
    $failEvents=@($testEvents | Where-Object { $_.Action -ceq 'fail' }); $skipEvents=@($testEvents | Where-Object { $_.Action -ceq 'skip' })
    $packagePass=@($testEvents | Where-Object { $_.Action -ceq 'pass' -and -not $_.PSObject.Properties['Test'] })
    if ($failEvents.Count -or -not $packagePass.Count) { throw 'Native JSON tests include failure or no completed package PASS.' }
    NewJson (Join-Path $evidence 'test-summary.json') ([ordered]@{failCount=$failEvents.Count; skipCount=$skipEvents.Count; skipped=$skipEvents; packagePassCount=$packagePass.Count; noPhysicalGUIClaim=$true; noRealBundleOptIns=$true})
    $result.testSkipCount=$skipEvents.Count
    Command '03-vet' $go (@('vet')+$script:ProductRoots)
    Command '04-build' $go (@('build')+$script:ProductRoots)
    $bundle=Join-Path $evidence 'package'; $packageEvidence=Join-Path $evidence 'package-build'
    Command '05-package' $pwsh @('-NoLogo','-NoProfile','-NonInteractive','-File',(Join-Path $native 'scripts/package-markdown-worker.ps1'),'-OutputDirectory',$bundle,'-EvidenceDirectory',$packageEvidence,'-Version',$version)
    $packageFacts=CheckFivePEPackage $native $bundle $packageEvidence $version
    $bundleBefore=@($packageFacts.files); $result.packageFacts=$packageFacts
    NewJson (Join-Path $evidence 'five-pe-check.json') $packageFacts
} catch { RecordError 'run' $_ }
finally {
    Guard 'source-unchanged' { $after=@(SourceInventory $repo); SameInventory $sourceBefore $after 'Complete frozen source/docs'; NewJson (Join-Path $evidence 'source-after.json') $after; return $after.Count }
    Guard 'vendor26-unchanged' { CheckVendor $native }
    Guard 'activation-and-helper-pins' {
        SameFingerprint $rootReview (Fingerprint $contract.rootReview.path)
        SameFingerprint $sourceManifestFact (Fingerprint $contract.sourceInventory.path)
        $null=Pin $contractFull $InputContractSHA256; $null=Pin $libraryPath $contract.helpers.librarySHA256; $null=Pin $PSCommandPath $contract.helpers.runnerSHA256
        SameFingerprint $ownedFact (Fingerprint $ownedPath); SameFingerprint $runtimeFact (Fingerprint $pwsh); SameFingerprint $archiveFact (Fingerprint $contract.goArchive.path)
        return 'All named activation/runtime/helper/archive pins unchanged'
    }
    if ($null -ne $toolsBefore) { Guard 'toolchain-unchanged' { SameInventory $toolsBefore @(Inventory $tools) 'Verified Go extraction'; return $toolsBefore.Count } }
    if ($null -ne $bundleBefore) { Guard 'bundle-unchanged' { SameInventory $bundleBefore @(Inventory (Join-Path $evidence 'package')) 'Fresh five-PE bundle'; return $bundleBefore.Count } }
    if ($null -ne $config) { Guard 'retained-config' { $facts=@(Inventory $config); NewJson (Join-Path $evidence 'config-after.json') $facts; return [ordered]@{path=$config; count=$facts.Count; retained=$true} } }
    $result.commands=$commands.ToArray(); $result.errors=$errors.ToArray(); $result.postguards=$guards.ToArray(); $result.finishedAtUtc=[DateTime]::UtcNow.ToString('o')
    if ($errors.Count -eq 0 -and $commands.Count -eq 6 -and $null -ne $bundleBefore) { $result.status='PASS_LIMITED_NATIVE_CI' }
    NewJson (Join-Path $evidence 'result.json') $result
}
$result | ConvertTo-Json -Depth 6
if ($result.status -cne 'PASS_LIMITED_NATIVE_CI') { exit 1 }
exit 0
