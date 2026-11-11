$ErrorActionPreference = 'Stop'

function Assert-AbsolutePath([string]$Path, [string]$Label) {
    if ([string]::IsNullOrWhiteSpace($Path) -or -not [IO.Path]::IsPathFullyQualified($Path)) {
        throw "$Label must be an absolute path: $Path"
    }
    return [IO.Path]::GetFullPath($Path)
}

function Assert-AbsoluteDirectory([string]$Path, [string]$Label) {
    $full = Assert-AbsolutePath $Path $Label
    if (-not (Test-Path -LiteralPath $full -PathType Container)) {
        throw "$Label directory is absent: $full"
    }
    return $full
}

function Assert-AbsentAbsolutePath([string]$Path, [string]$Label) {
    $full = Assert-AbsolutePath $Path $Label
    if (Test-Path -LiteralPath $full) {
        throw "$Label must be absent: $full"
    }
    return $full
}

function Read-DeliveryContext([string]$EvidenceRoot) {
    $path = Join-Path $EvidenceRoot 'context.json'
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Delivery context is absent: $path"
    }
    return Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
}

function Assert-DeliveryExecution([object]$Context) {
    if (-not [bool]$Context.ExecutionAuthorized) {
        throw 'Delivery execution is not authorized. Stop without builds, tests, or launches.'
    }
}

function Get-FileFingerprint([string]$Path) {
    $item = Get-Item -LiteralPath $Path -ErrorAction Stop
    if ($item.PSIsContainer) {
        throw "Expected a regular file: $Path"
    }
    return [ordered]@{
        Path = $item.FullName
        Bytes = [Int64]$item.Length
        SHA256 = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash
        LastWriteUtcTicks = [Int64]$item.LastWriteTimeUtc.Ticks
    }
}

function Get-OptionalFileState([string]$Path) {
    $full = Assert-AbsolutePath $Path 'optional file state'
    if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
        return [ordered]@{
            Present = $false
            Path = $full
            Bytes = $null
            SHA256 = $null
            LastWriteUtcTicks = $null
        }
    }
    $fingerprint = Get-FileFingerprint $full
    return [ordered]@{
        Present = $true
        Path = [string]$fingerprint.Path
        Bytes = [Int64]$fingerprint.Bytes
        SHA256 = [string]$fingerprint.SHA256
        LastWriteUtcTicks = [Int64]$fingerprint.LastWriteUtcTicks
    }
}

function Test-FileStateContentEqual([object]$Before, [object]$After) {
    return ([bool]$Before.Present) -and ([bool]$After.Present) -and
        [Int64]$Before.Bytes -eq [Int64]$After.Bytes -and
        [string]$Before.SHA256 -ceq [string]$After.SHA256 -and
        [Int64]$Before.LastWriteUtcTicks -eq [Int64]$After.LastWriteUtcTicks
}

