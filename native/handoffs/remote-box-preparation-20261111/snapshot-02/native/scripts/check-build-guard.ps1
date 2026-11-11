#requires -Version 7.0
param(
    [Parameter(Mandatory = $true)][string]$EvidenceDirectory,
    [ValidateSet('All', 'ContextAndRefresh')][string]$CaseSet = 'All'
)
$ErrorActionPreference = 'Stop'
$nativeRoot = [IO.Path]::GetFullPath((Split-Path $PSScriptRoot -Parent))
$evidenceRoot = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($EvidenceDirectory)
. (Join-Path $PSScriptRoot 'portable-package-lib.ps1')
if (Test-Path -LiteralPath $evidenceRoot) { throw 'Guard evidence directory must be fresh.' }
Assert-PlainDirectoryChain (Split-Path $evidenceRoot -Parent)
New-Item -ItemType Directory -Path $evidenceRoot -ErrorAction Stop | Out-Null
$probeRoot = Join-Path $evidenceRoot 'guard fixture with spaces'
$probeScripts = Join-Path $probeRoot 'scripts'
$probeVendor = Join-Path $probeRoot 'vendor\github.com\egoist\mygo'
New-Item -ItemType Directory -Path $probeScripts, $probeVendor | Out-Null
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'go.ps1') -Destination (Join-Path $probeScripts 'go.ps1')
# Only this isolated fixture uses a junction to the verified project toolchain;
# no production vendor/module is edited, downloaded or removed.
New-Item -ItemType Junction -Path (Join-Path $probeRoot 'tools') -Target (Join-Path $nativeRoot 'tools') | Out-Null
$probeWrapper = Join-Path $probeScripts 'go.ps1'
$probeSource = Join-Path $probeVendor 'app.go'
$probeManifest = Join-Path $probeRoot 'vendor\modules.txt'
$sourcePatchManifest = Join-Path $nativeRoot 'dependency-patches/mygo-v0.8.3-patch-files.json'
$probePatchManifest = Join-Path $probeRoot 'dependency-patches/mygo-v0.8.3-patch-files.json'
$marker = '// LEVNOTE PATCH: this app does not enable MyGo updates'
$module = "module example.local/buildguard`n`ngo 1.27.3`n`nrequire github.com/egoist/mygo v0.8.3`n"
[IO.File]::WriteAllText((Join-Path $probeRoot 'go.mod'), $module)
[IO.File]::WriteAllText((Join-Path $probeRoot 'main.go'), "package main`nfunc main() {}`n")
[IO.File]::WriteAllText((Join-Path $probeRoot 'main_test.go'), "package main`nimport `"testing`"`nfunc TestFixture(t *testing.T) {}`n")
$shellPath = (Get-Command pwsh -ErrorAction Stop).Source
$cases = [Collections.Generic.List[object]]::new()
function Quote-Literal([string]$Value) { "'" + $Value.Replace("'", "''") + "'" }
function Set-ValidFixture {
    [IO.File]::WriteAllText($probeManifest, "# github.com/egoist/mygo v0.8.3`n## explicit; go 1.27.3`n")
    $manifest = Get-Content -LiteralPath $sourcePatchManifest -Raw | ConvertFrom-Json
    foreach ($file in $manifest.files) {
        $destination = Join-Path $probeRoot $file.path
        New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $nativeRoot $file.path) -Destination $destination -Force
    }
    Copy-Item -LiteralPath $sourcePatchManifest -Destination $probePatchManifest -Force
}
function Invoke-GuardProbe([string]$Name, [string[]]$Arguments, [bool]$MustReject, [hashtable]$Environment = @{}, [string]$Artifact = '', [hashtable]$ExpectedEnvironment = @{}, [string]$RejectMessage = 'MyGo startup-safety patch is missing or bypassed') {
    $caseRoot = Join-Path $evidenceRoot $Name
    New-Item -ItemType Directory -Path $caseRoot -ErrorAction Stop | Out-Null
    $runner = Join-Path $caseRoot 'run.ps1'
    $body = @("`$ErrorActionPreference = 'Stop'")
    foreach ($variable in @('GOFLAGS', 'GOWORK', 'GOENV')) { $body += "[Environment]::SetEnvironmentVariable('$variable',`$null,'Process')" }
    $body += "`$env:GOOS='windows'`n`$env:GOARCH='amd64'`n`$env:GOAMD64='v1'"
    foreach ($entry in $Environment.GetEnumerator()) { $body += "[Environment]::SetEnvironmentVariable(" + (Quote-Literal $entry.Key) + ',' + (Quote-Literal $entry.Value) + ",'Process')" }
    $literalArgs = @($Arguments | ForEach-Object { Quote-Literal $_ }) -join ','
    $body += '& ' + (Quote-Literal $probeWrapper) + ' -GoArguments @(' + $literalArgs + ')'
    $body += 'exit $LASTEXITCODE'
    [IO.File]::WriteAllText($runner, ($body -join "`n"), [Text.UTF8Encoding]::new($false))
    $stdout = Join-Path $caseRoot 'stdout.log'; $stderr = Join-Path $caseRoot 'stderr.log'
    $start = [DateTime]::UtcNow; $watch = [Diagnostics.Stopwatch]::StartNew()
    $process = Start-Process -FilePath $shellPath -ArgumentList @('-NoProfile', '-File', ('"' + $runner + '"')) -WorkingDirectory $probeRoot -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -Wait -PassThru
    $watch.Stop()
    $out = [IO.File]::ReadAllText($stdout); $err = [IO.File]::ReadAllText($stderr)
    $passed = if ($MustReject) { $process.ExitCode -ne 0 -and ($out + $err).Contains($RejectMessage) } else { $process.ExitCode -eq 0 }
    $artifactInfo = $null
    if ($Artifact) {
        $exists = Test-Path -LiteralPath $Artifact -PathType Leaf
        $passed = $passed -and $exists
        if ($exists) { $artifactInfo = [ordered]@{ Path = $Artifact; Bytes = (Get-Item -LiteralPath $Artifact).Length; SHA256 = (Get-FileHash -LiteralPath $Artifact -Algorithm SHA256).Hash }; $passed = $passed -and $artifactInfo.Bytes -gt 0 }
    }
    $outputValidation = $null
    if ($ExpectedEnvironment.Count -gt 0) {
        try {
            $reported = $out | ConvertFrom-Json -AsHashtable -ErrorAction Stop
            $passed = $passed -and $reported.Count -eq $ExpectedEnvironment.Count
            foreach ($entry in $ExpectedEnvironment.GetEnumerator()) {
                $passed = $passed -and ($reported.Keys -ccontains $entry.Key) -and $reported[$entry.Key] -is [string] -and $reported[$entry.Key] -ceq $entry.Value
            }
        } catch { $passed = $false; $outputValidation = $_.Exception.Message }
    }
    $record = [ordered]@{ Name = $Name; GoArguments = $Arguments; Environment = $Environment; ExpectedEnvironment = $ExpectedEnvironment; OutputValidationError = $outputValidation; MustReject = $MustReject; StartedAtUtc = $start.ToString('o'); FinishedAtUtc = [DateTime]::UtcNow.ToString('o'); WallSeconds = $watch.Elapsed.TotalSeconds; ExitCode = $process.ExitCode; Passed = $passed; StdoutPath = $stdout; StderrPath = $stderr; StdoutBytes = (Get-Item -LiteralPath $stdout).Length; StderrBytes = (Get-Item -LiteralPath $stderr).Length; Artifact = $artifactInfo }
    $record | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $caseRoot 'result.json') -Encoding utf8
    $cases.Add($record)
    Write-Output "$Name passed=$passed exit=$($process.ExitCode)"
    if (-not $passed) { throw "Guard fixture failed: $Name; actual streams retained at $caseRoot" }
}
$failure = $null
try {
    if ($CaseSet -eq 'All') {
    foreach ($command in @('build', 'run', 'install', 'test')) { Invoke-GuardProbe ('missing-vendor-' + $command) @($command) $true }
    [IO.File]::WriteAllText($probeSource, "package mygo`n$marker`n")
    Invoke-GuardProbe 'missing-vendor-manifest' @('build') $true
    Set-ValidFixture
    [IO.File]::WriteAllText($probeSource, "package mygo`n$marker`nfunc start(){go cleanUpdateLeftovers()}`n")
    Invoke-GuardProbe 'unsafe-startup' @('build') $true
    [IO.File]::WriteAllText($probeSource, 'package mygo')
    Invoke-GuardProbe 'missing-patch' @('build') $true
    Set-ValidFixture
    [IO.File]::WriteAllText($probeManifest, '# github.com/egoist/mygo v0.8.4')
    Invoke-GuardProbe 'wrong-vendor-version' @('build') $true
    Set-ValidFixture
    $retirementError = 'MyGo bitmap-disposal patch is incomplete or changed'
    Remove-Item -LiteralPath $probePatchManifest
    Invoke-GuardProbe 'missing-disposal-manifest' @('build') $true -RejectMessage $retirementError
    Set-ValidFixture
    [IO.File]::AppendAllText($probePatchManifest, ' ')
    Invoke-GuardProbe 'changed-disposal-manifest' @('test') $true -RejectMessage $retirementError
    Set-ValidFixture
    foreach ($entry in @(
        @{Name='public-bitmap';Path='vendor/github.com/egoist/mygo/ui/bitmap.go'},
        @{Name='d3d11';Path='vendor/github.com/egoist/mygo/internal/gpu/d3d11/d3d11.go'},
        @{Name='metal';Path='vendor/github.com/egoist/mygo/internal/gpu/metal/present.go'},
        @{Name='gl';Path='vendor/github.com/egoist/mygo/internal/gpu/gl/gl.go'},
        @{Name='focused-tests';Path='vendor/github.com/egoist/mygo/internal/gpu/retirement_test.go'},
        @{Name='disposal-patch';Path='dependency-patches/mygo-v0.8.3-bitmap-dispose.patch'},
        @{Name='startup-patch';Path='dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch'}
    )) {
        [IO.File]::AppendAllText((Join-Path $probeRoot $entry.Path), ' ')
        Invoke-GuardProbe ('changed-'+$entry.Name) @('build') $true -RejectMessage $retirementError
        Set-ValidFixture
    }
    Remove-Item -LiteralPath (Join-Path $probeRoot 'vendor/github.com/egoist/mygo/internal/scene/retirement.go')
    Invoke-GuardProbe 'missing-retirement-source' @('build') $true -RejectMessage $retirementError
    Set-ValidFixture
    foreach ($entry in @(
        @{Name='uppercase-C';Args=@('-C',$nativeRoot,'build')},
        @{Name='uppercase-C-equals';Args=@(('-C='+$nativeRoot),'build')},
        @{Name='double-C';Args=@(('--C='+$nativeRoot),'build')},
        @{Name='modfile';Args=@('build','-modfile','alternate.mod')},
        @{Name='modfile-equals';Args=@('build','--modfile=alternate.mod')},
        @{Name='overlay';Args=@('build','-overlay','alternate.json')},
        @{Name='overlay-equals';Args=@('test','--overlay=alternate.json')},
        @{Name='mod-mod';Args=@('build','-mod=mod')},
        @{Name='mod-readonly';Args=@('build','--mod=readonly')},
        @{Name='mod-separate';Args=@('build','-mod','mod')}
    )) { Invoke-GuardProbe ('args-' + $entry.Name) $entry.Args $true }
    foreach ($entry in @(
        @{Name='module';Value='-mod=mod'}, @{Name='double-module';Value='--mod=readonly'},
        @{Name='quoted-module';Value='"-mod=mod"'}, @{Name='modfile';Value='-modfile=alternate.mod'},
        @{Name='quoted-modfile';Value="'--modfile=alternate path.mod'"},
        @{Name='overlay';Value='-overlay=alternate.json'}, @{Name='quoted-overlay';Value='"--overlay=alternate file.json"'},
        @{Name='uppercase-C';Value='-C=elsewhere'}
    )) { Invoke-GuardProbe ('env-' + $entry.Name) @('test') $true @{GOFLAGS=$entry.Value} }
    $tester = Join-Path $probeRoot 'fixture tester.exe'
    Invoke-GuardProbe 'lowercase-test-c' @('test','-c','-o',$tester,'.') $false @{} $tester
    $normalExe = Join-Path $probeRoot 'ordinary build.exe'
    Invoke-GuardProbe 'ordinary-build-vendor' @('build','-mod=vendor','-o',$normalExe,'.') $false @{} $normalExe
    $flagsExe = Join-Path $probeRoot 'quoted flags.exe'
    Invoke-GuardProbe 'quoted-useful-GOFLAGS' @('build','-o',$flagsExe,'.') $false @{GOFLAGS='"-ldflags=-s -w" -mod=vendor'} $flagsExe
    }
    Set-ValidFixture
    # These inputs would be effective and invalid in ordinary Go. The wrapper
    # must actually emit output while isolating them, not merely report env off.
    $work = Join-Path $probeRoot 'external go.work'
    [IO.File]::WriteAllText($work, "go 1.27.3`nuse ./missing-external-module`n")
    $persisted = Join-Path $probeRoot 'persisted go env'
    [IO.File]::WriteAllText($persisted, "GOFLAGS=-modfile=does-not-exist.mod`n")
    if ($CaseSet -eq 'All') {
        $isolatedExe = Join-Path $probeRoot 'isolated build.exe'
        Invoke-GuardProbe 'workspace-persisted-config-isolated' @('build','-o',$isolatedExe,'.') $false @{GOWORK=$work;GOENV=$persisted} $isolatedExe
    }
    # GOENV=off disables the configuration file; Go reports its effective path
    # as empty. GOWORK retains the literal off value. Check named JSON fields.
    Invoke-GuardProbe 'effective-context-off' @('env','-json','GOWORK','GOENV') $false @{GOWORK=$work;GOENV=$persisted} '' @{GOWORK='off';GOENV=''}
    [IO.File]::WriteAllText($probeSource, 'package mygo')
    Invoke-GuardProbe 'refresh-unpatched-mod-edit' @('mod','edit','-go=1.27.3') $false
} catch { $failure = $_.Exception.Message }
$summary = [ordered]@{ RecordedAtUtc = [DateTime]::UtcNow.ToString('o'); Status = $(if ($failure) {'FAIL'} else {'PASS'}); CaseSet = $CaseSet; Fixture = $probeRoot; GuardCopySHA256 = (Get-FileHash -LiteralPath $probeWrapper -Algorithm SHA256).Hash; SourceGuardSHA256 = (Get-FileHash -LiteralPath (Join-Path $PSScriptRoot 'go.ps1') -Algorithm SHA256).Hash; Passed = @($cases | Where-Object {$_.Passed}).Count; Total = $cases.Count; Cases = $cases; Error = $failure; Scope = 'Tiny isolated fixture modules and actual pinned toolchain; no production vendor mutation or product execution.' }
$summary | ConvertTo-Json -Depth 7 | Set-Content -LiteralPath (Join-Path $evidenceRoot 'result.json') -Encoding utf8
if ($failure) { throw $failure }
