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
