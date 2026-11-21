param(
    [Parameter(Mandatory = $true)][string]$ReviewedScratchRoot,
    [Parameter(Mandatory = $true)][string]$OfflineModuleCache
)
$ErrorActionPreference = 'Stop'
$nativeRoot = Split-Path $PSScriptRoot -Parent
$goWrapper = Join-Path $nativeRoot 'scripts\go.ps1'
$patches = @('mygo-v0.8.3-no-startup-cleanup.patch', 'mygo-v0.8.3-bitmap-dispose.patch', 'mygo-v0.8.3-native-input.patch')
$nativeInputManifestSHA256 = '176b42f0696c5dd168a4965605401ffecbbe2ebb9914a91a189128b5e0982ced'
$repoRoot = (& git -C $nativeRoot rev-parse --show-toplevel).Trim()
if ($LASTEXITCODE -ne 0 -or -not $repoRoot) { throw 'Unable to resolve the reviewed scratch checkout.' }
$repoRoot = [IO.Path]::GetFullPath($repoRoot)
if ($repoRoot -cne [IO.Path]::GetFullPath($ReviewedScratchRoot)) { throw 'Refresh is restricted to the explicitly reviewed scratch checkout.' }
if (-not (Test-Path -LiteralPath $OfflineModuleCache -PathType Container)) { throw 'The complete reviewed offline module cache is required before refresh.' }
$cache = [IO.Path]::GetFullPath($OfflineModuleCache)
$consumed = Join-Path $repoRoot '.native-input-refresh-consumed'
if (Test-Path -LiteralPath $consumed) { throw 'This scratch refresh has already been consumed; use a new reviewed scratch checkout.' }

# Validate all patch assets before invoking even an offline dependency command.
$oldManifestPath = Join-Path $PSScriptRoot 'mygo-v0.8.3-patch-files.json'
if ((Get-FileHash -LiteralPath $oldManifestPath -Algorithm SHA256).Hash.ToLowerInvariant() -cne '9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a') { throw 'Preserved patch manifest changed.' }
$old = [IO.File]::ReadAllText($oldManifestPath) | ConvertFrom-Json
if ($old.schema -ne 1 -or $old.module -cne 'github.com/egoist/mygo v0.8.3' -or $old.files.Count -ne 26) { throw 'Preserved patch manifest shape changed.' }
$manifestPath = Join-Path $PSScriptRoot 'mygo-v0.8.3-native-input-files.json'
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf) -or (Get-Item -LiteralPath $manifestPath).Attributes.HasFlag([IO.FileAttributes]::ReparsePoint) -or (Get-FileHash -LiteralPath $manifestPath -Algorithm SHA256).Hash.ToLowerInvariant() -cne $nativeInputManifestSHA256) { throw 'Native-input manifest pin changed.' }
$manifest = [IO.File]::ReadAllText($manifestPath) | ConvertFrom-Json
$expected = @(
    'dependency-patches/mygo-v0.8.3-native-input.patch',
    'vendor/github.com/egoist/mygo/internal/platform/text_input.go',
    'vendor/github.com/egoist/mygo/internal/windows/surface.go',
    'vendor/github.com/egoist/mygo/internal/windows/surface_input.go',
    'vendor/github.com/egoist/mygo/ui/text_input_cancel_test.go',
    'vendor/github.com/egoist/mygo/ui/text_input_client.go'
)
if ($manifest.schema -ne 1 -or $manifest.module -cne 'github.com/egoist/mygo v0.8.3' -or $manifest.files.Count -ne $expected.Count) { throw 'Native-input manifest shape changed.' }
$seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($file in $manifest.files) {
    if ($file.path -cnotin $expected -or -not $seen.Add($file.path) -or $file.path -in $old.files.path -or $file.sha256 -cnotmatch '^[0-9a-f]{64}$') { throw 'Native-input membership must be exact and disjoint.' }
}
foreach ($patch in $patches) {
    $rel = 'dependency-patches/' + $patch
    $entry = @($old.files) + @($manifest.files) | Where-Object { $_.path -ceq $rel }
    $path = Join-Path $PSScriptRoot $patch
    if (@($entry).Count -ne 1 -or -not (Test-Path -LiteralPath $path -PathType Leaf) -or (Get-Item -LiteralPath $path).Attributes.HasFlag([IO.FileAttributes]::ReparsePoint) -or (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() -cne $entry.sha256) { throw "Patch asset is absent or changed: $patch" }
}
$goMod = [IO.File]::ReadAllText((Join-Path $nativeRoot 'go.mod'))
if ($goMod -notmatch '(?m)^\s*github\.com/egoist/mygo\s+v0\.8\.3(?:\s|$)') { throw 'Refresh requires exactly MyGo v0.8.3.' }

# Preserve the caller environment. The wrapper sets GOPATH; explicitly supplying
# GOMODCACHE is what isolates dependency input for the actual child Go process.
$names = @('GOPROXY', 'GOSUMDB', 'GOVCS', 'GOTOOLCHAIN', 'GOMODCACHE', 'GOFLAGS', 'GOWORK', 'GOENV')
$previous = @{}
foreach ($name in $names) { $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
Push-Location $nativeRoot
try {
    $env:GOPROXY = 'off'
    $env:GOSUMDB = 'off'
    $env:GOVCS = '*:off'
    $env:GOTOOLCHAIN = 'local'
    $env:GOMODCACHE = $cache
    $env:GOFLAGS = ''
    $env:GOWORK = 'off'
    $env:GOENV = 'off'
    # Offline preflight occurs BEFORE vendoring. Missing pristine module input or
    # any transitive source fails closed; patched vendor is not a substitute.
    $pristine = & $goWrapper mod download -json github.com/egoist/mygo@v0.8.3
    if ($LASTEXITCODE -ne 0) { throw 'Pristine MyGo v0.8.3 is unavailable offline; refresh did not start.' }
    $pristineRecord = ($pristine -join "`n") | ConvertFrom-Json
    if ($pristineRecord.Error -or $pristineRecord.Path -cne 'github.com/egoist/mygo' -or $pristineRecord.Version -cne 'v0.8.3' -or -not (Test-Path -LiteralPath $pristineRecord.Zip -PathType Leaf)) { throw 'The pristine module ZIP was not proved by the offline preflight.' }
    & $goWrapper mod download -json all
    if ($LASTEXITCODE -ne 0) { throw 'The complete offline dependency closure is unavailable; refresh did not start.' }
    $marker = [IO.File]::Open($consumed, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $marker.Dispose()
    & $goWrapper mod vendor
    if ($LASTEXITCODE -ne 0) { throw "go mod vendor failed with exit code $LASTEXITCODE" }
    foreach ($patch in $patches) {
        $patchPath = Join-Path $PSScriptRoot $patch
        & git -C $repoRoot apply --directory=native --check --verbose $patchPath
        if ($LASTEXITCODE -ne 0) { throw "Local MyGo patch does not apply cleanly: $patchPath" }
        & git -C $repoRoot apply --directory=native --verbose $patchPath
        if ($LASTEXITCODE -ne 0) { throw "Failed to apply local MyGo patch: $patchPath" }
    }
    & $goWrapper test ./dependency-patches -count=1
    if ($LASTEXITCODE -ne 0) { throw 'Vendored MyGo patch verification failed.' }
} finally {
    Pop-Location
    foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $previous[$name], 'Process') }
}
