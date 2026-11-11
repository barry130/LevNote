function Get-DeliverySourceInputPaths([string]$RepositoryRoot, [string]$NativeRoot) {
    $paths = [Collections.Generic.List[string]]::new()
    foreach ($relativeDirectory in @(
        'native/app',
        'native/core',
        'native/formats',
        'native/modules',
        'native/cmd',
        'native/ui'
    )) {
        $directory = Join-Path $RepositoryRoot $relativeDirectory
        if (-not (Test-Path -LiteralPath $directory -PathType Container)) {
            throw "Required source scope directory is absent: $directory"
        }
        foreach ($file in Get-ChildItem -LiteralPath $directory -File -Recurse) {
            $paths.Add($file.FullName)
        }
    }
    . (Join-Path $NativeRoot 'scripts/portable-package-lib.ps1')
    foreach ($copy in Get-PortableAssetCopies $NativeRoot) {
        $paths.Add($copy.Source)
    }
    foreach ($relativeFile in @(
        'native/go.mod',
        'native/go.sum',
        'native/portable/README.md',
        'native/portable/NOTICE-PREAMBLE.md',
        'native/scripts/package-markdown-worker.ps1',
        'native/scripts/portable-package-lib.ps1',
        'native/scripts/go.ps1'
    )) {
        $paths.Add((Join-Path $RepositoryRoot $relativeFile))
    }
    $result = @($paths | Sort-Object -Unique -CaseSensitive)
    foreach ($path in $result) {
        $item = Get-Item -LiteralPath $path -ErrorAction Stop
        if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw "Source/package input must be one regular file: $path"
        }
    }
    return $result
}
