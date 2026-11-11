#requires -Version 7.0
# Internal packaging helpers. No process launch, command override or downloader.
function Assert-PlainDirectoryChain([string]$Path) {
    $current = [IO.Path]::GetFullPath($Path)
    if ($current -notmatch '^[A-Za-z]:\\') { throw 'Package directories require a local Windows drive path.' }
    while ($current) {
        $item = Get-Item -LiteralPath $current -Force -ErrorAction Stop
        if (-not $item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw 'Package directory components must be ordinary directories, not reparse points.'
        }
        $parent = [IO.Path]::GetDirectoryName($current.TrimEnd('\'))
        if (-not $parent -or $parent -eq $current) { break }
        if ($parent -match '^[A-Za-z]:$') { $parent += '\' }
        $current = $parent
    }
}

function Assert-NewPackageDirectories([string]$Output, [string]$Evidence) {
    $out = [IO.Path]::GetFullPath($Output).TrimEnd('\')
    $proof = [IO.Path]::GetFullPath($Evidence).TrimEnd('\')
    if ($out.Equals($proof, [StringComparison]::OrdinalIgnoreCase) -or
        $out.StartsWith($proof + '\', [StringComparison]::OrdinalIgnoreCase) -or
        $proof.StartsWith($out + '\', [StringComparison]::OrdinalIgnoreCase)) {
        throw 'OutputDirectory and EvidenceDirectory must be distinct and non-nested.'
    }
    foreach ($path in @($out, $proof)) {
        if (Test-Path -LiteralPath $path) { throw 'Package directories must be fresh; existing files or directories are never overwritten.' }
        Assert-PlainDirectoryChain ([IO.Path]::GetDirectoryName($path))
    }
}

function Get-PortableAssetCopies([string]$NativeRoot) {
    $assets = @(
        [pscustomobject]@{ Source = (Join-Path (Split-Path $NativeRoot -Parent) 'LICENSE'); Relative = 'LICENSE' },
        [pscustomobject]@{ Source = (Join-Path $NativeRoot 'THIRD_PARTY_NOTICES.md'); Relative = 'attribution/SOURCE_THIRD_PARTY_NOTICES.txt' },
        [pscustomobject]@{ Source = (Join-Path $NativeRoot 'vendor/github.com/egoist/mygo/README.md'); Relative = 'attribution/mygo-README.md' },
        [pscustomobject]@{ Source = (Join-Path $NativeRoot 'vendor/github.com/ebitengine/purego/README.md'); Relative = 'attribution/purego-README.md' },
        [pscustomobject]@{ Source = (Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch'); Relative = 'attribution/mygo-v0.8.3-no-startup-cleanup.patch' },
        [pscustomobject]@{ Source = (Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-bitmap-dispose.patch'); Relative = 'attribution/mygo-v0.8.3-bitmap-dispose.patch' },
        [pscustomobject]@{ Source = (Join-Path $NativeRoot 'dependency-patches/README.md'); Relative = 'attribution/dependency-patches-README.md' }
    )
    foreach ($name in @('Go-BSD-3-Clause.txt', 'go-text-typesetting.txt', 'mygo-MIT.txt', 'purego-Apache-2.0.txt', 'x-image-BSD-3-Clause.txt', 'x-net-BSD-3-Clause.txt', 'x-text-BSD-3-Clause.txt')) {
        $assets += [pscustomobject]@{ Source = (Join-Path $NativeRoot ('licenses/' + $name)); Relative = ('licenses/' + $name) }
    }
    $assets
}

function Assert-PortableInputs([string]$NativeRoot) {
    $paths = @((Get-PortableAssetCopies $NativeRoot).Source) + @(
        (Join-Path $NativeRoot 'portable/README.md'),
        (Join-Path $NativeRoot 'portable/NOTICE-PREAMBLE.md'),
        (Join-Path $NativeRoot 'scripts/go.ps1'),
        (Join-Path $NativeRoot 'tools/go/bin/go.exe')
    )
    foreach ($path in $paths) {
        $item = Get-Item -LiteralPath $path -Force -ErrorAction Stop
        if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Portable package input must be a regular file.' }
    }
}

function Write-PortableAssets([string]$NativeRoot, [string]$OutputRoot, [string]$Version) {
    $copies = Get-PortableAssetCopies $NativeRoot
    foreach ($entry in $copies) {
        $target = Join-Path $OutputRoot $entry.Relative
        $parent = Split-Path $target -Parent
        if (-not (Test-Path -LiteralPath $parent)) { New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null }
        Assert-PlainDirectoryChain $parent
        Copy-Item -LiteralPath $entry.Source -Destination $target -ErrorAction Stop
        if ((Get-FileHash -LiteralPath $target -Algorithm SHA256).Hash -cne (Get-FileHash -LiteralPath $entry.Source -Algorithm SHA256).Hash) { throw 'Portable asset copy digest mismatch.' }
    }
    $readme = [IO.File]::ReadAllText((Join-Path $NativeRoot 'portable/README.md')).Replace('{{VERSION}}', $Version)
    [IO.File]::WriteAllText((Join-Path $OutputRoot 'README.md'), $readme, [Text.UTF8Encoding]::new($false))
    $notice = [IO.File]::ReadAllText((Join-Path $NativeRoot 'THIRD_PARTY_NOTICES.md'))
    $notice = $notice.Replace('[LICENSE](../LICENSE)', '[LICENSE](LICENSE)').Replace('(vendor/github.com/ebitengine/purego/README.md)', '(attribution/purego-README.md)').Replace('(dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch)', '(attribution/mygo-v0.8.3-no-startup-cleanup.patch)').Replace('(dependency-patches/README.md)', '(attribution/dependency-patches-README.md)')
    $notice = $notice.Replace('(dependency-patches/mygo-v0.8.3-bitmap-dispose.patch)', '(attribution/mygo-v0.8.3-bitmap-dispose.patch)')
    $preamble = [IO.File]::ReadAllText((Join-Path $NativeRoot 'portable/NOTICE-PREAMBLE.md'))
    [IO.File]::WriteAllText((Join-Path $OutputRoot 'THIRD_PARTY_NOTICES.md'), ($preamble + "`n" + $notice), [Text.UTF8Encoding]::new($false))
}

function Assert-PortableLocalLinks([string]$OutputRoot) {
    foreach ($name in @('README.md', 'THIRD_PARTY_NOTICES.md')) {
        foreach ($match in [regex]::Matches([IO.File]::ReadAllText((Join-Path $OutputRoot $name)), '\]\(([^)]+)\)')) {
            $target = $match.Groups[1].Value
            if ($target -match '^(https?:|#)') { continue }
            $resolved = [IO.Path]::GetFullPath((Join-Path $OutputRoot $target))
            if (-not $resolved.StartsWith($OutputRoot.TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase) -or -not (Test-Path -LiteralPath $resolved)) {
                throw "Portable authored local link does not resolve inside output: $name -> $target"
            }
        }
    }
}

function Write-PortableReleaseManifest([string]$OutputRoot, [string]$Version) {
    $files = @(Get-ChildItem -LiteralPath $OutputRoot -File -Recurse | ForEach-Object {
        [pscustomobject][ordered]@{
            path = [IO.Path]::GetRelativePath($OutputRoot, $_.FullName).Replace('\', '/')
            size = $_.Length
            sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    } | Sort-Object -Property path -CaseSensitive)
    $manifest = [ordered]@{
        schema = 1
        version = $Version
        goos = 'windows'
        goarch = 'amd64'
        goamd64 = 'v1'
        excludes = @('release-manifest.json')
        scope = 'Every shipped file except this manifest; local integrity fingerprints, not signatures or acceptance evidence.'
        configuration = '%APPDATA%/LevNoteNative'
        files = $files
    }
    [IO.File]::WriteAllText((Join-Path $OutputRoot 'release-manifest.json'), ($manifest | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
}
