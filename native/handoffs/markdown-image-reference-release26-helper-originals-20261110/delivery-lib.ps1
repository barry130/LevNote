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
            '单行', '![标签](目标)',
            'Ctrl/Cmd+点击或 Tab+Enter',
            '预览不自动载入目标图片',
            '主动点击后', '打开流程',
            '1..4096 UTF-8 字节',
            'ASCII 空白（空格、制表符、换行等）',
            '图片引用本身不渲染',
            '内联像素',
            '不获取网络资源',
            'HTML 导出保留转义后的原始语法，不生成 img 元素',
            '普通本地相对', '路径保留原始 Windows 反斜杠',
            'Windows 反斜杠和', '%xx', '原样传给宿主',
            '格式层不反转义或', '百分号解码',
            '显式', 'URL 和锚点执行自身原有的解码规则',
            '缺失或不可读目标会报告失败',
            '可读但不支持的', '文件可继续使用既有字节检查入口',
            'JPEG仅支持8位baseline（SOF0）的灰度或标准三通道图像',
            'CMYK或渐进式JPEG',
            '源文件超过16 MiB时，可读取的文件仍可做最多64 KiB的有界字节检查',
            '不能使用“另存副本”复制图片原始',
            '图片的读取、解码和',
            '复制一次只进行一项',
            '重载或“另存为”原生对话框期间',
            '旧图及旧源文件可能在替换读取被采纳或对话框',
            '结束前短暂保留',
            '这些生命周期不构成进程RSS/VRAM峰值',
            '静默900毫秒', '调度独立于绘制', '主线程繁忙或原生',
            '恢复快照不会自动保存到原文件', '本地草稿仍须按提示明确完成',
            '图片区域获焦后', '方向键每次平移32 DIP',
            '焦点返回图片区域', '调整窗口仍保留实际大小',
            '源文件无法读取且没有已载入快照时会报告失败',
            '前插列／后插列／删此列', '表格保留1至8列',
            '保留单元格的原始字节、空白、对齐标记和行尾',
            '先聚焦当前行的第一个操作', 'Tab/Shift+Tab 移动',
            'Enter/Space 执行', '列插入／删除',
            '会明确拒绝操作，可编辑完整表格源码'
        )
        Forbidden = @(
            'path-only', 'single-lane', '单车道行为',
            '列插入／删除、排序或新增表格语法',
            '按 F6 进入表格操作',
            'Ctrl/Cmd+click', 'UTF-8 bytes', 'Href',
            '不读取目标图片文件', '不渲染内联图片像素'
        )
    }
}

function Get-ReadingSourceFacts([string]$RepositoryRoot) {
    $path = Join-Path $RepositoryRoot 'native/handoffs/native-autosave-evidence-20261110/10-corrected-source-after.json'
    $fingerprint = Get-FileFingerprint $path
    if ($fingerprint.SHA256 -cne '1DCDA557B8AD218DA8D31224937066C9CD5E4588D1AB09CF371F0D06B4D55008') {
        throw 'Final15 source inventory differs from the accepted corrected validation.'
    }
    $inventory = Read-JsonFile $path
    if ($inventory.sourceCount -ne 15 -or -not $inventory.allMatch -or @($inventory.sources).Count -ne 15) {
        throw 'Final corrected source inventory is incomplete.'
    }
    $files = @(
        foreach ($entry in $inventory.sources) {
            $actual = Get-FileFingerprint (Join-Path $RepositoryRoot $entry.path)
            if ($actual.Bytes -ne $entry.bytes -or $actual.SHA256 -cne $entry.sha256) {
                throw "Accepted prior source changed: $($entry.path)"
            }
            $actual
        }
    )
    $tablePath = Join-Path $RepositoryRoot 'native/handoffs/markdown-table-columns-ui-evidence-20261110/08-final-source-after.json'
    $tableFingerprint = Get-FileFingerprint $tablePath
    if ($tableFingerprint.SHA256 -cne '2432812FA53197C42B2431BD965810AAB7C864C3A5FF24DFCD9F5318C82D2377') {
        throw 'Final11 table source inventory differs from source acceptance.'
    }
    $tables = Read-JsonFile $tablePath
    if ($tables.Count -ne 11 -or -not $tables.Pass -or @($tables.Files).Count -ne 11) {
        throw 'Final table source inventory is incomplete.'
    }
    foreach ($entry in $tables.Files) {
        $actual = Get-FileFingerprint (Join-Path $RepositoryRoot $entry.Path)
        if (-not $entry.Match -or $actual.Bytes -ne $entry.Bytes -or $actual.SHA256 -cne $entry.SHA256) {
            throw "Accepted table source changed: $($entry.Path)"
        }
        $files += $actual
    }
    $imagePath = Join-Path $RepositoryRoot 'native/handoffs/markdown-image-reference-ui-evidence-20261110/08-final-source-after.json'
    $imageFingerprint = Get-FileFingerprint $imagePath
    if ($imageFingerprint.SHA256 -cne 'D95C29FB86AF3F8E32E5871A77228B7545202B0813EB57F0CAAE1DB41FE54FFD') {
        throw 'Final12 image-reference source inventory differs from final validation.'
    }
    $images = @(Read-JsonFile $imagePath)
    if ($images.Count -ne 12) {
        throw "Final image-reference source inventory contains $($images.Count) entries, expected 12."
    }
    foreach ($entry in $images) {
        $actual = Get-FileFingerprint (Join-Path $RepositoryRoot $entry.Path)
        if ($actual.Bytes -ne $entry.Bytes -or $actual.SHA256 -cne $entry.SHA256) {
            throw "Accepted image-reference source changed: $($entry.Path)"
        }
        $files += $actual
    }
    return [ordered]@{
        Manifest = @($fingerprint, $tableFingerprint, $imageFingerprint)
        Count = $files.Count
        ImageReferenceManifest = $imageFingerprint
        ImageReferenceCount = $images.Count
        Files = $files
    }
}

