#requires -Version 7.0
param(
    [Parameter(Mandatory=$true)][string]$InputContractPath,
    [Parameter(Mandatory=$true)][ValidatePattern('^[0-9A-Fa-f]{64}$')][string]$InputContractSHA256,
    [switch]$ExecuteReviewed
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $ExecuteReviewed) { throw 'Prepared-only harness. Root must review, freeze inputs, and explicitly select ExecuteReviewed.' }
if (-not $IsWindows -or -not [Environment]::Is64BitProcess) { throw 'Requires native Windows x64 PowerShell 7, not Linux/Wine/32-bit.' }
# Verify the new helper before loading it. Contract is data, never command text.
$contractFull = [IO.Path]::GetFullPath($InputContractPath)
if ((Get-FileHash -LiteralPath $contractFull -Algorithm SHA256).Hash -cne $InputContractSHA256.ToUpperInvariant()) { throw 'Input contract pin differs.' }
$contract = [IO.File]::ReadAllText($contractFull, [Text.UTF8Encoding]::new($false, $true)) | ConvertFrom-Json
if ($contract.schema -ne 1 -or $contract.executionAuthorized -isnot [bool] -or -not $contract.executionAuthorized -or
    $contract.sourceFrozen -isnot [bool] -or -not $contract.sourceFrozen -or
    $contract.packageFrozen -isnot [bool] -or -not $contract.packageFrozen) { throw 'Root-reviewed frozen contract required.' }
