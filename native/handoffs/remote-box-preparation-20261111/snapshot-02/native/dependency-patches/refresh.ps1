param()
$ErrorActionPreference = 'Stop'
$nativeRoot = Split-Path $PSScriptRoot -Parent
$goWrapper = Join-Path $nativeRoot 'scripts\go.ps1'
$patches = @('mygo-v0.8.3-no-startup-cleanup.patch', 'mygo-v0.8.3-bitmap-dispose.patch')
$repoRoot = (& git -C $nativeRoot rev-parse --show-toplevel).Trim()
if ($LASTEXITCODE -ne 0 -or -not $repoRoot) {
	throw "unable to resolve repository root for local dependency patch"
}
Push-Location $nativeRoot
try {
	& $goWrapper mod vendor
	if ($LASTEXITCODE -ne 0) {
		throw "go mod vendor failed with exit code $LASTEXITCODE"
	}
	foreach ($patch in $patches) {
		$patchPath = Join-Path $PSScriptRoot $patch
		& git -C $repoRoot apply --directory=native --check --verbose $patchPath
		if ($LASTEXITCODE -ne 0) { throw "local MyGo patch does not apply cleanly: $patchPath" }
		& git -C $repoRoot apply --directory=native --verbose $patchPath
		if ($LASTEXITCODE -ne 0) { throw "failed to apply local MyGo patch: $patchPath" }
	}
	& $goWrapper test ./dependency-patches -count=1
	if ($LASTEXITCODE -ne 0) {
		throw "vendored MyGo patch verification failed"
	}
} finally {
	Pop-Location
}