function Get-DirectoryInventory([string]$Root) {
    $rootFull = Assert-AbsoluteDirectory $Root 'inventory root'
    $trimmed = $rootFull.TrimEnd('\')
    return @(
        Get-ChildItem -LiteralPath $trimmed -File -Recurse |
            Sort-Object FullName |
            ForEach-Object {
                $fingerprint = Get-FileFingerprint $_.FullName
                $fingerprint.RelativePath = [IO.Path]::GetRelativePath($trimmed, $_.FullName).Replace('\', '/')
                [pscustomobject]$fingerprint
            }
    )
}

function Get-InventoryTotalBytes([object[]]$Inventory) {
    [Int64]$total = 0
    foreach ($entry in @($Inventory)) { $total += [Int64]$entry.Bytes }
    return $total
}

function Get-InventoryIdentity([object[]]$Inventory) {
    $lines = @(
        @($Inventory) | ForEach-Object {
            $relative = $_.RelativePath
            if ([string]::IsNullOrWhiteSpace($relative)) {
                $relative = [IO.Path]::GetFullPath([string]$_.Path)
            }
            '{0}`t{1}`t{2}`t{3}' -f $relative, [Int64]$_.Bytes, $_.SHA256, [Int64]$_.LastWriteUtcTicks
        }
    ) -join "`n"
    return [Convert]::ToHexString(
        [Security.Cryptography.SHA256]::HashData([Text.Encoding]::UTF8.GetBytes($lines))
    )
}

function Test-FingerprintEqual([object]$Before, [object]$After, [bool]$CompareTime = $true) {
    $same = $Before.Path -ceq $After.Path -and
        [Int64]$Before.Bytes -eq [Int64]$After.Bytes -and
        $Before.SHA256 -ceq $After.SHA256
    if ($CompareTime) {
        $same = $same -and [Int64]$Before.LastWriteUtcTicks -eq [Int64]$After.LastWriteUtcTicks
    }
    return $same
}

function Test-FileContentEqual([object]$Before, [object]$After) {
    return [Int64]$Before.Bytes -eq [Int64]$After.Bytes -and
        $Before.SHA256 -ceq $After.SHA256
}

function Get-PEInfo([string]$Path, [string]$SHA256) {
    $stream = [IO.File]::OpenRead($Path)
    $reader = [IO.BinaryReader]::new($stream)
    try {
        $mz = $reader.ReadUInt16()
        $stream.Position = 60
        $offset = $reader.ReadInt32()
        if ($offset -lt 64 -or $offset + 94 -gt $stream.Length) {
            return [ordered]@{ Path = $Path; SHA256 = $SHA256; Pass = $false }
        }
        $stream.Position = $offset
        $signature = $reader.ReadUInt32()
        $machine = $reader.ReadUInt16()
        $stream.Position = $offset + 24
        $magic = $reader.ReadUInt16()
        $stream.Position = $offset + 92
        $subsystem = $reader.ReadUInt16()
        return [ordered]@{
            Path = $Path
            SHA256 = $SHA256
            Machine = $machine
            Magic = $magic
            Subsystem = $subsystem
            Pass = $mz -eq 0x5A4D -and $signature -eq 0x00004550 -and
                $machine -eq 0x8664 -and $magic -eq 0x20B -and $subsystem -eq 2
        }
    } finally {
        $reader.Dispose()
        $stream.Dispose()
    }
}

function Assert-InventoriesEqual(
    [object[]]$Before,
    [object[]]$After,
    [string]$Label,
    [bool]$CompareTime = $true
) {
    $left = @($Before)
    $right = @($After)
    if ($left.Count -ne $right.Count) {
        throw "$Label inventory count changed: before=$($left.Count), after=$($right.Count)"
    }
    for ($index = 0; $index -lt $left.Count; $index++) {
        $leftRelative = [string]$left[$index].RelativePath
        $rightRelative = [string]$right[$index].RelativePath
        if ([string]::IsNullOrWhiteSpace($leftRelative)) {
            $leftRelative = [IO.Path]::GetFullPath([string]$left[$index].Path)
        }
        if ([string]::IsNullOrWhiteSpace($rightRelative)) {
            $rightRelative = [IO.Path]::GetFullPath([string]$right[$index].Path)
        }
        $same = $leftRelative -ceq $rightRelative -and
            [Int64]$left[$index].Bytes -eq [Int64]$right[$index].Bytes -and
            $left[$index].SHA256 -ceq $right[$index].SHA256
        if ($CompareTime) {
            $same = $same -and
                [Int64]$left[$index].LastWriteUtcTicks -eq [Int64]$right[$index].LastWriteUtcTicks
        }
        if (-not $same) { throw "$Label changed: $leftRelative" }
    }
}

function Assert-InventoryContainsExact(
    [object[]]$Expected,
    [object[]]$Actual,
    [string]$Label,
    [bool]$CompareTime = $true
) {
    if (@($Expected).Count -ne @($Actual).Count) {
        throw "$Label count differs: expected=$(@($Expected).Count), actual=$(@($Actual).Count)"
    }
    Assert-InventoriesEqual $Expected $Actual $Label $CompareTime
}

function Assert-Hash64([string]$Value, [string]$Label) {
    if ($Value -cnotmatch '^[0-9A-F]{64}$') {
        throw "$Label must be 64 uppercase hexadecimal characters: $Value"
    }
    return $Value
}

function Invoke-CapturedPowerShellScript(
    [string]$Name,
    [string]$ScriptPath,
    [string]$WorkingDirectory,
    [string]$EvidenceRoot
) {
    $scriptFull = [IO.Path]::GetFullPath($ScriptPath)
    $workingFull = Assert-AbsoluteDirectory $WorkingDirectory 'process working directory'
    $stdout = Join-Path $EvidenceRoot ($Name + '.stdout.log')
    $stderr = Join-Path $EvidenceRoot ($Name + '.stderr.log')
    foreach ($capturePath in @($stdout, $stderr)) {
        Assert-AbsentAbsolutePath $capturePath 'process capture' | Out-Null
    }
    $pwsh = (Get-Command pwsh -ErrorAction Stop).Source
    $arguments = @('-NoProfile', '-File', ('"' + $scriptFull + '"'))
    $started = [DateTime]::UtcNow
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $process = Start-Process -FilePath $pwsh -ArgumentList $arguments `
        -WorkingDirectory $workingFull -WindowStyle Hidden `
        -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru -Wait
    $watch.Stop()
    $record = [ordered]@{
        Schema = 1
        Name = $Name
        Command = $pwsh
        Arguments = $arguments
        ScriptPath = $scriptFull
        WorkingDirectory = $workingFull
        StartedAtUtc = $started.ToString('o')
        FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
        WallSeconds = $watch.Elapsed.TotalSeconds
        ExitCode = $process.ExitCode
        StdoutPath = $stdout
        StderrPath = $stderr
        StdoutBytes = [Int64](Get-Item -LiteralPath $stdout).Length
        StderrBytes = [Int64](Get-Item -LiteralPath $stderr).Length
        WindowStyle = 'Hidden'
    }
    $process.Dispose()
    return $record
}

function Write-JsonFile([string]$Path, [object]$Value, [int]$Depth = 12) {
    $full = Assert-AbsolutePath $Path 'JSON output path'
    $parent = Split-Path $full -Parent
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null
    }
    $text = $Value | ConvertTo-Json -Depth $Depth
    [IO.File]::WriteAllText($full, $text + "`n", [Text.UTF8Encoding]::new($false))
}

function Read-JsonFile([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Quote-PowerShellLiteral([string]$Value) {
    return "'" + $Value.Replace("'", "''") + "'"
}

function Assert-ExpectedText([string]$Text, [string]$Needle, [string]$Label) {
    if (-not $Text.Contains($Needle, [StringComparison]::Ordinal)) {
        throw "$Label is missing required text: $Needle"
    }
}

function Assert-ForbiddenText([string]$Text, [string]$Needle, [string]$Label) {
    if ($Text.Contains($Needle, [StringComparison]::Ordinal)) {
        throw "$Label contains forbidden implementation text: $Needle"
    }
}

function Get-ImageReadmeContract {
    return [ordered]@{
        Required = @(
            'PNG', 'JPEG', '适合窗口', '100%', '图片', '另存副本', '字节视图',
            '16 MiB', '8192', '8,388,608', '单图原始字节', 'RSS/VRAM',
            '会话恢复只保存绝对路径等元数据',
            '恢复时会重新',
            '读取保存的绝对路径',
            'JPEG仅支持8位baseline（SOF0）的灰度或标准三通道图像',
            'CMYK或渐进式JPEG',
            '源文件超过16 MiB时仍可做最多64 KiB的有界字节检查',
            '不能使用“另存副本”复制图片原始',
            '图片的读取、解码和',
            '复制一次只进行一项',
            '重载或“另存为”原生对话框期间',
            '旧图及旧源文件可能在替换读取被采纳或对话框',
            '结束前短暂保留',
            '这些生命周期不构成进程RSS/VRAM峰值'
        )
        Forbidden = @('path-only', 'single-lane', '单车道行为')
    }
}

function Get-FrozenBitmapFacts(
    [string]$NativeRoot,
    [string]$FrozenFilesPath,
    [string]$ExpectedFrozenFilesSHA256 = '2B1D185CB3302427E692B0FE955CE2D0AAC0BAB8F559775B8E522B56BCA10E8A'
) {
    $native = Assert-AbsoluteDirectory $NativeRoot 'native root'
    $frozen = [IO.Path]::GetFullPath($FrozenFilesPath)
    $frozenFile = Get-FileFingerprint $frozen
    if ($frozenFile.SHA256 -cne $ExpectedFrozenFilesSHA256) {
        throw "Frozen bitmap lifecycle identity differs: $($frozenFile.SHA256)"
    }
    $document = Read-JsonFile $frozen
    if ($document.schema -ne 1 -or
        -not [string]::Equals(
            [IO.Path]::GetFullPath([string]$document.sourceRoot),
            $native,
            [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Frozen bitmap lifecycle schema or source root differs.'
    }
    $entries = @($document.files)
    if ($entries.Count -ne 35 -or
        [int]$document.changedOrAdded -ne 33 -or
        [int]$document.unchangedReferences -ne 2) {
        throw "Frozen bitmap lifecycle counts differ: entries=$($entries.Count)."
    }
    $checks = @(
        foreach ($entry in $entries) {
            $relative = ([string]$entry.path).Replace('/', '\')
            $path = [IO.Path]::GetFullPath((Join-Path $native $relative))
            if (-not $path.StartsWith($native.TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase)) {
                throw "Frozen bitmap path escapes native root: $($entry.path)"
            }
            $actual = Get-FileFingerprint $path
            $pass = $actual.Bytes -eq [Int64]$entry.bytes -and
                $actual.SHA256.ToLowerInvariant() -ceq ([string]$entry.sha256).ToLowerInvariant()
            [ordered]@{
                Path = [string]$entry.path
                Role = [string]$entry.role
                ExpectedBytes = [Int64]$entry.bytes
                ExpectedSHA256 = ([string]$entry.sha256).ToUpperInvariant()
                Actual = $actual
                Pass = $pass
            }
        }
    )
    if (@($checks | Where-Object { -not $_.Pass }).Count -ne 0) {
        throw 'One or more frozen bitmap lifecycle files differ from the accepted list.'
    }
    return [ordered]@{
        FrozenFilesPath = $frozen
        FrozenFiles = $frozenFile
        Schema = [int]$document.schema
        SourceRoot = $native
        FileCount = $entries.Count
        ChangedOrAdded = [int]$document.changedOrAdded
        UnchangedReferences = [int]$document.unchangedReferences
        Entries = $entries
        Checks = $checks
    }
}

function Get-PinnedPatchFacts(
    [string]$NativeRoot,
    [object]$FrozenBitmapFacts,
    [string]$ExpectedManifestSHA256 = '9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A',
    [string]$ExpectedBitmapPatchSHA256 = '995D5505C3B5B1BD869547DF3F1AD5954C2DD57F1C9F32E53554851B5DB0D4D1'
) {
    $native = Assert-AbsoluteDirectory $NativeRoot 'native root'
    $manifestPath = Join-Path $native 'dependency-patches/mygo-v0.8.3-patch-files.json'
    $manifestFile = Get-FileFingerprint $manifestPath
    if ($manifestFile.SHA256 -cne $ExpectedManifestSHA256) {
        throw "Pinned MyGo patch manifest differs: $($manifestFile.SHA256)"
    }
    $manifest = Read-JsonFile $manifestPath
    if ($manifest.schema -ne 1 -or
        [string]$manifest.module -cne 'github.com/egoist/mygo v0.8.3') {
        throw 'Pinned MyGo patch manifest schema or module differs.'
    }
    $manifestEntries = @($manifest.files)
    if ($manifestEntries.Count -ne 25) {
        throw "Pinned MyGo patch manifest contains $($manifestEntries.Count) entries, expected 25."
    }
    $frozenByPath = [Collections.Generic.Dictionary[string, object]]::new(
        [StringComparer]::Ordinal)
    foreach ($entry in @($FrozenBitmapFacts.Entries)) {
        $frozenByPath.Add([string]$entry.path, $entry)
    }
    $checks = @(
        foreach ($entry in $manifestEntries) {
            $path = [string]$entry.path
            if (-not $frozenByPath.ContainsKey($path)) {
                throw "Patch-manifest file is absent from the frozen lifecycle list: $path"
            }
            $frozenEntry = $frozenByPath[$path]
            if (([string]$entry.sha256).ToLowerInvariant() -cne
                ([string]$frozenEntry.sha256).ToLowerInvariant()) {
                throw "Patch-manifest hash differs from the frozen lifecycle list: $path"
            }
            $actual = Get-FileFingerprint ([IO.Path]::GetFullPath(
                (Join-Path $native $path.Replace('/', '\'))))
            [ordered]@{
                Path = $path
                ExpectedSHA256 = ([string]$entry.sha256).ToUpperInvariant()
                Actual = $actual
                Pass = $actual.SHA256.ToLowerInvariant() -ceq ([string]$entry.sha256).ToLowerInvariant()
            }
        }
    )
    if (@($checks | Where-Object { -not $_.Pass }).Count -ne 0) {
        throw 'One or more patch-manifest files differ from the pinned identities.'
    }
    $bitmapPatch = Get-FileFingerprint (Join-Path $native 'dependency-patches/mygo-v0.8.3-bitmap-dispose.patch')
    $startupPatch = Get-FileFingerprint (Join-Path $native 'dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch')
    if ($bitmapPatch.SHA256 -cne $ExpectedBitmapPatchSHA256 -or
        $bitmapPatch.Bytes -ne 47176 -or
        $startupPatch.SHA256 -cne '1DD7D35B0E6CFD66510F9CF0D82F703AD7656C740BC682CCF1D0C21E0F8B110A' -or
        $startupPatch.Bytes -ne 516) {
        throw 'Startup or bitmap dependency patch identity differs from acceptance.'
    }
    return [ordered]@{
        ManifestPath = $manifestPath
        Manifest = $manifestFile
        Entries = $manifestEntries
        Checks = $checks
        BitmapPatch = $bitmapPatch
        StartupPatch = $startupPatch
        GuardedFilesMatchFrozenList = $true
    }
}

function Assert-RequiredProperties([object]$Object, [string[]]$Names, [string]$Label) {
    $actual = @($Object.PSObject.Properties.Name)
    $missing = @($Names | Where-Object { $actual -notcontains $_ })
    if ($missing.Count -ne 0) {
        throw "$Label is missing fields: $($missing -join ', ')"
    }
}

function Assert-ForbiddenProperties([object]$Object, [string[]]$Names, [string]$Label) {
    $actual = @($Object.PSObject.Properties.Name)
    $present = @($Names | Where-Object { $actual -contains $_ })
    if ($present.Count -ne 0) {
        throw "$Label contains forbidden fields: $($present -join ', ')"
    }
}

function Get-ReleaseSessionFacts(
    [string]$SessionPath,
    [string]$ExpectedTextPath,
    [string]$ExpectedImagePath
) {
    if (-not (Test-Path -LiteralPath $SessionPath -PathType Leaf)) {
        throw "Session is absent: $SessionPath"
    }
    $bytes = [IO.File]::ReadAllBytes($SessionPath)
    if ($bytes.Count -eq 0 -or
        $bytes[$bytes.Count - 1] -eq 10 -or
        $bytes[$bytes.Count - 1] -eq 13) {
        throw 'Session is empty or ends with a terminal newline.'
    }
    $session = Read-JsonFile $SessionPath
    $tabs = @($session.tabs)
    if ($session.version -ne 1 -or [int]$session.active -ne 1 -or $tabs.Count -ne 2) {
        throw "Mixed session shape differs: version=$($session.version), active=$($session.active), tabs=$($tabs.Count)"
    }
    $text = $tabs[0]
    $image = $tabs[1]
    Assert-RequiredProperties $text @('path', 'title', 'encoding', 'bom', 'pluginId', 'view') 'Text session record'
    Assert-ForbiddenProperties $text @(
        'record', 'draft', 'kind', 'pixel', 'pixels', 'bitmap', 'blob', 'base64',
        'source', 'binaryData', 'imageData') 'Text session record'
    if ([string]$text.path -cne $ExpectedTextPath -or
        [string]$text.encoding -cne 'UTF-8' -or
        $text.bom -ne $true -or
        [string]$text.pluginId -cne 'diagram-notes' -or
        [string]$text.view -cne 'preview') {
        throw 'Text session record differs from the mixed UTF-8/BOM route.'
    }
    Assert-RequiredProperties $image @(
        'record', 'kind', 'path', 'pluginId', 'view') 'Binary session record'
    Assert-ForbiddenProperties $image @(
        'draft', 'encoding', 'bom', 'originalPath', 'selection', 'scroll',
        'titleText', 'pixel', 'pixels', 'bitmap', 'blob', 'base64', 'source',
        'binaryData', 'imageData') 'Binary session record'
    $allowedBinary = @('record', 'kind', 'path', 'title', 'pluginId', 'view')
    $unknownBinary = @($image.PSObject.Properties.Name | Where-Object {
        $allowedBinary -notcontains $_
    })
    if ($unknownBinary.Count -ne 0 -or
        [string]$image.record -cne 'binary' -or
        [string]$image.kind -cne 'png' -or
        [string]$image.pluginId -cne 'image' -or
        [string]$image.view -cne 'preview' -or
        -not [IO.Path]::IsPathFullyQualified([string]$image.path) -or
        [string]$image.path -cne $ExpectedImagePath) {
        throw "Binary session record differs: record=$($image.record); kind=$($image.kind); path=$($image.path); pluginId=$($image.pluginId); view=$($image.view); unknown=$($unknownBinary -join ',')"
    }
    return [ordered]@{
        Path = [IO.Path]::GetFullPath($SessionPath)
        SessionBytes = [Int64]$bytes.Length
        SessionSHA256 = (Get-FileHash -LiteralPath $SessionPath -Algorithm SHA256).Hash
        TerminalNewline = $false
        ActiveIndex = [int]$session.active
        TabCount = $tabs.Count
        Text = [ordered]@{
            Path = [string]$text.path
            PluginID = [string]$text.pluginId
            View = [string]$text.view
            Encoding = [string]$text.encoding
            BOM = [bool]$text.bom
            HasDraft = $false
        }
        Image = [ordered]@{
            Record = [string]$image.record
            Kind = [string]$image.kind
            Path = [string]$image.path
            PluginID = [string]$image.pluginId
            View = [string]$image.view
            ForbiddenPayloadAbsent = $true
        }
        Raw = $session
    }
}