function Get-ProtectedReleaseFacts([string]$NativeRoot) {
    $expected = @(
        @{ Relative = 'bin/levnote-native-worker-20261110.25'; Count = 26; Bytes = 27259765; Identity = '07391CBF37FF6406D5D4279CCED96CDBA0FDD3F5829B9685AA0066AA46CC7204' },
        @{ Relative = 'artifacts/acceptance/markdown-table-columns-release25-20261110'; Count = 146; Bytes = 55539894; Identity = 'F383C6A160D3B56F55A1EEF0B4A083693C1C09EF2F97910AE06C37DDC5BC9646' },
        @{ Relative = 'artifacts/acceptance/markdown-table-columns-release25-20261110-aggregate-capture-attempt-1'; Count = 6; Bytes = 3354; Identity = 'FCD2AE8E96784802DBA7C3A193D350634F1F8E281EC97507799B70CA3E27ABAF' }
    )
    $directories = @(
        foreach ($entry in $expected) {
            $path = Join-Path $NativeRoot $entry.Relative
            $files = @(Get-DirectoryInventory $path)
            $identity = Get-InventoryIdentity $files
            $bytes = Get-InventoryTotalBytes $files
            if ($files.Count -ne $entry.Count -or $bytes -ne $entry.Bytes -or $identity -cne $entry.Identity) {
                throw "Protected .25 inventory differs: $path"
            }
            [ordered]@{ Directory = $path; Count = $files.Count; Bytes = $bytes; Identity = $identity; Files = $files }
        }
    )
    $closed = @(
        Get-FileFingerprint (Join-Path $NativeRoot 'artifacts/acceptance/markdown-table-columns-release25-20261110-closed-inventory.json')
    )
    if ($closed[0].Bytes -ne 66343 -or
        $closed[0].SHA256 -cne '39BADBA047C552252A13AD7645FA19D93B70DDF4FA8414FEB9B8E8362B3C99DF') {
        throw 'Protected .25 external closed inventory differs.'
    }
    $hostFile = Get-FileFingerprint (Join-Path $directories[0].Directory 'levnote-native.exe')
    $manifestFile = Get-FileFingerprint (Join-Path $directories[0].Directory 'release-manifest.json')
    $manifest = Read-JsonFile $manifestFile.Path
    if ($manifest.version -cne 'native-worker.20261110.25' -or
        $hostFile.Bytes -ne 14443520 -or $hostFile.SHA256 -cne '2421D8FC98B144DCCB0ECC1CFF239A18E11DDEF05F12FF0505B307058419AE49' -or
        $manifestFile.Bytes -ne 4674 -or $manifestFile.SHA256 -cne 'CA064C3E7824EDCB179465CD8E60D982D057D2EA1D45A0E8407BD44508CF9BC2') {
        throw 'Protected .25 host or manifest differs.'
    }
    return [ordered]@{ Directories = $directories; ClosedInventory = $closed; HostFile = $hostFile; Manifest = $manifestFile }
}

function Assert-ProtectedReleaseUnchanged([object]$Before, [object]$After) {
    if (@($Before.Directories).Count -ne 3 -or @($After.Directories).Count -ne 3) {
        throw 'Complete .25 protection requires package, direct evidence, and its capture sibling.'
    }
    for ($i = 0; $i -lt 3; $i++) {
        Assert-InventoriesEqual @($Before.Directories[$i].Files) @($After.Directories[$i].Files) 'Protected .25 directory'
    }
    if (@($Before.ClosedInventory).Count -ne 1 -or @($After.ClosedInventory).Count -ne 1) {
        throw 'The .25 external closed inventory must be protected.'
    }
    Assert-InventoriesEqual @($Before.ClosedInventory) @($After.ClosedInventory) 'Protected .25 closed inventory'
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
    if ($manifestEntries.Count -ne 26) {
        throw "Pinned MyGo patch manifest contains $($manifestEntries.Count) entries, expected 26."
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
