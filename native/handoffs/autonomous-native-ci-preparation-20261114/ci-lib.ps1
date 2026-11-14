#requires -Version 7.0
# Task20 draft: local integrity checks, not signatures, a sandbox or product acceptance.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$script:GoArchiveSHA256 = 'f26081f438319757f8a1b46d7d0b2c5ec7583d4038e485e0aa725cfde7885fff'
$script:GoArchiveBytes = 79010309L
$script:PatchManifestSHA256 = '9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a'
$script:ProductRoots = @('./app/','./cmd/...','./core/...','./formats/...','./modules/...','./ui/...')
function LocalPath([string]$Path) {
    if ($Path -notmatch '^[A-Za-z]:[\\/]' -or $Path -match '[\x00-\x1F]' -or $Path.Substring(2).Contains(':')) { throw "Not an absolute local drive path: $Path" }
    $full=[IO.Path]::GetFullPath($Path)
    if ($full.Length -eq 3) { return $full }
    return $full.TrimEnd([IO.Path]::DirectorySeparatorChar)
}
function Within([string]$Path,[string]$Root) {
    return (LocalPath $Path).StartsWith((LocalPath $Root)+'\',[StringComparison]::OrdinalIgnoreCase)
}
function PlainChain([string]$Path) {
    $current=LocalPath $Path
    while ($current) {
        if (Test-Path -LiteralPath $current) {
            $item=Get-Item -LiteralPath $current -Force -ErrorAction Stop
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Reparse input: $current" }
        }
        $parent=[IO.Path]::GetDirectoryName($current)
        if (-not $parent -or $parent -eq $current) { break }
        $current=$parent
    }
}
function PlainFile([string]$Path) {
    $full=LocalPath $Path; PlainChain $full
    if (-not (Test-Path -LiteralPath $full -PathType Leaf)) { throw "Missing regular file: $full" }
    return $full
}
function PlainDirectory([string]$Path) {
    $full=LocalPath $Path; PlainChain $full
    if (-not (Test-Path -LiteralPath $full -PathType Container)) { throw "Missing ordinary directory: $full" }
    return $full
}
function FreshDirectory([string]$Path) {
    $full=LocalPath $Path
    if (Test-Path -LiteralPath $full) { throw "Destination already exists: $full" }
    PlainChain ([IO.Path]::GetDirectoryName($full))
    New-Item -ItemType Directory -Path $full -ErrorAction Stop | Out-Null
    PlainDirectory $full
}
function RelativeFile([string]$Root,[string]$Relative) {
    if (-not $Relative -or $Relative.Contains('\') -or [IO.Path]::IsPathRooted($Relative) -or
        @($Relative.Split('/') | Where-Object { $_ -in @('','.', '..') -or $_.Contains(':') -or $_ -match '[\x00-\x1F]' }).Count) { throw "Noncanonical relative path: $Relative" }
    $full=LocalPath (Join-Path $Root $Relative)
    if (-not (Within $full $Root)) { throw 'Relative path escapes root.' }
    return $full
}
function NewJson([string]$Path,[object]$Value) {
    $bytes=[Text.UTF8Encoding]::new($false).GetBytes((ConvertTo-Json -InputObject $Value -Depth 20))
    $file=[IO.File]::Open($Path,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try { $file.Write($bytes,0,$bytes.Length); $file.Flush($true) } finally { $file.Dispose() }
}
function Json([string]$Path) {
    $bytes=[IO.File]::ReadAllBytes((PlainFile $Path))
    return ([Text.UTF8Encoding]::new($false,$true).GetString($bytes) | ConvertFrom-Json -ErrorAction Stop)
}
function Fingerprint([string]$Path,[string]$Relative='') {
    $full=PlainFile $Path; $before=Get-Item -LiteralPath $full -Force
    $length=$before.Length; $ticks=$before.LastWriteTimeUtc.Ticks
    $hash=(Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToLowerInvariant()
    $after=Get-Item -LiteralPath $full -Force
    if ($length -ne $after.Length -or $ticks -ne $after.LastWriteTimeUtc.Ticks) { throw "Hash input changed: $full" }
    return [pscustomobject][ordered]@{path=$Relative; absolutePath=$full; bytes=$length; sha256=$hash; lastWriteUtcTicks=$ticks}
}
function Pin([string]$Path,[string]$SHA256) {
    if ($SHA256 -notmatch '^[0-9A-Fa-f]{64}$') { throw "Missing real SHA256 pin: $Path" }
    $fact=Fingerprint $Path
    if ($fact.sha256 -cne $SHA256.ToLowerInvariant()) { throw "Pin mismatch: $Path" }
    return $fact
}
function SameFingerprint([object]$Before,[object]$After) {
    if ($Before.bytes -ne $After.bytes -or $Before.sha256 -cne $After.sha256 -or $Before.lastWriteUtcTicks -ne $After.lastWriteUtcTicks) { throw "Protected input changed: $($Before.path)" }
}
function Inventory([string]$Root) {
    $full=PlainDirectory $Root; $stack=[Collections.Generic.Stack[string]]::new(); $stack.Push($full)
    $files=[Collections.Generic.List[object]]::new(); $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    while ($stack.Count) {
        foreach ($entry in Get-ChildItem -LiteralPath $stack.Pop() -Force -ErrorAction Stop) {
            if ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Reparse inventory member: $($entry.FullName)" }
            if ($entry.PSIsContainer) { $stack.Push($entry.FullName) } else {
                $relative=[IO.Path]::GetRelativePath($full,$entry.FullName).Replace('\','/')
                if (-not $seen.Add($relative)) { throw "Case alias inventory member: $relative" }
                $files.Add((Fingerprint $entry.FullName $relative))
            }
        }
    }
    return @($files.ToArray() | Sort-Object -Property path -CaseSensitive)
}
function SameInventory([object[]]$Before,[object[]]$After,[string]$Label) {
    $left=@($Before | Sort-Object -Property path -CaseSensitive); $right=@($After | Sort-Object -Property path -CaseSensitive)
    if ($left.Count -ne $right.Count) { throw "$Label file count changed." }
    for ($i=0; $i -lt $left.Count; $i++) {
        if ($left[$i].path -cne $right[$i].path) { throw "$Label path set changed." }
        SameFingerprint $left[$i] $right[$i]
    }
}
function SourceInventory([string]$RepositoryRoot) {
    $root=PlainDirectory $RepositoryRoot; $facts=[Collections.Generic.List[object]]::new()
    foreach ($scope in @('app','cmd','core','formats','modules','ui','vendor','dependency-patches','scripts','portable','examples','licenses')) {
        foreach ($fact in @(Inventory (Join-Path $root ('native/'+$scope)))) {
            $fact.path='native/'+$scope+'/'+$fact.path; $facts.Add($fact)
        }
    }
    foreach ($name in @('LICENSE','native/go.mod','native/go.sum','native/THIRD_PARTY_NOTICES.md','native/AGENTS.md','native/README.md','native/VALIDATION.md')) {
        $facts.Add((Fingerprint (Join-Path $root $name) $name))
    }
    return @($facts.ToArray() | Sort-Object -Property path -CaseSensitive)
}
function CheckVendor([string]$NativeRoot) {
    $path=Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-patch-files.json'
    $manifestFact=Pin $path $script:PatchManifestSHA256; $manifest=Json $path
    if ($manifest.schema -ne 1 -or $manifest.module -cne 'github.com/egoist/mygo v0.8.3' -or @($manifest.files).Count -ne 26) { throw 'Incomplete vendor26 patch manifest.' }
    $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($file in $manifest.files) {
        if (-not $seen.Add($file.path)) { throw 'Duplicate vendor26 member.' }
        $null=Pin (RelativeFile $NativeRoot $file.path) $file.sha256
    }
    $app=[IO.File]::ReadAllText((Join-Path $NativeRoot 'vendor/github.com/egoist/mygo/app.go'))
    $modules=[IO.File]::ReadAllText((Join-Path $NativeRoot 'vendor/modules.txt'))
    if (-not $app.Contains('// LEVNOTE PATCH: this app does not enable MyGo updates') -or $app -match '\bgo\s+cleanUpdateLeftovers\s*\(' -or
        -not $modules.Contains('# github.com/egoist/mygo v0.8.3')) { throw 'MyGo startup safety/vendor selection differs.' }
    return $manifestFact
}
function ExtractVerifiedGo([string]$Archive,[string]$ToolsRoot) {
    $fact=Pin $Archive $script:GoArchiveSHA256
    if ($fact.bytes -ne $script:GoArchiveBytes) { throw 'Official Go archive byte count differs.' }
    $null=FreshDirectory $ToolsRoot
    $zip=[IO.Compression.ZipFile]::OpenRead($fact.absolutePath)
    $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $expected=[Collections.Generic.List[object]]::new()
    try {
        foreach ($entry in $zip.Entries) {
            $name=$entry.FullName
            if ($name -cne 'go/' -and -not $name.StartsWith('go/',[StringComparison]::Ordinal)) { throw 'Go ZIP entry outside fixed go root.' }
            $directory=$name.EndsWith('/',[StringComparison]::Ordinal)
            $relative=$name.TrimEnd('/')
            if (-not $seen.Add($relative)) { throw 'Duplicate/case-alias Go ZIP entry.' }
            $destination=RelativeFile $ToolsRoot $relative
            if (($entry.ExternalAttributes -shr 16 -band 0xF000) -eq 0xA000) { throw 'Go ZIP symlink entry refused.' }
            if ($directory) { New-Item -ItemType Directory -Path $destination -Force | Out-Null; continue }
            $parent=[IO.Path]::GetDirectoryName($destination)
            New-Item -ItemType Directory -Path $parent -Force | Out-Null; PlainChain $parent
            $input=$entry.Open(); $output=[IO.File]::Open($destination,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
            try { $input.CopyTo($output); $output.Flush($true) } finally { $output.Dispose(); $input.Dispose() }
            if ((Get-Item -LiteralPath $destination).Length -ne $entry.Length) { throw 'Go extraction length differs.' }
            $expected.Add((Fingerprint $destination $relative))
        }
    } finally { $zip.Dispose() }
    SameFingerprint $fact (Fingerprint $Archive)
    SameInventory $expected.ToArray() @(Inventory $ToolsRoot) 'Complete verified Go extraction'
    $null=PlainFile (Join-Path $ToolsRoot 'go/bin/go.exe')
    return $fact
}
function PE([string]$Path) {
    $stream=[IO.File]::OpenRead((PlainFile $Path)); $reader=[IO.BinaryReader]::new($stream)
    try {
        if ($stream.Length -lt 128 -or $reader.ReadUInt16() -ne 0x5A4D) { throw 'Not MZ executable.' }
        $stream.Position=0x3C; $offset=$reader.ReadInt32()
        if ($offset -lt 64 -or $offset+94 -gt $stream.Length) { throw 'Invalid PE offset.' }
        $stream.Position=$offset
        if ($reader.ReadUInt32() -ne 0x4550) { throw 'Not PE signature.' }
        $machine=$reader.ReadUInt16(); $stream.Position=$offset+24; $magic=$reader.ReadUInt16()
        $stream.Position=$offset+92; $subsystem=$reader.ReadUInt16()
        if ($machine -ne 0x8664 -or $magic -ne 0x20B -or $subsystem -ne 2) { throw 'Not Windows amd64 PE32+ GUI subsystem2.' }
        return [ordered]@{machine='0x8664'; magic='0x020B'; subsystem=2}
    } finally { $reader.Dispose() }
}
function CheckFivePEPackage([string]$NativeRoot,[string]$Bundle,[string]$PackageEvidence,[string]$Version) {
    $root=PlainDirectory $Bundle; $record=Json (Join-Path $PackageEvidence 'package-result.json')
    $manifestPath=Join-Path $root 'release-manifest.json'; $manifest=Json $manifestPath
    if ($manifest.schema -ne 1 -or $manifest.version -cne $Version -or $manifest.goos -cne 'windows' -or $manifest.goarch -cne 'amd64' -or $manifest.goamd64 -cne 'v1' -or
        $manifest.configuration -cne '%APPDATA%/LevNoteNative' -or @($manifest.excludes).Count -ne 1 -or $manifest.excludes[0] -cne 'release-manifest.json') { throw 'Release manifest header differs.' }
    $definitions=@(
        @('markdown-export','levnote.markdown-export','html','levnote-markdown-export.exe'),
        @('mindmap-svg','levnote.mindmap-svg','svg','levnote-mindmap-svg.exe'),
        @('mindmap-preview','levnote.mindmap-preview','mindmap-scene-v1','levnote-mindmap-preview.exe'),
        @('drawio-svg','levnote.drawio-svg','svg','levnote-drawio-svg.exe')
    )
    $required=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    $null=$required.Add('levnote-native.exe'); $null=$required.Add('README.md'); $null=$required.Add('THIRD_PARTY_NOTICES.md')
    $assets=@(
        @((Join-Path (Split-Path $NativeRoot -Parent) 'LICENSE'),'LICENSE'),
        @((Join-Path $NativeRoot 'THIRD_PARTY_NOTICES.md'),'attribution/SOURCE_THIRD_PARTY_NOTICES.txt'),
        @((Join-Path $NativeRoot 'vendor/github.com/egoist/mygo/README.md'),'attribution/mygo-README.md'),
        @((Join-Path $NativeRoot 'vendor/github.com/ebitengine/purego/README.md'),'attribution/purego-README.md'),
        @((Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch'),'attribution/mygo-v0.8.3-no-startup-cleanup.patch'),
        @((Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-bitmap-dispose.patch'),'attribution/mygo-v0.8.3-bitmap-dispose.patch'),
        @((Join-Path $NativeRoot 'dependency-patches/README.md'),'attribution/dependency-patches-README.md')
    )
    foreach ($name in @('Go-BSD-3-Clause.txt','go-text-typesetting.txt','mygo-MIT.txt','purego-Apache-2.0.txt','x-image-BSD-3-Clause.txt','x-net-BSD-3-Clause.txt','x-text-BSD-3-Clause.txt')) {
        $assets+=,@((Join-Path $NativeRoot ('licenses/'+$name)),('licenses/'+$name))
    }
    foreach ($asset in $assets) {
        $null=$required.Add($asset[1]); $before=Fingerprint $asset[0]; $copy=Fingerprint (RelativeFile $root $asset[1])
        if ($before.bytes -ne $copy.bytes -or $before.sha256 -cne $copy.sha256) { throw "Shipped attribution/license copy differs: $($asset[1])" }
    }
    $expectedReadme=[IO.File]::ReadAllText((Join-Path $NativeRoot 'portable/README.md')).Replace('{{VERSION}}',$Version)
    if ([IO.File]::ReadAllText((Join-Path $root 'README.md')) -cne $expectedReadme) { throw 'Generated portable README differs from source/version.' }
    foreach ($name in @('README.md','THIRD_PARTY_NOTICES.md')) {
        foreach ($link in [regex]::Matches([IO.File]::ReadAllText((Join-Path $root $name)),'\]\(([^)]+)\)')) {
            $target=$link.Groups[1].Value
            if ($target -match '^(https?:|#)') { continue }
            $null=PlainFile (RelativeFile $root $target)
        }
    }
    if ($record.Status -cne 'PACKAGED; build and asset checks passed, product acceptance is separate' -or $record.Version -cne $Version -or
        $record.TargetGOOS -cne 'windows' -or $record.TargetGOARCH -cne 'amd64' -or $record.TargetGOAMD64 -cne 'v1' -or @($record.Engines).Count -ne 4 -or
        (LocalPath $record.OutputDirectory) -cne $root -or (LocalPath $record.EvidenceDirectory) -cne (LocalPath $PackageEvidence)) { throw 'Package result identity differs.' }
    $peFacts=[Collections.Generic.List[object]]::new()
    $hostFact=Fingerprint (Join-Path $root 'levnote-native.exe')
    if ((LocalPath $record.HostPath) -cne $hostFact.absolutePath -or $record.HostBytes -ne $hostFact.bytes -or $record.HostSHA256.ToLowerInvariant() -cne $hostFact.sha256) { throw 'Package host binding differs.' }
    $peFacts.Add([ordered]@{path='levnote-native.exe'; pe=(PE $hostFact.absolutePath); sha256=$hostFact.sha256})
    foreach ($definition in $definitions) {
        $relative='engines/'+$definition[0]; $workerRelative=$relative+'/'+$definition[3]; $metadataRelative=$relative+'/manifest.json'
        $null=$required.Add($workerRelative); $null=$required.Add($metadataRelative)
        $worker=Fingerprint (RelativeFile $root $workerRelative); $metadataPath=RelativeFile $root $metadataRelative; $metadataFact=Fingerprint $metadataPath
        if ($metadataFact.bytes -le 0 -or $metadataFact.bytes -gt 16KB) { throw 'Worker manifest exceeds runtime cap.' }
        $metadata=Json $metadataPath
        if ($worker.bytes -le 0 -or $worker.bytes -gt 64MB -or $metadata.schema -ne 1 -or $metadata.engine -cne $definition[1] -or $metadata.engineVersion -cne $Version -or
            $metadata.protocol -ne 1 -or $metadata.format -cne $definition[2] -or $metadata.goos -cne 'windows' -or $metadata.goarch -cne 'amd64' -or
            $metadata.filename -cne $definition[3] -or $metadata.size -ne $worker.bytes -or $metadata.sha256 -cne $worker.sha256) { throw "Fixed worker manifest differs: $relative" }
        $match=@($record.Engines | Where-Object { $_.Engine -ceq $definition[1] })
        if ($match.Count -ne 1 -or $match[0].Format -cne $definition[2] -or $match[0].RelativePath -cne $relative -or
            (LocalPath $match[0].WorkerPath) -cne $worker.absolutePath -or $match[0].WorkerBytes -ne $worker.bytes -or
            $match[0].WorkerSHA256.ToLowerInvariant() -cne $worker.sha256 -or (LocalPath $match[0].ManifestPath) -cne (LocalPath $metadataPath)) { throw 'Package worker-record binding differs.' }
        $null=Pin $metadataPath $match[0].ManifestSHA256
        $peFacts.Add([ordered]@{path=$workerRelative; pe=(PE $worker.absolutePath); sha256=$worker.sha256})
    }
    $actual=@(Inventory $root | Where-Object { $_.path -cne 'release-manifest.json' })
    if ($required.Count -ne 25 -or $actual.Count -ne 25 -or @($manifest.files).Count -ne 25) { throw 'Exact complete five-PE/asset package file count differs.' }
    $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($file in $manifest.files) {
        if (-not $required.Contains($file.path) -or -not $seen.Add($file.path)) { throw 'Missing/extra/aliased manifest package member.' }
        $fact=Fingerprint (RelativeFile $root $file.path)
        if ($file.size -ne $fact.bytes -or $file.sha256 -cne $fact.sha256) { throw 'Release manifest byte/hash mismatch.' }
    }
    foreach ($file in $actual) { if (-not $required.Contains($file.path)) { throw 'Extra shipped file.' } }
    if ((LocalPath $record.ReleaseManifestPath) -cne (LocalPath $manifestPath)) { throw 'Package manifest path differs.' }
    $null=Pin $manifestPath $record.ReleaseManifestSHA256
    return [ordered]@{scope='Exact fresh host + all four fixed worker GUI PEs, full25-file assets and self-excluded manifest; no product launch'; pe=$peFacts.ToArray(); files=@(Inventory $root)}
}