$helperPins = @($contract.helpers)
foreach ($name in @('run-host-smoke.ps1','delivery-lib.ps1','owned-host.cs')) {
    $rows = @($helperPins | Where-Object { $_.name -ceq $name })
    if ($rows.Count -ne 1 -or $rows[0].sha256 -cnotmatch '^[0-9A-Fa-f]{64}$') { throw "Missing unique helper pin: $name" }
    $path = Join-Path $PSScriptRoot $name
    if ((Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash -cne $rows[0].sha256.ToUpperInvariant()) { throw "Reviewed helper changed: $name" }
}
if ($helperPins.Count -ne 3) { throw 'Runtime helper set must be exactly the three named helpers.' }
. (Join-Path $PSScriptRoot 'delivery-lib.ps1')
$null = Assert-PlainFile $contractFull
$runtimePath = [Diagnostics.Process]::GetCurrentProcess().MainModule.FileName
if (-not [string]::Equals((Get-LocalPath $contract.powerShellRuntime.path), (Get-LocalPath $runtimePath), [StringComparison]::OrdinalIgnoreCase) -or
    $PSVersionTable.PSVersion.ToString() -cne [string]$contract.powerShellRuntime.version) { throw 'Reviewed exact PowerShell runtime path/version differs.' }
$null = Assert-Pin $runtimePath $contract.powerShellRuntime.sha256
$repository = Assert-PlainDirectory $contract.repositoryRoot
$candidate = Assert-PlainDirectory $contract.candidateDirectory
$evidence = Get-LocalPath $contract.evidenceDirectory
$version = [string]$contract.version
if ($version -cnotmatch '^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$') { throw 'Candidate version is invalid.' }
if ((Test-Within $evidence $candidate) -or (Test-Within $candidate $evidence) -or (Test-Within $candidate $PSScriptRoot) -or
    (Test-Within $evidence $PSScriptRoot)) { throw 'Evidence, candidate, and prepared helpers must be distinct non-nested roots.' }
$evidenceRelative = [IO.Path]::GetRelativePath($repository, $evidence).Replace('\','/')
if ($evidenceRelative -cnotmatch '^native/(artifacts/acceptance|handoffs)/autonomous-local-windows-[A-Za-z0-9._+-]+$') {
    throw 'Evidence must use a fresh autonomous-local-windows namespace under this repository.'
}
$candidateRelative = [IO.Path]::GetRelativePath($repository, $candidate).Replace('\','/')
if ($candidateRelative -cnotmatch '^native/bin/[A-Za-z0-9._+-]+$' -or
    $candidateRelative -cmatch '^native/bin/levnote-native-worker-20261110\.(29|30)$' -or
    $version -cin @('native-worker.20261110.29','native-worker.20261110.30')) { throw 'Fresh candidate required; accepted .29 and closed .30 are never executed.' }
Assert-PlainChain ([IO.Path]::GetDirectoryName($evidence))
if (Test-Path -LiteralPath $evidence) { throw 'Evidence namespace already exists; no rerun or overwrite.' }
$normalTemp = Assert-PlainDirectory ([IO.Path]::GetTempPath())
if (-not [IO.Path]::GetPathRoot($normalTemp).Equals('C:\', [StringComparison]::OrdinalIgnoreCase)) { throw 'Normal default TEMP must be on ordinary local C:.' }
$configRoot = Join-Path $normalTemp ('levnote-native-autonomous-local-' + [Guid]::NewGuid().ToString('N'))
if ((Test-Within $configRoot $candidate) -or (Test-Within $configRoot $evidence)) { throw 'Config must be isolated from bundle/evidence.' }
$settle = [int]$contract.settlingMilliseconds
$ready = [int]$contract.readinessMilliseconds
$exitBound = [int]$contract.closeExitMilliseconds
if ($settle -lt 5000 -or $settle -gt 10000 -or $ready -lt 1000 -or $ready -gt 20000 -or
    $exitBound -lt 1000 -or $exitBound -gt 15000) { throw 'Readiness/settling/close bounds differ from reviewed limits.' }
if ($contract.expectedWindowTitle -cne '轻笺 · 原生版') { throw 'Reviewed native title differs.' }
$ownedHost = $null
$baseline = $null
$configCreated = $false
$errors = [Collections.Generic.List[object]]::new()
$guards = [Collections.Generic.List[object]]::new()
$started = [DateTime]::UtcNow
$watch = [Diagnostics.Stopwatch]::StartNew()
$result = [ordered]@{
    schema=1; status='FAIL'; scope='One exact copied hidden Windows host; no physical UI/IME/save/restore/worker/resource acceptance.'
    startedAtUtc=$started.ToString('o'); version=$version; repositoryRoot=$repository
    inputContractPath=$contractFull; inputContractSHA256=$InputContractSHA256.ToUpperInvariant()
    powerShellRuntimePath=$runtimePath; powerShellRuntimeVersion=$PSVersionTable.PSVersion.ToString()
    powerShellRuntimeSHA256=$contract.powerShellRuntime.sha256.ToUpperInvariant()
    candidateDirectory=$candidate; evidenceDirectory=$evidence; configRoot=$configRoot; normalTemp=$normalTemp
    launchCount=0; readinessMilliseconds=$ready; settlingMillisecondsRequested=$settle
    settlingMillisecondsActual=0; settlingAliveChecks=0; settlingCompleted=$false
    strictPreCloseSession=$false; newerNormalClosePublication=$false; closeRequestedAtUtc=$null
    processIdentity=$null; window=$null; jobAfterResume=$null; jobBeforeClose=$null; jobAfterExit=$null
    jobAfterCleanup=$null; cleanupForced=$false; cleanupLiveProcess=$null; appExitCode=$null
    wmClosePostAttempts=0; wmClosePosted=$false; handlesDisposed=$false; configRetained=$false
    preCloseSessionFingerprint=$null; postCloseSessionFingerprint=$null
    preCloseSessionFacts=$null; postCloseSessionFacts=$null
}
function Add-RunError([string]$Phase, [object]$Failure) {
    $errors.Add([pscustomobject]@{phase=$Phase; atUtc=[DateTime]::UtcNow.ToString('o'); message=[string]$Failure.Exception.Message; detail=$Failure.Exception.ToString(); scriptStackTrace=[string]$Failure.ScriptStackTrace})
}
function Invoke-Guard([string]$Name, [scriptblock]$Action) {
    try { $facts = & $Action; $guards.Add([pscustomobject]@{name=$Name; pass=$true; facts=$facts}) }
    catch { $guards.Add([pscustomobject]@{name=$Name; pass=$false; error=$_.Exception.Message}); Add-RunError ('postguard:'+$Name) $_ }
}
$null = New-FreshDirectory $evidence
try {
    $pins = [Collections.Generic.List[object]]::new()
    $pins.Add((Assert-Pin $contractFull $InputContractSHA256))
    $pins.Add((Assert-Pin $runtimePath $contract.powerShellRuntime.sha256))
    foreach ($row in $helperPins) { $pins.Add((Assert-Pin (Join-Path $PSScriptRoot $row.name) $row.sha256)) }
    foreach ($key in @('sourceInventory','sourceAcceptance','rootReview','packageRecord','textFixture','pngFixture','diagramNotes')) {
        $pin = $contract.$key
        if (Test-Within $pin.path $evidence) { throw "Named input must preexist outside new evidence: $key" }
        $pins.Add((Assert-Pin $pin.path $pin.sha256))
    }
    $manifestPin = Assert-Pin (Join-Path $candidate 'release-manifest.json') $contract.packageManifestSHA256
    $pins.Add($manifestPin)
    $sourceManifest = Read-LocalJson $contract.sourceInventory.path
    $source = @(Get-SourceInventory $repository $sourceManifest)
    $packageRecord = Read-LocalJson $contract.packageRecord.path
    $package = Assert-Package $candidate $version (Read-LocalJson $manifestPin.absolutePath) $packageRecord
    Assert-TextFixture $contract.textFixture.path
    Assert-PngFixture $contract.pngFixture.path
    $plugin = Read-LocalJson $contract.diagramNotes.path
    Assert-SessionKeys $plugin @('schema','id','name','version','apiVersion','extensions','nativeRoute') @('schema','id','name','version','apiVersion','extensions','nativeRoute') 'Reviewed local plugin'
    if ($plugin.schema -ne 3 -or $plugin.id -cne 'diagram-notes' -or $plugin.apiVersion -ne 1 -or
        @($plugin.extensions).Count -ne 1 -or $plugin.extensions[0] -cne '.diagramnote' -or
        $plugin.nativeRoute -cne 'builtin.drawio') { throw 'Reviewed data-only diagram-notes plugin differs.' }
    $bundle = New-FreshDirectory (Join-Path $evidence 'bundle')
    $workspace = New-FreshDirectory (Join-Path $evidence 'workspace')
    Copy-ExactTree $candidate $bundle @($package.files)
    $textPath = Join-Path $workspace 'route-gate.diagramnote'
    $imagePath = Join-Path $workspace 'image-fixture.png'
    [IO.File]::Copy($contract.textFixture.path, $textPath, $false)
    [IO.File]::Copy($contract.pngFixture.path, $imagePath, $false)
    Assert-ExactFingerprint (Get-Fingerprint $contract.textFixture.path) (Get-Fingerprint $textPath) $false
    Assert-ExactFingerprint (Get-Fingerprint $contract.pngFixture.path) (Get-Fingerprint $imagePath) $false
    $null = New-FreshDirectory $configRoot
    $configCreated = $true
    $roaming = New-FreshDirectory (Join-Path $configRoot 'appdata/Roaming')
    $local = New-FreshDirectory (Join-Path $configRoot 'appdata/Local')
    $profile = New-FreshDirectory (Join-Path $configRoot 'profile')
    $temp = New-FreshDirectory (Join-Path $configRoot 'temp')
    $pluginDirectory = New-FreshDirectory (Join-Path $roaming 'LevNoteNative/plugins')
    $installedPlugin = Join-Path $pluginDirectory 'diagram-notes.json'
    [IO.File]::Copy($contract.diagramNotes.path, $installedPlugin, $false)
    Assert-ExactFingerprint (Get-Fingerprint $contract.diagramNotes.path) (Get-Fingerprint $installedPlugin) $false
    $session = Join-Path $roaming 'LevNoteNative/session.json'
    if (Test-Path -LiteralPath $session) { throw 'Fresh isolated pre-run session must be absent.' }
    $exe = Join-Path $bundle 'levnote-native.exe'
    $baseline = [ordered]@{
        pins=@($pins.ToArray()); source=$source; candidate=@($package.files); packagePe=$package.pe
        bundle=@(Get-TreeInventory $bundle); workspace=@(Get-TreeInventory $workspace)
        installedPlugin=@(Get-TreeInventory $pluginDirectory); configBefore=@(Get-TreeInventory $configRoot)
        sessionPresentBeforeStart=$false; sessionPath=$session; cliArguments=@($textPath,$imagePath)
    }
    Write-NewJson (Join-Path $evidence 'inputs-before.json') $baseline
    $environment = [Collections.Generic.Dictionary[string,string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in [Environment]::GetEnvironmentVariables('Process').GetEnumerator()) {
        $name = [string]$entry.Key
        # Windows' hidden drive-current-directory entries start with '=' and are unnecessary for absolute CLI paths.
        if (-not $name.StartsWith('=')) { $environment[$name] = [string]$entry.Value }
    }
    $environment['APPDATA']=$roaming; $environment['LOCALAPPDATA']=$local
    $environment['USERPROFILE']=$profile; $environment['HOME']=$profile
    $environment['HOMEDRIVE']='C:'; $environment['HOMEPATH']=$profile.Substring(2)
    $environment['TEMP']=$temp; $environment['TMP']=$temp
    $environment['XDG_CONFIG_HOME']=$roaming; $environment['XDG_CACHE_HOME']=$local
    $environment['GOTELEMETRY']='off'
    $environmentRows = @(foreach ($entry in $environment.GetEnumerator()) { $entry.Key + '=' + $entry.Value })
    Write-NewJson (Join-Path $evidence 'launch-intent.json') ([ordered]@{
        executable=$exe; executableFingerprint=(Get-Fingerprint $exe); arguments=@($textPath,$imagePath)
        workingDirectory=$workspace; windowTitle=$contract.expectedWindowTitle
        method='CreateProcessW CREATE_SUSPENDED + EXTENDED_STARTUPINFO_PRESENT; assign/verify non-inheritable KILL_ON_JOB_CLOSE Job before ResumeThread'
        envOverrides=[ordered]@{APPDATA=$roaming; LOCALAPPDATA=$local; USERPROFILE=$profile; HOME=$profile; TEMP=$temp; TMP=$temp; GOTELEMETRY='off'}
        stdout=(Join-Path $evidence 'app.stdout.log'); stderr=(Join-Path $evidence 'app.stderr.log')
    })
    $stdin = Join-Path $evidence 'app.stdin.empty'
    $stream = [IO.File]::Open($stdin, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::Read)
    $stream.Dispose()
    Add-Type -TypeDefinition ([IO.File]::ReadAllText((Join-Path $PSScriptRoot 'owned-host.cs'))) -Language CSharp
    $ownedHost = [LevNoteLocalAcceptance.OwnedHost]::new()
    $ownedHost.Start($exe, [string[]]@($textPath,$imagePath), $workspace, [string[]]$environmentRows,
        $stdin, (Join-Path $evidence 'app.stdout.log'), (Join-Path $evidence 'app.stderr.log'))
    $result.launchCount=$ownedHost.StartAttempts
    $result.processIdentity=[ordered]@{
        pid=$ownedHost.ProcessId; imagePath=$ownedHost.ImagePath; startedAtUtc=$ownedHost.StartUtc.ToString('o')
        heldProcessHandle=$ownedHost.HeldProcessHandle; heldJobHandle=$ownedHost.HeldJobHandle
        createdSuspended=$ownedHost.CreatedSuspended; assignedBeforeResume=$ownedHost.AssignedBeforeResume; resumed=$ownedHost.Resumed
    }
    Write-NewJson (Join-Path $evidence 'process-receipt.json') $result.processIdentity
    $result.jobAfterResume=$ownedHost.Accounting()
    $readyWatch=[Diagnostics.Stopwatch]::StartNew()
    $window=$null
    while ($readyWatch.ElapsedMilliseconds -lt $ready) {
        if (-not $ownedHost.IsAlive()) { throw 'Host exited before native-window readiness.' }
        $windows=@($ownedHost.Windows($contract.expectedWindowTitle))
        if ($windows.Count -gt 1) { throw 'Multiple exact-title windows for held host.' }
        if ($windows.Count -eq 1) { $window=$windows[0]; break }
        Start-Sleep -Milliseconds 100
    }
    if ($null -eq $window -or $window.Visible) { throw 'Exactly one hidden owned native window not observed within bound.' }
    $result.window=$window
    $settleWatch=[Diagnostics.Stopwatch]::StartNew()
    while ($settleWatch.ElapsedMilliseconds -lt $settle) {
        if (-not $ownedHost.IsAlive()) { throw 'Host exited during bounded alive settling.' }
        $accounting=$ownedHost.Accounting()
        if ($accounting.TotalProcesses -ne 1 -or $accounting.ActiveProcesses -ne 1) { throw 'Unexpected child/worker during settling.' }
        $result.settlingAliveChecks++
        Start-Sleep -Milliseconds ([Math]::Min(100, [Math]::Max(1, $settle - [int]$settleWatch.ElapsedMilliseconds)))
    }
    if (-not $ownedHost.IsAlive()) { throw 'Host exited at settling boundary.' }
    $result.settlingMillisecondsActual=$settleWatch.Elapsed.TotalMilliseconds
    $result.settlingCompleted=$true
    $beforeSession=Get-SessionCopy $session (Join-Path $evidence 'session-before-close.json')
    $beforeFacts=Get-CleanSession (Join-Path $evidence 'session-before-close.json') $textPath $imagePath
    if ($beforeSession.lastWriteUtcTicks -lt $ownedHost.StartUtc.Ticks) { throw 'Pre-close autosave predates exact process.' }
    $result.preCloseSessionFingerprint=$beforeSession; $result.preCloseSessionFacts=$beforeFacts
    $result.strictPreCloseSession=$true
    Write-NewJson (Join-Path $evidence 'session-before-close-facts.json') ([ordered]@{
        capturedAtUtc=[DateTime]::UtcNow.ToString('o'); capturedBeforeWMClose=$true
        fingerprint=$beforeSession; facts=$beforeFacts; processAlive=$ownedHost.IsAlive()
    })
    $result.jobBeforeClose=$ownedHost.Accounting()
    if ($result.jobBeforeClose.TotalProcesses -ne 1 -or $result.jobBeforeClose.ActiveProcesses -ne 1) { throw 'Job not exact single live host before close.' }
    $closeTicks=[DateTime]::UtcNow.Ticks
    $result.closeRequestedAtUtc=[DateTime]::new($closeTicks,[DateTimeKind]::Utc).ToString('o')
    # No fallback WM_CLOSE: strict pre-close gate above is mandatory, including on failures.
    $ownedHost.PostOwnedClose($window.Handle, $contract.expectedWindowTitle)
    if (-not $ownedHost.WaitExit($exitBound)) { throw 'Normal owned close exit bound exceeded.' }
    $result.appExitCode=$ownedHost.ExitCode()
    if ($result.appExitCode -ne 0) { throw "Host exited nonzero: $($result.appExitCode)" }
    $result.jobAfterExit=$ownedHost.Accounting()
    if ($result.jobAfterExit.TotalProcesses -ne 1 -or $result.jobAfterExit.ActiveProcesses -ne 0) { throw 'Job not exact total1/active0 after normal exit.' }
    $afterSession=Get-SessionCopy $session (Join-Path $evidence 'session-after-close.json')
    $afterFacts=Get-CleanSession (Join-Path $evidence 'session-after-close.json') $textPath $imagePath
    $result.postCloseSessionFingerprint=$afterSession; $result.postCloseSessionFacts=$afterFacts
    if ($afterSession.lastWriteUtcTicks -le $beforeSession.lastWriteUtcTicks -or
        $afterSession.lastWriteUtcTicks -lt $closeTicks -or $afterSession.lastWriteUtcTicks -lt $ownedHost.StartUtc.Ticks) {
        throw 'Normal-close session publication is not newer than pre-close and after owned close request.'
    }
    $result.newerNormalClosePublication=$true
    Write-NewJson (Join-Path $evidence 'session-after-close-facts.json') ([ordered]@{fingerprint=$afterSession; facts=$afterFacts; newerAfterWMClose=$true})
} catch { Add-RunError 'run' $_ }
finally {
    if ($null -ne $ownedHost) {
        try {
            $ownedHost.ForceCleanup(10000)
            if ($ownedHost.HasProcess) {
                $result.cleanupLiveProcess=$ownedHost.IsAlive()
                if (-not $result.cleanupLiveProcess) { $result.appExitCode=$ownedHost.ExitCode() }
            }
            $result.jobAfterCleanup=$ownedHost.Accounting()
            if ($result.jobAfterCleanup.ActiveProcesses -ne 0) { throw 'Owned Job still active after cleanup.' }
        } catch { Add-RunError 'owned-cleanup' $_ }
        $result.launchCount=$ownedHost.StartAttempts
        if ($null -eq $result.processIdentity -and $ownedHost.HasProcess) {
            $result.processIdentity=[ordered]@{
                pid=$ownedHost.ProcessId; imagePath=$ownedHost.ImagePath; startedAtUtc=$ownedHost.StartUtc.ToString('o')
                heldProcessHandle=$ownedHost.HeldProcessHandle; heldJobHandle=$ownedHost.HeldJobHandle
                createdSuspended=$ownedHost.CreatedSuspended; assignedBeforeResume=$ownedHost.AssignedBeforeResume; resumed=$ownedHost.Resumed
                receiptPhase='failed-start cleanup'
            }
        }
        $result.cleanupForced=$ownedHost.ForcedCleanup
        $result.wmClosePostAttempts=$ownedHost.ClosePostAttempts; $result.wmClosePosted=$ownedHost.ClosePosted
        try { $ownedHost.Dispose(); $result.handlesDisposed=$ownedHost.JobClosed -and $ownedHost.ProcessClosed }
        catch { Add-RunError 'handle-disposal' $_ }
    }
    if ($null -ne $baseline) {
        Invoke-Guard 'named-inputs-and-helpers' {
            $facts=@(foreach ($pin in $baseline.pins) { Get-Fingerprint $pin.absolutePath })
            for ($i=0; $i -lt $facts.Count; $i++) { Assert-ExactFingerprint $baseline.pins[$i] $facts[$i] }
            Write-NewJson (Join-Path $evidence 'pins-after.json') $facts
            return $facts
        }
        Invoke-Guard 'complete-frozen-source' {
            $facts=@(Get-SourceInventory $repository (Read-LocalJson $contract.sourceInventory.path))
            Assert-ExactInventory @($baseline.source) $facts 'Source during actual host'
            Write-NewJson (Join-Path $evidence 'source-after.json') $facts
            return [ordered]@{count=$facts.Count; unchanged=$true}
        }
        Invoke-Guard 'candidate-package' {
            $facts=@(Get-TreeInventory $candidate); Write-NewJson (Join-Path $evidence 'candidate-after.json') $facts
            Assert-ExactInventory @($baseline.candidate) $facts 'Candidate package during host'
            return [ordered]@{count=$facts.Count; unchanged=$true}
        }
        Invoke-Guard 'executed-bundle' {
            $facts=@(Get-TreeInventory $bundle); Write-NewJson (Join-Path $evidence 'bundle-after.json') $facts
            Assert-ExactInventory @($baseline.bundle) $facts 'Executed copied bundle'
            return [ordered]@{count=$facts.Count; unchanged=$true}
        }
        Invoke-Guard 'ordered-fixtures' {
            $facts=@(Get-TreeInventory $workspace); Write-NewJson (Join-Path $evidence 'workspace-after.json') $facts
            Assert-ExactInventory @($baseline.workspace) $facts 'Copied fixture workspace'
            return [ordered]@{count=$facts.Count; unchanged=$true}
        }
        Invoke-Guard 'installed-plugin-directory' {
            $facts=@(Get-TreeInventory $pluginDirectory); Write-NewJson (Join-Path $evidence 'plugin-after.json') $facts
            Assert-ExactInventory @($baseline.installedPlugin) $facts 'Installed plugin inventory'
            return [ordered]@{count=$facts.Count; unchanged=$true}
        }
    }
    if ($configCreated) {
        $result.configRetained=$true
        Invoke-Guard 'retained-isolated-config' {
            $facts=@(Get-TreeInventory $configRoot); Write-NewJson (Join-Path $evidence 'config-after.json') $facts
            return [ordered]@{count=$facts.Count; retained=$true; ordinaryCVolume=$true}
        }
    }
    $watch.Stop()
    $result.finishedAtUtc=[DateTime]::UtcNow.ToString('o'); $result.wallSeconds=$watch.Elapsed.TotalSeconds
    $result.streams=@()
    Invoke-Guard 'retained-owned-streams' {
        $streamFacts=@(foreach ($name in @('app.stdin.empty','app.stdout.log','app.stderr.log')) {
            $path=Join-Path $evidence $name
            if (Test-Path -LiteralPath $path -PathType Leaf) { Get-Fingerprint $path $name }
        })
        $result.streams=$streamFacts
        if ($result.processIdentity -and $streamFacts.Count -ne 3) { throw 'Owned started host stream set incomplete.' }
        return $streamFacts
    }
    $result.errors=@($errors.ToArray()); $result.postguards=@($guards.ToArray())
    $result.status=$(if ($errors.Count -eq 0 -and $result.launchCount -eq 1 -and $result.strictPreCloseSession -and
        $result.newerNormalClosePublication -and $result.wmClosePostAttempts -eq 1 -and $result.wmClosePosted -and
        $result.appExitCode -eq 0 -and -not $result.cleanupForced -and $result.handlesDisposed) { 'PASS' } else { 'FAIL' })
    Write-NewJson (Join-Path $evidence 'result.json') $result
}
$result | ConvertTo-Json -Depth 8
if ($result.status -cne 'PASS') { exit 1 }
exit 0
