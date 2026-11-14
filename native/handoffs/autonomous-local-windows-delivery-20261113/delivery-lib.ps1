#requires -Version 7.0
# Fresh local acceptance helpers; never dot-source the historical delivery context.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-LocalPath([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or $Path -notmatch '^[A-Za-z]:[\\/]') {
        throw "An absolute local Windows drive path is required: $Path"
    }
    return [IO.Path]::GetFullPath($Path)
}
function Assert-PlainChain([string]$Path) {
    $current = Get-LocalPath $Path
    while ($current) {
        if (Test-Path -LiteralPath $current) {
            $item = Get-Item -LiteralPath $current -Force
            if (-not $item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
                throw "Directory ancestor is not ordinary: $current"
            }
        }
        $parent = [IO.Directory]::GetParent($current)
        if ($null -eq $parent) { break }
        $current = $parent.FullName
    }
}
function Assert-PlainFile([string]$Path) {
    $full = Get-LocalPath $Path
    Assert-PlainChain ([IO.Path]::GetDirectoryName($full))
    $item = Get-Item -LiteralPath $full -Force
    if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw "Expected an ordinary regular file: $full"
    }
    return $full
}
function Assert-PlainDirectory([string]$Path) {
    $full = Get-LocalPath $Path
    Assert-PlainChain $full
    if (-not (Test-Path -LiteralPath $full -PathType Container)) { throw "Directory absent: $full" }
    return $full
}
function Test-Within([string]$Path, [string]$Root) {
    $p = (Get-LocalPath $Path).TrimEnd('\')
    $r = (Get-LocalPath $Root).TrimEnd('\')
    return $p.Equals($r, [StringComparison]::OrdinalIgnoreCase) -or
        $p.StartsWith($r + '\', [StringComparison]::OrdinalIgnoreCase)
}
function Get-RelativeFile([string]$Root, [string]$Relative) {
    if ([string]::IsNullOrWhiteSpace($Relative) -or $Relative.Contains('\') -or
        $Relative -match '(^/|:|[\x00-\x1f])' -or
        @($Relative.Split('/') | Where-Object { $_ -in @('', '.', '..') }).Count -ne 0) {
        throw "Relative inventory path is invalid: $Relative"
    }
    $full = Get-LocalPath (Join-Path $Root $Relative)
    if (-not (Test-Within $full $Root) -or $full -eq (Get-LocalPath $Root)) { throw 'Relative path escapes root.' }
    return $full
}
function New-FreshDirectory([string]$Path) {
    $full = Get-LocalPath $Path
    Assert-PlainChain ([IO.Path]::GetDirectoryName($full))
    if (Test-Path -LiteralPath $full) { throw "Create-only directory already exists: $full" }
    New-Item -ItemType Directory -Path $full -ErrorAction Stop | Out-Null
    Assert-PlainChain $full
    return $full
}
function Write-NewJson([string]$Path, [object]$Value) {
    $full = Get-LocalPath $Path
    Assert-PlainChain ([IO.Path]::GetDirectoryName($full))
    $bytes = [Text.UTF8Encoding]::new($false).GetBytes(($Value | ConvertTo-Json -Depth 30) + "`n")
    $stream = [IO.File]::Open($full, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::Read)
    try { $stream.Write($bytes, 0, $bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
}
function Read-LocalJson([string]$Path) {
    $full = Assert-PlainFile $Path
    return [IO.File]::ReadAllText($full, [Text.UTF8Encoding]::new($false, $true)) | ConvertFrom-Json
}
function Get-Fingerprint([string]$Path, [string]$Relative = '') {
    $full = Assert-PlainFile $Path
    $item = Get-Item -LiteralPath $full -Force
    $hash = (Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToUpperInvariant()
    $after = Get-Item -LiteralPath $full -Force
    if ($after.Length -ne $item.Length -or $after.LastWriteTimeUtc.Ticks -ne $item.LastWriteTimeUtc.Ticks) {
        throw "File changed while fingerprinting: $full"
    }
    return [pscustomobject][ordered]@{
        path = $Relative; absolutePath = $full; bytes = [int64]$item.Length
        sha256 = $hash; lastWriteUtcTicks = [int64]$item.LastWriteTimeUtc.Ticks
    }
}
function Assert-Pin([string]$Path, [string]$Hash) {
    if ($Hash -cnotmatch '^[0-9A-Fa-f]{64}$') { throw "Missing real SHA256 pin for $Path" }
    $actual = Get-Fingerprint $Path
    if ($actual.sha256 -cne $Hash.ToUpperInvariant()) { throw "SHA256 pin differs: $Path" }
    return $actual
}
function Assert-ExactFingerprint([object]$Before, [object]$After, [bool]$Time = $true) {
    if ($Before.bytes -ne $After.bytes -or $Before.sha256 -cne $After.sha256 -or
        ($Time -and $Before.lastWriteUtcTicks -ne $After.lastWriteUtcTicks)) {
        throw "Fingerprint changed: $($Before.path)"
    }
}
function Get-TreeInventory([string]$Root) {
    $rootFull = Assert-PlainDirectory $Root
    $map = [Collections.Generic.Dictionary[string, object]]::new([StringComparer]::Ordinal)
    $pending = [Collections.Generic.Stack[string]]::new()
    $pending.Push($rootFull)
    while ($pending.Count -gt 0) {
        foreach ($item in Get-ChildItem -LiteralPath $pending.Pop() -Force) {
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Reparse inventory entry: $($item.FullName)" }
            if ($item.PSIsContainer) { $pending.Push($item.FullName); continue }
            $relative = [IO.Path]::GetRelativePath($rootFull, $item.FullName).Replace('\', '/')
            $map.Add($relative, (Get-Fingerprint $item.FullName $relative))
        }
    }
    $keys = [Collections.Generic.List[string]]::new()
    foreach ($key in $map.Keys) { $keys.Add($key) }
    $keys.Sort([StringComparer]::Ordinal)
    return @(foreach ($key in $keys) { $map[$key] })
}
function Assert-ExactInventory([object[]]$Expected, [object[]]$Actual, [string]$Label, [bool]$Time = $true) {
    $left = [Collections.Generic.Dictionary[string, object]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in @($Expected)) {
        if ($left.ContainsKey([string]$entry.path)) { throw "$Label duplicate path: $($entry.path)" }
        $left.Add([string]$entry.path, $entry)
    }
    if ($left.Count -ne @($Actual).Count) { throw "$Label file count differs: $($left.Count)/$(@($Actual).Count)" }
    $seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in @($Actual)) {
        if (-not $seen.Add([string]$entry.path) -or -not $left.ContainsKey([string]$entry.path)) {
            throw "$Label unexpected path: $($entry.path)"
        }
        $before = $left[[string]$entry.path]
        if ([string]$before.path -cne [string]$entry.path) { throw "$Label path spelling changed: $($entry.path)" }
        Assert-ExactFingerprint $before $entry $Time
    }
}
function Get-SourceInventory([string]$RepositoryRoot, [object]$Frozen) {
    $repository = Assert-PlainDirectory $RepositoryRoot
    if ($Frozen.schema -ne 1 -or -not [string]::Equals((Get-LocalPath $Frozen.repositoryRoot), $repository, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Fresh source inventory schema/root differs.'
    }
    $required = @('native/app','native/cmd','native/core','native/formats','native/modules','native/ui','native/vendor',
        'native/dependency-patches','native/scripts','native/portable','native/examples','native/licenses')
    foreach ($scope in $required) {
        if (@($Frozen.directories) -cnotcontains $scope) { throw "Fresh source inventory lacks required directory: $scope" }
    }
    foreach ($file in @('LICENSE','native/go.mod','native/go.sum','native/THIRD_PARTY_NOTICES.md')) {
        if (@($Frozen.rootFiles) -cnotcontains $file) { throw "Fresh source inventory lacks required root file: $file" }
    }
    $paths = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $result = [Collections.Generic.List[object]]::new()
    foreach ($scope in @($Frozen.directories)) {
        if ($scope -match '^native/(artifacts|bin|handoffs|tools)(/|$)' -or $scope -match '^\.localtools(/|$)') {
            throw 'Source directories must not consume historical/evidence/output/toolchain namespaces.'
        }
        $full = Get-RelativeFile $repository $scope
        foreach ($file in @(Get-TreeInventory $full)) {
            $relative = $scope + '/' + $file.path
            if (-not $paths.Add($relative)) { throw "Overlapping source inventory scope: $relative" }
            $file.path = $relative
            $result.Add($file)
        }
    }
    foreach ($relative in @($Frozen.rootFiles)) {
        if (-not $paths.Add($relative)) { throw "Duplicate source file: $relative" }
        $result.Add((Get-Fingerprint (Get-RelativeFile $repository $relative) $relative))
    }
    if ($result.Count -eq 0 -or @($Frozen.files).Count -eq 0) { throw 'Source inventory is empty.' }
    foreach ($file in @($Frozen.files)) {
        $null = Get-RelativeFile $repository $file.path
        if ($file.sha256 -cnotmatch '^[0-9A-Fa-f]{64}$' -or $file.bytes -lt 0 -or $file.lastWriteUtcTicks -le 0) {
            throw 'Invalid source inventory fingerprint.'
        }
        $file.sha256 = $file.sha256.ToUpperInvariant()
    }
    Assert-ExactInventory @($Frozen.files) @($result.ToArray()) 'Frozen complete source scopes'
    return $result.ToArray()
}
function Get-PeFacts([string]$Path) {
    $fp = Get-Fingerprint $Path
    $stream = [IO.File]::OpenRead($fp.absolutePath)
    $reader = [IO.BinaryReader]::new($stream)
    try {
        if ($stream.Length -lt 64 -or $reader.ReadUInt16() -ne 0x5A4D) { throw "Missing DOS header: $Path" }
        $stream.Position = 60; $offset = $reader.ReadInt32()
        if ($offset -lt 64 -or [int64]$offset + 94 -gt $stream.Length) { throw 'Invalid PE header offset.' }
        $stream.Position = $offset
        if ($reader.ReadUInt32() -ne 0x00004550) { throw 'Missing PE signature.' }
        $machine = $reader.ReadUInt16()
        $stream.Position = $offset + 20; $optionalBytes = $reader.ReadUInt16()
        if ($optionalBytes -lt 70 -or [int64]$offset + 24 + $optionalBytes -gt $stream.Length) { throw 'Invalid optional header size.' }
        $stream.Position = $offset + 24; $magic = $reader.ReadUInt16()
        $stream.Position = $offset + 92; $subsystem = $reader.ReadUInt16()
        if ($machine -ne 0x8664 -or $magic -ne 0x20B -or $subsystem -ne 2) { throw "Not Windows amd64 GUI PE: $Path" }
        return [ordered]@{ file=$fp; machine='0x8664'; optionalHeaderMagic='0x020B'; subsystem=2 }
    } finally { $reader.Dispose() }
}
function Assert-Package([string]$Candidate, [string]$Version, [object]$Manifest, [object]$Record) {
    if ($Manifest.schema -ne 1 -or $Manifest.version -cne $Version -or $Manifest.goos -cne 'windows' -or
        $Manifest.goarch -cne 'amd64' -or $Manifest.goamd64 -cne 'v1' -or
        $Manifest.configuration -cne '%APPDATA%/LevNoteNative' -or
        @($Manifest.excludes).Count -ne 1 -or $Manifest.excludes[0] -cne 'release-manifest.json') {
        throw 'Package manifest schema/version/target/self-exclusion differs.'
    }
    $actual = @(Get-TreeInventory $Candidate)
    $expected = @(foreach ($entry in @($Manifest.files)) {
        $null = Get-RelativeFile $Candidate $entry.path
        if ($entry.sha256 -cnotmatch '^[0-9A-Fa-f]{64}$' -or $entry.size -lt 0 -or $entry.path -ceq 'release-manifest.json') { throw 'Invalid release manifest entry.' }
        [pscustomobject]@{path=$entry.path;bytes=[int64]$entry.size;sha256=$entry.sha256.ToUpperInvariant()}
    })
    $payload = @($actual | Where-Object { $_.path -cne 'release-manifest.json' })
    Assert-ExactInventory $expected $payload 'Complete shipped manifest' $false
    $expectedExe = @('levnote-native.exe','engines/markdown-export/levnote-markdown-export.exe',
        'engines/mindmap-svg/levnote-mindmap-svg.exe','engines/mindmap-preview/levnote-mindmap-preview.exe',
        'engines/drawio-svg/levnote-drawio-svg.exe')
    $exeEntries = @($payload | Where-Object { $_.path.EndsWith('.exe', [StringComparison]::OrdinalIgnoreCase) })
    if ($exeEntries.Count -ne 5) { throw 'Expected exactly host plus four shipped worker PEs.' }
    $pe = @(foreach ($relative in $expectedExe) { Get-PeFacts (Get-RelativeFile $Candidate $relative) })
    foreach ($definition in @(
        @('markdown-export','levnote.markdown-export','html','levnote-markdown-export.exe'),
        @('mindmap-svg','levnote.mindmap-svg','svg','levnote-mindmap-svg.exe'),
        @('mindmap-preview','levnote.mindmap-preview','mindmap-scene-v1','levnote-mindmap-preview.exe'),
        @('drawio-svg','levnote.drawio-svg','svg','levnote-drawio-svg.exe'))) {
        $engine = Read-LocalJson (Join-Path $Candidate ('engines/' + $definition[0] + '/manifest.json'))
        $binary = Get-Fingerprint (Join-Path $Candidate ('engines/' + $definition[0] + '/' + $definition[3]))
        $recordRows = @($Record.Engines | Where-Object { $_.Engine -ceq $definition[1] })
        $engineManifestFile = Get-Fingerprint (Join-Path $Candidate ('engines/' + $definition[0] + '/manifest.json'))
        if ($recordRows.Count -ne 1) { throw 'Pinned package result requires each unique fixed engine.' }
        $recordEngine = $recordRows[0]
        if ($recordEngine.Format -cne $definition[2] -or $recordEngine.RelativePath -cne ('engines/' + $definition[0]) -or
            $recordEngine.WorkerBytes -ne $binary.bytes -or $recordEngine.WorkerSHA256.ToUpperInvariant() -cne $binary.sha256 -or
            $recordEngine.ManifestSHA256.ToUpperInvariant() -cne $engineManifestFile.sha256 -or
            -not [string]::Equals((Get-LocalPath $recordEngine.WorkerPath), $binary.absolutePath, [StringComparison]::OrdinalIgnoreCase) -or
            -not [string]::Equals((Get-LocalPath $recordEngine.ManifestPath), $engineManifestFile.absolutePath, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Pinned package result engine does not bind shipped binary/manifest.'
        }
        if ($engine.schema -ne 1 -or $engine.engine -cne $definition[1] -or $engine.engineVersion -cne $Version -or
            $engine.protocol -ne 1 -or $engine.format -cne $definition[2] -or $engine.filename -cne $definition[3] -or
            $engine.goos -cne 'windows' -or $engine.goarch -cne 'amd64' -or
            $engine.size -ne $binary.bytes -or $engine.sha256.ToUpperInvariant() -cne $binary.sha256) { throw 'Shipped engine manifest differs.' }
    }
    $hostFile = Get-Fingerprint (Join-Path $Candidate 'levnote-native.exe')
    $manifestFile = Get-Fingerprint (Join-Path $Candidate 'release-manifest.json')
    if ($Record.Status -cne 'PACKAGED; build and asset checks passed, product acceptance is separate' -or
        $Record.Version -cne $Version -or $Record.TargetGOOS -cne 'windows' -or $Record.TargetGOARCH -cne 'amd64' -or
        $Record.TargetGOAMD64 -cne 'v1' -or @($Record.Engines).Count -ne 4 -or
        -not [string]::Equals((Get-LocalPath $Record.OutputDirectory), (Get-LocalPath $Candidate), [StringComparison]::OrdinalIgnoreCase) -or
        $Record.HostBytes -ne $hostFile.bytes -or $Record.HostSHA256.ToUpperInvariant() -cne $hostFile.sha256 -or
        $Record.ReleaseManifestSHA256.ToUpperInvariant() -cne $manifestFile.sha256) { throw 'Pinned package result does not bind the candidate.' }
    return [ordered]@{files=$actual; pe=$pe; manifest=$manifestFile}
}
function Assert-TextFixture([string]$Path) {
    $data = [IO.File]::ReadAllBytes((Assert-PlainFile $Path))
    if ($data.Length -le 3 -or $data.Length -gt 4MB -or $data[0] -ne 0xEF -or $data[1] -ne 0xBB -or $data[2] -ne 0xBF) {
        throw 'Text fixture requires nonempty bounded UTF-8 with BOM.'
    }
    $text = [Text.UTF8Encoding]::new($false, $true).GetString($data, 3, $data.Length - 3)
    if (-not $text.Contains("`r`n") -or [regex]::IsMatch($text, '(?<!\r)\n|\r(?!\n)') -or $text.Contains([char]0)) {
        throw 'Text fixture must contain CRLF only and no NUL.'
    }
}
function Assert-PngFixture([string]$Path) {
    $data = [IO.File]::ReadAllBytes((Assert-PlainFile $Path))
    if ($data.Length -lt 33 -or $data.Length -gt 16MB) { throw 'PNG fixture size is invalid.' }
    $signature = [byte[]]@(137,80,78,71,13,10,26,10)
    for ($i=0; $i -lt 8; $i++) { if ($data[$i] -ne $signature[$i]) { throw 'PNG signature differs.' } }
    if ([Text.Encoding]::ASCII.GetString($data, 12, 4) -cne 'IHDR') { throw 'PNG first chunk is not IHDR.' }
    # This is a signature gate, not a decoder or visual PNG acceptance claim.
}
function Get-SessionCopy([string]$Session, [string]$Copy) {
    $before = Get-Fingerprint $Session
    if ($before.bytes -le 0 -or $before.bytes -gt 16MB) { throw 'Session size is invalid.' }
    $bytes = [IO.File]::ReadAllBytes($before.absolutePath)
    $stream = [IO.File]::Open($Copy, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::Read)
    try { $stream.Write($bytes, 0, $bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
    [IO.File]::SetLastWriteTimeUtc($Copy, [DateTime]::new($before.lastWriteUtcTicks, [DateTimeKind]::Utc))
    Assert-ExactFingerprint $before (Get-Fingerprint $Session)
    Assert-ExactFingerprint $before (Get-Fingerprint $Copy)
    return $before
}
function Assert-SessionKeys([object]$Object, [string[]]$Required, [string[]]$Allowed, [string]$Label) {
    $names = @($Object.PSObject.Properties.Name)
    foreach ($key in $Required) { if ($names -cnotcontains $key) { throw "$Label missing $key" } }
    foreach ($key in $names) { if ($Allowed -cnotcontains $key) { throw "$Label forbidden/unknown $key" } }
}
function Get-CleanSession([string]$Path, [string]$TextPath, [string]$ImagePath) {
    $bytes = [IO.File]::ReadAllBytes((Assert-PlainFile $Path))
    if ($bytes.Length -eq 0 -or $bytes[-1] -in @(10,13)) { throw 'Session is empty or has terminal newline.' }
    $session = Read-LocalJson $Path
    Assert-SessionKeys $session @('version','active','tabs','preferences') @('version','root','active','tabs','preferences') 'Session'
    if ($session.version -ne 1 -or $session.active -ne 1 -or @($session.tabs).Count -ne 2) { throw 'Ordered two-tab active1 session differs.' }
    $text = $session.tabs[0]; $image = $session.tabs[1]
    Assert-SessionKeys $session.preferences @('wordWrap') @('theme','fontSize','lineNumbers','wordWrap','sidebarWidth') 'Session preferences'
    if ($session.preferences.wordWrap -isnot [bool]) { throw 'Session wordWrap is not boolean.' }
    Assert-SessionKeys $text.selection @('Anchor','Head') @('Anchor','Head') 'Clean text selection'
    if ($text.selection.Anchor -isnot [int64] -and $text.selection.Anchor -isnot [int] -or
        $text.selection.Head -isnot [int64] -and $text.selection.Head -isnot [int] -or
        $text.selection.Anchor -lt 0 -or $text.selection.Head -lt 0) { throw 'Clean text selection is not a nonnegative integer range.' }
    Assert-SessionKeys $text @('path','title','encoding','bom','selection','pluginId','view') @('path','title','encoding','bom','selection','scroll','pluginId','view') 'Clean text'
    Assert-SessionKeys $image @('record','kind','path','pluginId','view') @('record','kind','path','title','pluginId','view') 'Clean binary'
    if ($text.path -cne $TextPath -or $text.title -cne [IO.Path]::GetFileName($TextPath) -or
        $text.encoding -cne 'UTF-8' -or $text.bom -isnot [bool] -or -not $text.bom -or
        $text.pluginId -cne 'diagram-notes' -or $text.view -cne 'preview' -or
        $image.path -cne $ImagePath -or $image.record -cne 'binary' -or $image.kind -cne 'png' -or
        $image.pluginId -cne 'image' -or $image.view -cne 'preview') { throw 'Clean session routes/paths/BOM/encoding differ.' }
    return [ordered]@{version=1; active=1; tabCount=2; clean=$true; text=$text; image=$image; raw=$session}
}
function Copy-ExactTree([string]$Source, [string]$Target, [object[]]$Inventory) {
    foreach ($file in @($Inventory)) {
        $destination = Get-RelativeFile $Target $file.path
        $parent = [IO.Path]::GetDirectoryName($destination)
        if (-not (Test-Path -LiteralPath $parent)) { New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null }
        Assert-PlainChain $parent
        [IO.File]::Copy((Get-RelativeFile $Source $file.path), $destination, $false)
        [IO.File]::SetLastWriteTimeUtc($destination, [DateTime]::new($file.lastWriteUtcTicks, [DateTimeKind]::Utc))
    }
    Assert-ExactInventory $Inventory @(Get-TreeInventory $Target) 'Exact copied bundle'
}
