# Ordinary copy/archive/hash preparation only. No product command is invoked.
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath('F:/AIproject/LevNote').TrimEnd('\')
$prep = Join-Path $repo 'native/handoffs/remote-box-preparation-20261111'
$snapshot = Join-Path $prep 'snapshot-02'
$archivePath = Join-Path $prep 'levnote-source-inputs-20261111-02.zip'
$utf8 = [Text.UTF8Encoding]::new($false)
$started = [DateTime]::UtcNow.ToString('o')
$treeRoots = @('native/app','native/cmd','native/core','native/dependency-patches','native/examples','native/formats','native/licenses','native/modules','native/portable','native/scripts','native/ui','native/vendor')
$excludedCaptureRoot = Join-Path $repo 'native/modules/markdownview/artifacts/acceptance/markdown-cell-view-20261108-final'
$singleFiles = @('LICENSE','README.md','native/README.md','native/THIRD_PARTY_NOTICES.md','native/go.mod','native/go.sum','native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json')
function New-Json([string]$Name, $Value) {
    $target = Join-Path $prep $Name
    if (Test-Path -LiteralPath $target) { throw "Refusing to overwrite $target" }
    [IO.File]::WriteAllText($target, (ConvertTo-Json -InputObject $Value -Depth 12) + "`n", $utf8)
}
function Assert-Plain([string]$Path, [bool]$Directory) {
    $entry = Get-Item -LiteralPath $Path -Force
    if (($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or $entry.PSIsContainer -ne $Directory) { throw "Not a plain expected path: $Path" }
    $resolved = [IO.Path]::GetFullPath($entry.FullName)
    if (-not $resolved.StartsWith($repo + '\', [StringComparison]::OrdinalIgnoreCase)) { throw "Escaped repository: $Path" }
}
function Input-Paths {
    $paths = [Collections.Generic.List[string]]::new()
    foreach ($root in $treeRoots) {
        $queue = [Collections.Generic.Queue[string]]::new()
        $queue.Enqueue((Join-Path $repo $root))
        while ($queue.Count -gt 0) {
            $directory = $queue.Dequeue()
            Assert-Plain $directory $true
            foreach ($entry in Get-ChildItem -LiteralPath $directory -Force) {
                if ($entry.FullName -ceq $excludedCaptureRoot) { continue }
                if (($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Reparse input: $($entry.FullName)" }
                if ($entry.PSIsContainer) { $queue.Enqueue($entry.FullName) } else {
                    Assert-Plain $entry.FullName $false
                    $relative = $entry.FullName.Substring($repo.Length + 1).Replace('\','/')
                    $paths.Add($relative)
                }
            }
        }
    }
    foreach ($relative in $singleFiles) {
        Assert-Plain (Join-Path $repo $relative) $false
        $paths.Add($relative)
    }
    [string[]]$ordered = $paths.ToArray()
    [Array]::Sort($ordered, [StringComparer]::Ordinal)
    $seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($relative in $ordered) {
        if (-not $seen.Add($relative) -or $relative -match '(^|/)\.\.?(/|$)|[\x00-\x1f\\:]' -or $relative.StartsWith('/') -or $relative.Contains('remote-box-')) { throw "Unsafe/duplicate/excluded input: $relative" }
    }
    return $ordered
}
function Inventory([string]$Root, [string[]]$Paths) {
    foreach ($relative in $Paths) {
        $file = Join-Path $Root $relative
        $entry = Get-Item -LiteralPath $file -Force
        if ($entry.PSIsContainer -or ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Not regular: $file" }
        [pscustomobject][ordered]@{Path=$relative;Length=[long]$entry.Length;Sha256=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash}
    }
}
function Digest-Rows($Rows) {
    $text = [Text.StringBuilder]::new()
    foreach ($row in $Rows) { [void]$text.Append($row.Path).Append("`t").Append([string]$row.Length).Append("`t").Append($row.Sha256.ToLowerInvariant()).Append("`n") }
    $sha = [Security.Cryptography.SHA256]::Create()
    try { return ([BitConverter]::ToString($sha.ComputeHash($utf8.GetBytes($text.ToString())))).Replace('-','').ToLowerInvariant() } finally { $sha.Dispose() }
}
function Match-Rows($Expected, $Actual, [string]$Label) {
    if ($Expected.Count -ne $Actual.Count) { throw "$Label member count changed" }
    for ($i=0; $i -lt $Expected.Count; $i++) {
        if ($Expected[$i].Path -cne $Actual[$i].Path -or $Expected[$i].Length -ne $Actual[$i].Length -or $Expected[$i].Sha256 -cne $Actual[$i].Sha256) { throw "$Label changed $($Expected[$i].Path)" }
    }
}
$pinMap = [ordered]@{
    'native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json'='AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6'
    'native/dependency-patches/mygo-v0.8.3-patch-files.json'='9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A'
    'native/go.mod'='376860538BEFF9E47DAFBC110AFD5E2C92513C206DFC6947A3061E0A0B0308C3'
    'native/go.sum'='975E1F5B6D7E6E138DDDF44C173DF5ED1E7AA70395EAAD482ED955DFD417C3B7'
    'native/vendor/modules.txt'='D60E238D007653D4EAD22238C261087765C5D14ED4D3492A76D173E4DEC17B42'
    'native/scripts/go.ps1'='A3EFCD7DC4079641F1767341446168E6778AB07E1C4DD09A968C768D1CC1B178'
}
foreach ($pin in $pinMap.GetEnumerator()) {
    if ((Get-FileHash -LiteralPath (Join-Path $repo $pin.Key) -Algorithm SHA256).Hash -cne $pin.Value) { throw "Fixed pin changed: $($pin.Key)" }
}
$frozen = [IO.File]::ReadAllText((Join-Path $repo 'native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json')) | ConvertFrom-Json
if ($frozen.Count -ne 5) { throw 'Frozen source authority count changed' }
foreach ($entry in $frozen) {
    $source = Join-Path $repo ('native/' + $entry.Path)
    if ((Get-Item -LiteralPath $source).Length -ne $entry.Length -or (Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash -cne $entry.Sha256) { throw "Frozen source changed: $($entry.Path)" }
}
$patches = [IO.File]::ReadAllText((Join-Path $repo 'native/dependency-patches/mygo-v0.8.3-patch-files.json')) | ConvertFrom-Json
if ($patches.schema -ne 1 -or $patches.module -cne 'github.com/egoist/mygo v0.8.3' -or $patches.files.Count -ne 26) { throw 'Patch schema/module/count differs' }
foreach ($entry in $patches.files) {
    if ((Get-FileHash -LiteralPath (Join-Path $repo ('native/' + $entry.path)) -Algorithm SHA256).Hash.ToLowerInvariant() -cne $entry.sha256) { throw "Patch changed: $($entry.path)" }
}
[string[]]$paths = @(Input-Paths)
$before = @(Inventory $repo $paths)
New-Json 'origin-before-02.json' $before
if (Test-Path -LiteralPath $snapshot) { throw 'Snapshot directory exists' }
[void][IO.Directory]::CreateDirectory($snapshot)
foreach ($relative in $paths) {
    $destination = Join-Path $snapshot $relative
    [void][IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($destination))
    [IO.File]::Copy((Join-Path $repo $relative), $destination, $false)
}
$copied = @(Inventory $snapshot $paths)
Match-Rows $before $copied 'Snapshot copy'
New-Json 'inputs-02.json' $copied
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem
$zipStream = [IO.File]::Open($archivePath, [IO.FileMode]::CreateNew, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
$zip = [IO.Compression.ZipArchive]::new($zipStream, [IO.Compression.ZipArchiveMode]::Create, $false, $utf8)
try {
    foreach ($relative in $paths) {
        [void][IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, (Join-Path $snapshot $relative), $relative, [IO.Compression.CompressionLevel]::Optimal)
    }
} finally { $zip.Dispose(); $zipStream.Dispose() }
$archiveRows = [Collections.Generic.List[object]]::new()
$readZip = [IO.Compression.ZipFile]::OpenRead($archivePath)
try {
    foreach ($entry in $readZip.Entries) {
        $stream = $entry.Open()
        $sha = [Security.Cryptography.SHA256]::Create()
        try { $digest = ([BitConverter]::ToString($sha.ComputeHash($stream))).Replace('-','') } finally { $sha.Dispose(); $stream.Dispose() }
        $archiveRows.Add([pscustomobject][ordered]@{Path=$entry.FullName;Length=[long]$entry.Length;Sha256=$digest})
    }
} finally { $readZip.Dispose() }
Match-Rows $before $archiveRows.ToArray() 'Archive content'
New-Json 'archive-members-02.json' $archiveRows.ToArray()
[string[]]$afterPaths = @(Input-Paths)
if ($paths.Count -ne $afterPaths.Count) { throw 'Origin membership changed' }
for ($i=0; $i -lt $paths.Count; $i++) { if ($paths[$i] -cne $afterPaths[$i]) { throw 'Origin membership changed' } }
$after = @(Inventory $repo $afterPaths)
New-Json 'origin-after-02.json' $after
Match-Rows $before $after 'Origin after archive'
$summary = [ordered]@{
    Schema=1;Operation='ordinary byte copy/archive/hash; no product execution';StartedUtc=$started;FinishedUtc=[DateTime]::UtcNow.ToString('o')
    Repository='F:/AIproject/LevNote';PathBase='repository root; forward slash';SnapshotDirectory='snapshot-02';TreeRoots=$treeRoots;SingleFiles=$singleFiles
    Files=$before.Count;Bytes=[long](($before | Measure-Object -Property Length -Sum).Sum);InputIdentity=(Digest-Rows $before)
    IdentityAlgorithm='SHA256 of UTF8 sorted-ordinal records Path + TAB + decimal Length + TAB + lowercase Sha256 + LF'
    Manifest=[ordered]@{Path='inputs-02.json';Length=(Get-Item -LiteralPath (Join-Path $prep 'inputs-02.json')).Length;Sha256=(Get-FileHash -LiteralPath (Join-Path $prep 'inputs-02.json') -Algorithm SHA256).Hash}
    Archive=[ordered]@{Path=[IO.Path]::GetFileName($archivePath);Length=(Get-Item -LiteralPath $archivePath).Length;Sha256=(Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash}
    SourceCopyArchiveSourceAfterIdentical=$true;FixedPinCount=$pinMap.Count;PickerFiles=5;PatchFiles=26;ReparseOrCollisionCount=0
    Excluded=@('native/modules/markdownview/artifacts/acceptance/markdown-cell-view-20261108-final (4 historical output PNGs, 121014 bytes)','all new remote-box preparation/results/preflight','native/tools','native/build-cache','native/bin','native/artifacts','native/native/handoffs','native/drawio-resources.test.exe','other historical handoffs','legacy web/Tauri tree','Git/credentials/configuration','nondependency historical validation docs')
}
New-Json 'snapshot-verification-02.json' $summary
$summary | ConvertTo-Json -Depth 8