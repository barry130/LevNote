#requires -Version 7.0
# INACTIVE STATIC REVISION. No process launcher, downloader or CI enablement.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$script:ProductRoots = @('./app/','./cmd/...','./core/...','./formats/...','./modules/...','./ui/...')
$script:GoArchiveSHA256 = 'f26081f438319757f8a1b46d7d0b2c5ec7583d4038e485e0aa725cfde7885fff'
$script:GoArchiveBytes = 79010309L

function AssertProductRoots([string[]]$Roots) {
    if ($null -eq $Roots -or $Roots.Count -ne 6) { throw 'Exactly six product root arguments required.' }
    for ($i=0; $i -lt 6; $i++) {
        if ($Roots[$i] -cne $script:ProductRoots[$i]) { throw "Changed/reordered product root at index $i." }
    }
}
function AssertRelativeGrammar([string]$Relative,[bool]$AllowTerminalSlash=$false) {
    # Validate the RAW spelling, before joining, normalization, decoding or stripping.
    if ([string]::IsNullOrEmpty($Relative) -or $Relative -match '[\x00-\x20\x7F\s\\:*?"<>|#%]' -or
        $Relative.StartsWith('/') -or $Relative.Contains('//')) { throw "Noncanonical relative path: $Relative" }
    $parts=$Relative.Split('/')
    for ($i=0; $i -lt $parts.Count; $i++) {
        $part=$parts[$i]
        if ($part -ceq '' -and $AllowTerminalSlash -and $i -eq $parts.Count-1 -and $i -gt 0) { continue }
        if ($part -in @('','.', '..') -or $part.EndsWith('.') -or $part.EndsWith(' ') -or
            $part -match '^(?i:CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³])(?:\.|$)') {
            throw "Noncanonical/aliased relative component: $Relative"
        }
    }
}
function LocalPath([string]$Path) {
    if (-not $IsWindows) { throw 'Windows filesystem semantics required; other OS is not a pass.' }
    if ($Path -notmatch '^[A-Za-z]:[\\/]' -or $Path -match '[\x00-\x1F\x7F]' -or $Path.Substring(2).Contains(':')) {
        throw "Not an absolute local-drive path: $Path"
    }
    $spelling=$Path.Replace('\','/')
    if ($spelling.Length -gt 3) {
        # Absolute OS roots can contain ordinary interior spaces (e.g. Program Files).
        $parts=$spelling.Substring(3).Split('/')
        for ($i=0; $i -lt $parts.Count; $i++) {
            $part=$parts[$i]
            if ($part -ceq '' -and $i -eq $parts.Count-1 -and $i -gt 0) { continue }
            if ($part -in @('','.', '..') -or $part -match '[*?"<>|]' -or $part.EndsWith('.') -or $part.EndsWith(' ') -or
                $part -match '^(?i:CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³])(?:\.|$)') { throw 'Noncanonical local path component.' }
        }
    }
    $full=[IO.Path]::GetFullPath($Path)
    if ($full.Length -eq 3) { return $full }
    return $full.TrimEnd('\')
}
function Within([string]$Path,[string]$Root) {
    $prefix=(LocalPath $Root).TrimEnd('\')+'\'
    return (LocalPath $Path).StartsWith($prefix,[StringComparison]::OrdinalIgnoreCase)
}
function AssertNotReparse([object]$Item) {
    $link=$Item.PSObject.Properties['LinkType']
    if (($Item.Attributes -band [IO.FileAttributes]::ReparsePoint) -or ($null -ne $link -and $link.Value)) {
        throw "Reparse/symlink/junction input refused: $($Item.FullName)"
    }
}
function PlainChain([string]$Path) {
    $current=LocalPath $Path; $first=$true
    while ($current) {
        # Get-Item also inspects the entry itself; Test-Path alone can hide a dangling link.
        $item=Get-Item -LiteralPath $current -Force -ErrorAction Stop
        AssertNotReparse $item
        if (-not $first -and -not $item.PSIsContainer) { throw "Non-directory ancestor: $current" }
        $first=$false
        $parent=[IO.Path]::GetDirectoryName($current)
        if (-not $parent -or $parent -ceq $current) { break }
        $current=$parent
    }
}
function PlainFile([string]$Path) {
    $full=LocalPath $Path; PlainChain $full
    $item=Get-Item -LiteralPath $full -Force -ErrorAction Stop
    if ($item.PSIsContainer -or $item -isnot [IO.FileInfo]) { throw "Not an ordinary file: $full" }
    return $full
}
function PlainDirectory([string]$Path) {
    $full=LocalPath $Path; PlainChain $full
    $item=Get-Item -LiteralPath $full -Force -ErrorAction Stop
    if (-not $item.PSIsContainer -or $item -isnot [IO.DirectoryInfo]) { throw "Not an ordinary directory: $full" }
    return $full
}
function RelativeFile([string]$Root,[string]$Relative) {
    AssertRelativeGrammar $Relative # licenses/ MUST still fail here.
    $base=PlainDirectory $Root
    $full=LocalPath (Join-Path $base $Relative)
    if (-not (Within $full $base)) { throw 'Relative file escapes root.' }
    return $full
}
function FreshDirectory([string]$Path) {
    $full=LocalPath $Path
    $null=PlainDirectory ([IO.Path]::GetDirectoryName($full))
    if (Test-Path -LiteralPath $full) { throw "Destination already exists: $full" }
    New-Item -ItemType Directory -Path $full -ErrorAction Stop | Out-Null
    return (PlainDirectory $full)
}
function NewJson([string]$Path,[object]$Value) {
    $full=LocalPath $Path; $null=PlainDirectory ([IO.Path]::GetDirectoryName($full))
    $bytes=[Text.UTF8Encoding]::new($false,$true).GetBytes((ConvertTo-Json -InputObject $Value -Depth 64))
    $stream=[IO.File]::Open($full,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try { $stream.Write($bytes,0,$bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
}
function AssertJsonProperties([System.Text.Json.JsonElement]$Element) {
    if ($Element.ValueKind -eq [System.Text.Json.JsonValueKind]::Object) {
        $names=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
        foreach ($property in $Element.EnumerateObject()) {
            if (-not $names.Add($property.Name)) { throw "Duplicate/aliased JSON property: $($property.Name)" }
            AssertJsonProperties $property.Value
        }
    } elseif ($Element.ValueKind -eq [System.Text.Json.JsonValueKind]::Array) {
        foreach ($member in $Element.EnumerateArray()) { AssertJsonProperties $member }
    }
}
function JsonElementValue([System.Text.Json.JsonElement]$Element) {
    # Preserve JSON kinds explicitly. In particular, timestamp-looking strings must
    # stay strings across PS7 versions; do not use ConvertFrom-Json date coercion.
    $kind=$Element.ValueKind
    if ($kind -eq [System.Text.Json.JsonValueKind]::Object) {
        $value=[Collections.Specialized.OrderedDictionary]::new([StringComparer]::Ordinal)
        foreach ($property in $Element.EnumerateObject()) { $value.Add($property.Name,(JsonElementValue $property.Value)) }
        return ,$value
    }
    if ($kind -eq [System.Text.Json.JsonValueKind]::Array) {
        $value=[Collections.Generic.List[object]]::new()
        foreach ($member in $Element.EnumerateArray()) { $value.Add((JsonElementValue $member)) }
        return ,([object[]]$value.ToArray())
    }
    if ($kind -eq [System.Text.Json.JsonValueKind]::String) { return ,$Element.GetString() }
    if ($kind -eq [System.Text.Json.JsonValueKind]::Number) {
        $integer=0L
        if ($Element.TryGetInt64([ref]$integer)) { return $integer }
        return $Element.GetDouble()
    }
    if ($kind -eq [System.Text.Json.JsonValueKind]::True) { return $true }
    if ($kind -eq [System.Text.Json.JsonValueKind]::False) { return $false }
    if ($kind -eq [System.Text.Json.JsonValueKind]::Null) { return ,$null }
    throw 'Unsupported JSON value kind.'
}
function StrictJson([string]$Text) {
    # JsonDocument rejects trailing values/comments/trailing commas; duplicate and
    # case-alias properties are inspected before any explicit value materialization.
    $document=[System.Text.Json.JsonDocument]::Parse($Text)
    try {
        if ($document.RootElement.ValueKind -ne [System.Text.Json.JsonValueKind]::Object) { throw 'JSON object required.' }
        AssertJsonProperties $document.RootElement
        return ,(JsonElementValue $document.RootElement)
    } finally { $document.Dispose() }
}
function Json([string]$Path) {
    $bytes=[IO.File]::ReadAllBytes((PlainFile $Path))
    return (StrictJson ([Text.UTF8Encoding]::new($false,$true).GetString($bytes)))
}
function Keys([Collections.IDictionary]$Object,[string[]]$Required,[string[]]$Optional=@()) {
    $actualNames=@($Object.PSBase.Keys)
    foreach ($name in $Required) {
        if (@($actualNames | Where-Object { $_ -ceq $name }).Count -ne 1) { throw "Missing/exact-case JSON key required: $name" }
    }
    foreach ($name in $actualNames) {
        if (-not (@($Required)+@($Optional) -ccontains $name)) { throw "Unknown JSON key: $name" }
    }
}
function SHA256Value([object]$Value) {
    if ($Value -isnot [string] -or $Value -cnotmatch '^[0-9a-f]{64}$') { throw 'Lowercase SHA256 required.' }
}
function Fingerprint([string]$Path,[string]$Relative='') {
    $full=PlainFile $Path; $before=Get-Item -LiteralPath $full -Force
    $length=$before.Length; $ticks=$before.LastWriteTimeUtc.Ticks
    $hash=(Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToLowerInvariant()
    $after=Get-Item -LiteralPath $full -Force; PlainChain $full
    if ($length -ne $after.Length -or $ticks -ne $after.LastWriteTimeUtc.Ticks) { throw "Hash input changed: $full" }
    return [pscustomobject][ordered]@{path=$Relative; absolutePath=$full; bytes=$length; sha256=$hash; lastWriteUtcTicks=$ticks}
}
function Pin([string]$Path,[string]$SHA256) {
    SHA256Value $SHA256
    $fact=Fingerprint $Path
    if ($fact.sha256 -cne $SHA256) { throw "Pin mismatch: $Path" }
    return $fact
}
function SameFingerprint([object]$Before,[object]$After) {
    if ($Before.bytes -ne $After.bytes -or $Before.sha256 -cne $After.sha256 -or $Before.lastWriteUtcTicks -ne $After.lastWriteUtcTicks) {
        throw "Protected input changed: $($Before.absolutePath)"
    }
}
function Inventory([string]$Root) {
    $full=PlainDirectory $Root; $stack=[Collections.Generic.Stack[string]]::new(); $stack.Push($full)
    $files=[Collections.Generic.List[object]]::new(); $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    while ($stack.Count) {
        foreach ($entry in Get-ChildItem -LiteralPath $stack.Pop() -Force -ErrorAction Stop) {
            AssertNotReparse $entry
            if ($entry.PSIsContainer) { $stack.Push($entry.FullName) } else {
                if ($entry -isnot [IO.FileInfo]) { throw 'Non-file inventory entry.' }
                $relative=[IO.Path]::GetRelativePath($full,$entry.FullName).Replace('\','/')
                AssertRelativeGrammar $relative
                if (-not $seen.Add($relative)) { throw "Case alias inventory member: $relative" }
                $files.Add((Fingerprint $entry.FullName $relative))
            }
        }
    }
    # Explicit ordinal sort, not locale-dependent Sort-Object.
    $ordered=[string[]]@($files.ToArray() | ForEach-Object { $_.path }); [Array]::Sort($ordered,[StringComparer]::Ordinal)
    $byPath=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($fact in $files) { $byPath.Add($fact.path,$fact) }
    return @($ordered | ForEach-Object { $byPath[$_] })
}
function SameInventory([object[]]$Before,[object[]]$After,[string]$Label) {
    if ($Before.Count -ne $After.Count) { throw "$Label file count changed." }
    $map=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($fact in $Before) { $map.Add($fact.path,$fact) }
    foreach ($fact in $After) {
        if (-not $map.ContainsKey($fact.path)) { throw "$Label path set changed." }
        SameFingerprint $map[$fact.path] $fact
    }
}
