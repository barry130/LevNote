param([Parameter(ValueFromRemainingArguments = $true)][string[]]$GoArguments)
$ErrorActionPreference = 'Stop'
$nativeRoot = Split-Path $PSScriptRoot -Parent
$goBinary = Join-Path $nativeRoot 'tools\go\bin\go.exe'
if (-not (Test-Path -LiteralPath $goBinary)) { throw 'Run node scripts/bootstrap.mjs, then extract the verified archive into tools first.' }
# Ordinary builds use Go's automatic vendor mode. A plain `mod vendor` replaces
# the local startup-safety and disposal patches, so verify them before commands that can run
# project code. Refresh is allowed to vendor first and applies the patch before
# its final `test`; there is no persistent opt-out or machine-wide Go setting.
# Build context is deliberately project-local: persisted Go environment and
# enclosing go.work cannot replace the inspected vendor inputs. This guard is
# an accidental-bypass check, not a hostile-code or operating-system sandbox.
function Split-GoFlags([string]$Value) {
    # Go's quoted.Split permits quotes only at the start of an entire field;
    # it performs no shell expansion or backslash unescaping.
    $offset = 0
    while ($offset -lt $Value.Length) {
        while ($offset -lt $Value.Length -and $Value[$offset] -in @(' ', "`t", "`r", "`n")) { $offset++ }
        if ($offset -eq $Value.Length) { break }
        $quote = $Value[$offset]
        if ($quote -eq "'" -or $quote -eq '"') {
            $start = ++$offset
            while ($offset -lt $Value.Length -and $Value[$offset] -ne $quote) { $offset++ }
            if ($offset -eq $Value.Length) { throw 'Invalid GOFLAGS: unterminated quoted field.' }
            $Value.Substring($start, $offset - $start)
            $offset++
        } else {
            $start = $offset
            while ($offset -lt $Value.Length -and $Value[$offset] -notin @(' ', "`t", "`r", "`n")) { $offset++ }
            $Value.Substring($start, $offset - $start)
        }
    }
}
$guardedCommand = $GoArguments | Where-Object { $_ -cin @('build', 'run', 'install', 'test') } | Select-Object -First 1
if ($guardedCommand) {
    $mygoSourcePath = Join-Path $nativeRoot 'vendor\github.com\egoist\mygo\app.go'
    $vendorManifestPath = Join-Path $nativeRoot 'vendor\modules.txt'
    $patchMarker = '// LEVNOTE PATCH: this app does not enable MyGo updates'
    $patchHelp = 'MyGo startup-safety patch is missing or bypassed. Run native/dependency-patches/refresh.ps1 from the repository, then retry native/scripts/go.ps1.'
    if (-not (Test-Path -LiteralPath $mygoSourcePath) -or -not (Test-Path -LiteralPath $vendorManifestPath)) { throw $patchHelp }
    $mygoSource = [System.IO.File]::ReadAllText($mygoSourcePath)
    $vendorManifest = [System.IO.File]::ReadAllText($vendorManifestPath)
    if (-not $mygoSource.Contains($patchMarker) -or $mygoSource -match '\bgo\s+cleanUpdateLeftovers\s*\(' -or -not $vendorManifest.Contains('# github.com/egoist/mygo v0.8.3')) { throw $patchHelp }
    # Pin the manifest as well as its files: a shortened manifest cannot bless
    # a partial patch. Include both reproducible patches and every changed
    # source/test file; this is still an accidental-bypass check, not a sandbox.
    $patchManifestSHA256 = '9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a'
    $patchManifestPath = Join-Path $nativeRoot 'dependency-patches/mygo-v0.8.3-patch-files.json'
    $disposalHelp = 'MyGo bitmap-disposal patch is incomplete or changed. Run native/dependency-patches/refresh.ps1; intentional patch updates must refresh the complete file manifest and its wrapper pin.'
    if (-not (Test-Path -LiteralPath $patchManifestPath -PathType Leaf) -or (Get-FileHash -LiteralPath $patchManifestPath -Algorithm SHA256).Hash.ToLowerInvariant() -cne $patchManifestSHA256) { throw $disposalHelp }
    $patchManifest = [IO.File]::ReadAllText($patchManifestPath) | ConvertFrom-Json
    if ($patchManifest.schema -ne 1 -or $patchManifest.module -cne 'github.com/egoist/mygo v0.8.3' -or $patchManifest.files.Count -ne 26) { throw $disposalHelp }
    foreach ($file in $patchManifest.files) {
        $path = Join-Path $nativeRoot $file.path
        if (-not (Test-Path -LiteralPath $path -PathType Leaf) -or (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() -cne $file.sha256) { throw "$disposalHelp File: $($file.path)" }
    }
    $argumentFlags = @($GoArguments) + @(Split-GoFlags $env:GOFLAGS)
    for ($index = 0; $index -lt $argumentFlags.Count; $index++) {
        $flag = $argumentFlags[$index]
        # Go flags are case-sensitive: test -c is legitimate, -C changes cwd.
        if ($flag -cmatch '^--?(?:C|modfile|overlay)(?:=|$)') { throw $patchHelp }
        if ($flag -cmatch '^--?mod=(.*)$' -and $Matches[1] -cne 'vendor') { throw $patchHelp }
        if ($flag -cmatch '^--?mod$' -and ($index + 1 -ge $argumentFlags.Count -or $argumentFlags[$index + 1] -cne 'vendor')) { throw $patchHelp }
    }
}
$env:GOROOT = Join-Path $nativeRoot 'tools\go'
$env:GOPATH = Join-Path $nativeRoot 'build-cache\gopath'
$env:GOCACHE = Join-Path $nativeRoot 'build-cache\build'
$env:GOTOOLCHAIN = 'local'
$cacheRoot = Join-Path $nativeRoot 'build-cache'
New-Item -ItemType Directory -Path $cacheRoot -Force | Out-Null
# Isolate Go telemetry under this repository and disable it before starting Go.
# TEST_TELEMETRY_DIR is supported by the bundled toolchain; do not run the
# global `go telemetry off` command or change the user's own telemetry mode.
$env:TEST_TELEMETRY_DIR = Join-Path $cacheRoot 'telemetry'
New-Item -ItemType Directory -Path $env:TEST_TELEMETRY_DIR -Force | Out-Null
Set-Content -LiteralPath (Join-Path $env:TEST_TELEMETRY_DIR 'mode') -Value 'off' -Encoding ascii
# Keep Go's ./... and tidy from walking downloaded dependency/toolchain sources.
if (-not (Test-Path -LiteralPath (Join-Path $cacheRoot 'go.mod'))) {
    Set-Content -LiteralPath (Join-Path $cacheRoot 'go.mod') -Value "module levnote.local/build-cache`n`ngo 1.27.3`n"
}
$previousGoWork = [Environment]::GetEnvironmentVariable('GOWORK', 'Process')
$previousGoEnv = [Environment]::GetEnvironmentVariable('GOENV', 'Process')
Push-Location $nativeRoot
try {
    $env:GOWORK = 'off'
    $env:GOENV = 'off'
    & $goBinary @GoArguments
    $taskExitCode = $LASTEXITCODE
} finally {
    Pop-Location
    [Environment]::SetEnvironmentVariable('GOWORK', $previousGoWork, 'Process')
    [Environment]::SetEnvironmentVariable('GOENV', $previousGoEnv, 'Process')
}
exit $taskExitCode
