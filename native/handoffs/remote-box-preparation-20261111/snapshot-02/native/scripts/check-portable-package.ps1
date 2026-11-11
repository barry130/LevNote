#requires -Version 7.0
param([Parameter(Mandatory = $true)][string]$EvidenceDirectory)
$ErrorActionPreference = 'Stop'
$nativeRoot = [IO.Path]::GetFullPath((Split-Path $PSScriptRoot -Parent))
$root = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($EvidenceDirectory)
. (Join-Path $PSScriptRoot 'portable-package-lib.ps1')
if (Test-Path -LiteralPath $root) { throw 'Package fixture evidence must be fresh.' }
Assert-PlainDirectoryChain (Split-Path $root -Parent)
New-Item -ItemType Directory -Path $root -ErrorAction Stop | Out-Null
$checks = [Collections.Generic.List[object]]::new()
function Check([string]$Name, [bool]$Pass, [string]$Detail) {
    $checks.Add([ordered]@{ Name=$Name; Passed=$Pass; Detail=$Detail })
    if (-not $Pass) { throw "Portable package check failed: $Name" }
}
function Reject([string]$Name, [scriptblock]$Action, [string]$Expected) {
    $caught = ''
    try { & $Action } catch { $caught = $_.Exception.Message }
    Check $Name ($caught.Contains($Expected)) $caught
}
$errorText = $null
try {
    $out = Join-Path $root 'fresh output with spaces'
    $proof = Join-Path $root 'fresh evidence with spaces'
    Assert-NewPackageDirectories $out $proof
    Check 'fresh-distinct-space-paths' $true 'Both parents validated; no directories created by preflight.'
    Check 'preflight-no-side-effects' (-not(Test-Path -LiteralPath $out) -and -not(Test-Path -LiteralPath $proof)) 'Preflight kept both destinations absent.'
    Reject 'same-directory' { Assert-NewPackageDirectories $out $out } 'distinct and non-nested'
    Reject 'evidence-inside-output' { Assert-NewPackageDirectories $out (Join-Path $out 'evidence') } 'distinct and non-nested'
    Reject 'output-inside-evidence' { Assert-NewPackageDirectories (Join-Path $proof 'output') $proof } 'distinct and non-nested'
    $existing = Join-Path $root 'historical output'
    New-Item -ItemType Directory -Path $existing | Out-Null
    $sentinel = Join-Path $existing 'keep.txt'
    [IO.File]::WriteAllText($sentinel, 'historical keep')
    $sentinelHash = (Get-FileHash -LiteralPath $sentinel -Algorithm SHA256).Hash
    Reject 'existing-output' { Assert-NewPackageDirectories $existing $proof } 'must be fresh'
    Reject 'existing-evidence' { Assert-NewPackageDirectories $out $existing } 'must be fresh'
    $asFile = Join-Path $root 'existing file'
    [IO.File]::WriteAllText($asFile, 'keep-file')
    Reject 'existing-file' { Assert-NewPackageDirectories $asFile $proof } 'must be fresh'
    Reject 'missing-parent' { Assert-NewPackageDirectories (Join-Path $root 'absent\child') $proof } 'does not exist'
    $junction = Join-Path $root 'redirecting parent'
    New-Item -ItemType Junction -Path $junction -Target $existing | Out-Null
    Reject 'ancestor-junction' { Assert-NewPackageDirectories (Join-Path $junction 'child') $proof } 'not reparse points'
    Check 'historical-sentinel-preserved' ((Get-FileHash -LiteralPath $sentinel -Algorithm SHA256).Hash-eq $sentinelHash -and -not(Test-Path -LiteralPath (Join-Path $existing 'child'))) 'No existing destination or junction target mutated.'

    # Invoke an exact copy of the top-level script from a changed PS location.
    # The copy deliberately lacks inputs so a broken resolver still cannot
    # reach a production compilation before this negative oracle fails.
    $scriptCopy = Join-Path $root 'isolated source\native\scripts'
    New-Item -ItemType Directory -Path $scriptCopy | Out-Null
    foreach ($name in @('package-markdown-worker.ps1','portable-package-lib.ps1')) {
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination (Join-Path $scriptCopy $name)
    }
    # .NET cwd remains elsewhere. Existing relative evidence must be rejected
    # there, before any build/output; an incorrect GetFullPath resolves elsewhere.
    $different = Join-Path $root 'different starting cwd'
    New-Item -ItemType Directory -Path (Join-Path $different 'already evidence') | Out-Null
    $dotNetCwd = [Environment]::CurrentDirectory
    Push-Location $different
    try {
        Check 'different-ps-location' (-not [string]::Equals([IO.Path]::GetFullPath('.'), (Get-Location).Path, [StringComparison]::OrdinalIgnoreCase)) 'PowerShell location deliberately differs from .NET cwd.'
        Reject 'real-script-relative-location' { & (Join-Path $scriptCopy 'package-markdown-worker.ps1') -OutputDirectory '.\new output' -EvidenceDirectory '.\already evidence' -Version 'fixture.1' } 'must be fresh'
        Check 'real-script-relative-no-output' (-not(Test-Path -LiteralPath '.\new output')) 'Rejected before creating relative output or invoking compiler.'
    } finally { Pop-Location }
    Check 'dotnet-cwd-unmodified' ([Environment]::CurrentDirectory-ceq $dotNetCwd) 'Test did not mutate process .NET cwd.'
    Reject 'missing-inputs-before-output' { & (Join-Path $scriptCopy 'package-markdown-worker.ps1') -OutputDirectory $out -EvidenceDirectory $proof -Version 'fixture.1' } 'does not exist'
    Check 'missing-inputs-no-destinations' (-not(Test-Path -LiteralPath $out) -and -not(Test-Path -LiteralPath $proof)) 'Incomplete source copy refuses before creating either directory.'

    Assert-PortableInputs $nativeRoot
    New-Item -ItemType Directory -Path $out | Out-Null
    # Asset-only fixture: payload bytes are explicit test data, never executed.
    New-Item -ItemType Directory -Path (Join-Path $out 'engines/markdown-export') | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $out 'engines/mindmap-svg') | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $out 'engines/mindmap-preview') | Out-Null
    [IO.File]::WriteAllText((Join-Path $out 'levnote-native.exe'), 'fixture host bytes, not executable')
    [IO.File]::WriteAllText((Join-Path $out 'engines/markdown-export/levnote-markdown-export.exe'), 'fixture worker bytes, not executable')
    [IO.File]::WriteAllText((Join-Path $out 'engines/markdown-export/manifest.json'), '{"fixture":true}')
    [IO.File]::WriteAllText((Join-Path $out 'engines/mindmap-svg/levnote-mindmap-svg.exe'), 'fixture worker bytes, not executable')
    [IO.File]::WriteAllText((Join-Path $out 'engines/mindmap-svg/manifest.json'), '{"fixture":true}')
    [IO.File]::WriteAllText((Join-Path $out 'engines/mindmap-preview/levnote-mindmap-preview.exe'), 'fixture worker bytes, not executable')
    [IO.File]::WriteAllText((Join-Path $out 'engines/mindmap-preview/manifest.json'), '{"fixture":true}')
    Write-PortableAssets $nativeRoot $out 'fixture.1'
    Write-PortableReleaseManifest $out 'fixture.1'
    Assert-PortableLocalLinks $out
    Check 'authored-local-links' $true 'Portable README/notices relative links resolve inside output, including release manifest.'
    $copies = @(Get-PortableAssetCopies $nativeRoot)
    $requiredAssetPaths = @(
        'LICENSE',
        'attribution/SOURCE_THIRD_PARTY_NOTICES.txt',
        'attribution/mygo-README.md',
        'attribution/purego-README.md',
        'attribution/mygo-v0.8.3-no-startup-cleanup.patch',
        'attribution/mygo-v0.8.3-bitmap-dispose.patch',
        'attribution/dependency-patches-README.md',
        'licenses/Go-BSD-3-Clause.txt',
        'licenses/go-text-typesetting.txt',
        'licenses/mygo-MIT.txt',
        'licenses/purego-Apache-2.0.txt',
        'licenses/x-image-BSD-3-Clause.txt',
        'licenses/x-net-BSD-3-Clause.txt',
        'licenses/x-text-BSD-3-Clause.txt'
    )
    $copyMismatch = @($copies | Where-Object {(Get-FileHash -LiteralPath $_.Source -Algorithm SHA256).Hash-cne (Get-FileHash -LiteralPath (Join-Path $out $_.Relative) -Algorithm SHA256).Hash})
    $copyPaths = @($copies | ForEach-Object {$_.Relative})
    Check 'required-asset-copies' ($copies.Count-eq 14 -and $copyPaths.Count-eq $requiredAssetPaths.Count -and -not(Compare-Object $copyPaths $requiredAssetPaths) -and $copyMismatch.Count-eq 0) 'All14 required project/attribution/license files are independently listed and match actual source bytes.'
    $manifestPath = Join-Path $out 'release-manifest.json'
    $manifestText = [IO.File]::ReadAllText($manifestPath)
    $manifest = $manifestText | ConvertFrom-Json
    $files = @(Get-ChildItem -LiteralPath $out -File -Recurse)
    Check 'manifest-complete-excluding-self' ($manifest.schema-eq 1 -and $manifest.version-ceq 'fixture.1' -and $manifest.files.Count-eq $files.Count-1 -and $manifest.excludes.Count-eq 1 -and $manifest.excludes[0]-ceq 'release-manifest.json') 'Schema explicitly excludes only itself and includes every actual fixture file.'
    Check 'three-fixed-engine-paths' (@($manifest.files | Where-Object {$_.path-in @('engines/markdown-export/levnote-markdown-export.exe','engines/markdown-export/manifest.json','engines/mindmap-svg/levnote-mindmap-svg.exe','engines/mindmap-svg/manifest.json','engines/mindmap-preview/levnote-mindmap-preview.exe','engines/mindmap-preview/manifest.json')}).Count-eq 6) 'All three fixed engine payload and manifest paths are inventoried.'
    $mismatch = @($manifest.files | Where-Object {$path=Join-Path $out $_.path; -not(Test-Path -LiteralPath $path) -or (Get-Item -LiteralPath $path).Length-ne $_.size -or (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash-ne $_.sha256})
    Check 'manifest-independent-file-oracle' ($mismatch.Count-eq 0) 'Actual directory files/bytes/digests independently re-read for every entry.'
    Check 'manifest-relative-no-machine-paths' (@($manifest.files | Where-Object {[IO.Path]::IsPathRooted($_.path) -or $_.path.Contains('\') -or $_.path.Contains('..')}).Count-eq 0 -and -not $manifestText.Contains($nativeRoot) -and -not $manifestText.Contains($root)) 'Only normalized relative file paths; no source machine paths.'
    $readme = [IO.File]::ReadAllText((Join-Path $out 'README.md'))
    Check 'version-config-unaccepted-template' ($readme.Contains('fixture.1') -and -not $readme.Contains('{{VERSION}}') -and $readme.Contains('%APPDATA%\LevNoteNative') -and $readme.Contains('生成成功不表示已经完成产品或交付验收')) 'Template substitutes actual version, states APPDATA, and does not claim future acceptance.'
    Check 'output-has-no-evidence' (@($files | Where-Object {$_.Name-match '(?i)(\.test\.exe$|\.log$|\.ps1$|package-result\.json$)'}).Count-eq 0) 'No build runners, logs, tester or evidence results in asset fixture.'
} catch { $errorText = $_.Exception.Message }
$record = [ordered]@{ RecordedAtUtc=[DateTime]::UtcNow.ToString('o'); Status=$(if($errorText){'FAIL'}else{'PASS'}); Passed=@($checks|Where-Object {$_.Passed}).Count; Total=$checks.Count; Checks=$checks; Error=$errorText; Scope='Package path preflight and asset/manifest fixture only; no product compile/launch or arbitrary executable engine.' }
$record | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $root 'result.json') -Encoding utf8
$record | ConvertTo-Json -Depth 6
if ($errorText) { throw $errorText }
