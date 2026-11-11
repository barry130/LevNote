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

function Assert-PathsDisjoint([string]$Left, [string]$Right, [string]$Label) {
    $leftFull = [IO.Path]::GetFullPath($Left).TrimEnd('\')
    $rightFull = [IO.Path]::GetFullPath($Right).TrimEnd('\')
    if ([string]::Equals($leftFull, $rightFull, [StringComparison]::OrdinalIgnoreCase)) {
        throw "$Label paths must differ: $leftFull"
    }
    if ($leftFull.StartsWith($rightFull + '\', [StringComparison]::OrdinalIgnoreCase) -or
        $rightFull.StartsWith($leftFull + '\', [StringComparison]::OrdinalIgnoreCase)) {
        throw "$Label paths must not contain one another: $leftFull ; $rightFull"
    }
}

function Read-DeliveryContext([string]$EvidenceRoot) {
    $path = Join-Path $EvidenceRoot 'context.json'
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Delivery context is absent: $path"
    }
    return Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
}

function Assert-DeliveryExecution([object]$Context) {
    if (-not [bool]$Context.executionAuthorized) {
        throw 'Delivery execution is not authorized. Stop without builds, tests or launches.'
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
    foreach ($entry in @($Inventory)) {
        $total += [Int64]$entry.Bytes
    }
    return $total
}

function Get-InventoryIdentity([object[]]$Inventory) {
    $lines = @(
        @($Inventory) | ForEach-Object {
            '{0}`t{1}`t{2}`t{3}' -f $_.RelativePath, [Int64]$_.Bytes, $_.SHA256, [Int64]$_.LastWriteUtcTicks
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

function Test-InventoryEntryEqual([object]$Before, [object]$After, [bool]$CompareTime = $true) {
    $same = $Before.RelativePath -ceq $After.RelativePath -and
        [Int64]$Before.Bytes -eq [Int64]$After.Bytes -and
        $Before.SHA256 -ceq $After.SHA256
    if ($CompareTime) {
        $same = $same -and [Int64]$Before.LastWriteUtcTicks -eq [Int64]$After.LastWriteUtcTicks
    }
    return $same
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
        if (-not (Test-InventoryEntryEqual $left[$index] $right[$index] $CompareTime)) {
            throw "$Label changed: $($left[$index].RelativePath)"
        }
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

function Assert-InventoryContainsSubsetWithoutTime(
    [object[]]$Subset,
    [object[]]$Superset,
    [string]$Label
) {
    $byPath = [Collections.Generic.Dictionary[string, object]]::new([StringComparer]::Ordinal)
    foreach ($entry in @($Superset)) {
        $byPath[$entry.RelativePath] = $entry
    }
    foreach ($entry in @($Subset)) {
        if (-not $byPath.ContainsKey($entry.RelativePath)) {
            throw "$Label is missing $($entry.RelativePath)"
        }
        $actual = $byPath[$entry.RelativePath]
        if ([Int64]$actual.Bytes -ne [Int64]$entry.Bytes -or
            $actual.SHA256 -cne $entry.SHA256) {
            throw "$Label differs at $($entry.RelativePath)"
        }
    }
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

function Get-JsonArrayProperty([string]$Path, [string]$PropertyName) {
    $full = Assert-AbsolutePath $Path 'JSON record'
    if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
        throw "JSON record is absent: $full"
    }
    $document = $null
    try {
        $document = [Text.Json.JsonDocument]::Parse([IO.File]::ReadAllText($full))
        if ($document.RootElement.ValueKind -ne [Text.Json.JsonValueKind]::Object) {
            throw "JSON record root must be an object: $full"
        }
        $matchingProperties = @(
            foreach ($candidate in $document.RootElement.EnumerateObject()) {
                if ($candidate.Name -ceq $PropertyName) {
                    $candidate
                }
            }
        )
        if ($matchingProperties.Count -ne 1) {
            throw "JSON record must contain exactly one '$PropertyName' property; found $($matchingProperties.Count): $full"
        }
        $value = $matchingProperties[0].Value
        if ($value.ValueKind -eq [Text.Json.JsonValueKind]::Null) {
            return [pscustomobject][ordered]@{
                Kind = 'Null'
                Elements = [object[]]@()
            }
        }
        if ($value.ValueKind -ne [Text.Json.JsonValueKind]::Array) {
            throw "JSON record property '$PropertyName' must be null or an array; found $($value.ValueKind): $full"
        }
        $elements = [Collections.Generic.List[object]]::new()
        foreach ($element in $value.EnumerateArray()) {
            if ($element.ValueKind -eq [Text.Json.JsonValueKind]::Null) {
                throw "JSON record array '$PropertyName' must not contain null elements: $full"
            }
            $elements.Add(($element.GetRawText() | ConvertFrom-Json))
        }
        return [pscustomobject][ordered]@{
            Kind = 'Array'
            Elements = [object[]]$elements.ToArray()
        }
    } finally {
        if ($null -ne $document) {
            $document.Dispose()
        }
    }
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

function Get-ManagedRecord([string]$StdoutPath) {
    $text = [IO.File]::ReadAllText($StdoutPath)
    $marker = 'drawio-managed-real-smoke-record-json'
    $markerIndex = $text.IndexOf($marker, [StringComparison]::Ordinal)
    if ($markerIndex -lt 0) {
        throw "Structured managed record marker is absent from $StdoutPath"
    }
    $jsonStart = $text.IndexOf('{', $markerIndex)
    if ($jsonStart -lt 0) {
        throw "Structured managed record object is absent from $StdoutPath"
    }
    $depth = 0
    $inString = $false
    $escaped = $false
    $jsonEnd = -1
    for ($index = $jsonStart; $index -lt $text.Length; $index++) {
        $character = $text[$index]
        if ($inString) {
            if ($escaped) {
                $escaped = $false
            } elseif ($character -eq '\') {
                $escaped = $true
            } elseif ($character -eq '"') {
                $inString = $false
            }
            continue
        }
        if ($character -eq '"') {
            $inString = $true
        } elseif ($character -eq '{') {
            $depth++
        } elseif ($character -eq '}') {
            $depth--
            if ($depth -eq 0) {
                $jsonEnd = $index
                break
            }
        }
    }
    if ($jsonEnd -lt $jsonStart) {
        throw "Structured managed record JSON is incomplete in $StdoutPath"
    }
    $json = $text.Substring($jsonStart, $jsonEnd - $jsonStart + 1)
    return $json | ConvertFrom-Json
}

function Quote-PowerShellLiteral([string]$Value) {
    return "'" + $Value.Replace("'", "''") + "'"
}
