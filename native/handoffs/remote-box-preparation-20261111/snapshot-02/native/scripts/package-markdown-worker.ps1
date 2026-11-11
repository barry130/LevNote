#requires -Version 7.0
param(
    [Parameter(Mandatory = $true)]
    [string]$OutputDirectory,
    [Parameter(Mandatory = $true)]
    [string]$EvidenceDirectory,
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$')]
    [string]$Version
)

$ErrorActionPreference = 'Stop'
$nativeRoot = [IO.Path]::GetFullPath((Split-Path $PSScriptRoot -Parent))
$outputRoot = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutputDirectory)
$evidenceRoot = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($EvidenceDirectory)
. (Join-Path $PSScriptRoot 'portable-package-lib.ps1')
# These checks reject already present redirection. They are not a claim of
# race-safe filesystem protection against an actor changing ancestors mid-build.
Assert-NewPackageDirectories $outputRoot $evidenceRoot
Assert-PortableInputs $nativeRoot
$wrapper = Join-Path $PSScriptRoot 'go.ps1'
$shellPath = (Get-Command pwsh -ErrorAction Stop).Source
# Validate all inputs first; failed build directories and evidence stay intact.
New-Item -ItemType Directory -Path $evidenceRoot -ErrorAction Stop | Out-Null
New-Item -ItemType Directory -Path $outputRoot -ErrorAction Stop | Out-Null
Assert-PlainDirectoryChain $evidenceRoot
Assert-PlainDirectoryChain $outputRoot
# These definitions mirror the compiled catalog. They are fixed repository
# inputs, never caller-supplied commands, paths or engine registrations.
$engineDefinitions = @(
    [pscustomobject]@{ Name = 'markdown'; Engine = 'levnote.markdown-export'; Format = 'html'; RelativePath = 'engines/markdown-export'; Filename = 'levnote-markdown-export.exe'; SourcePackage = './cmd/levnote-markdown-export'; SourceMaxBytes = 8MB; PayloadMaxBytes = 64MB },
    [pscustomobject]@{ Name = 'mindmap'; Engine = 'levnote.mindmap-svg'; Format = 'svg'; RelativePath = 'engines/mindmap-svg'; Filename = 'levnote-mindmap-svg.exe'; SourcePackage = './cmd/levnote-mindmap-svg'; SourceMaxBytes = 16MB; PayloadMaxBytes = 64MB },
    [pscustomobject]@{ Name = 'mindmap-preview'; Engine = 'levnote.mindmap-preview'; Format = 'mindmap-scene-v1'; RelativePath = 'engines/mindmap-preview'; Filename = 'levnote-mindmap-preview.exe'; SourcePackage = './cmd/levnote-mindmap-preview'; SourceMaxBytes = 16MB; PayloadMaxBytes = 32MB },
    [pscustomobject]@{ Name = 'drawio'; Engine = 'levnote.drawio-svg'; Format = 'svg'; RelativePath = 'engines/drawio-svg'; Filename = 'levnote-drawio-svg.exe'; SourcePackage = './cmd/levnote-drawio-svg'; SourceMaxBytes = 32MB; PayloadMaxBytes = 64MB }
)

function Invoke-PackageBuild([string]$Name, [string]$Source, [string]$Destination, [string]$LinkFlags) {
    $runner = Join-Path $evidenceRoot ($Name + '-build.ps1')
    # Structured arguments are serialized as literal PowerShell strings. No
    # caller-controlled command, source package, executable or shell is accepted.
    $quotedWrapper = "'" + $wrapper.Replace("'", "''") + "'"
    $quotedDestination = "'" + $Destination.Replace("'", "''") + "'"
    $quotedFlags = "'" + $LinkFlags.Replace("'", "''") + "'"
    $script = "`$env:GOOS = 'windows'`n`$env:GOARCH = 'amd64'`n`$env:GOAMD64 = 'v1'`n& $quotedWrapper -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags',$quotedFlags,'-o',$quotedDestination,'$Source')`nexit `$LASTEXITCODE`n"
    [IO.File]::WriteAllText($runner, $script, [Text.UTF8Encoding]::new($false))
    $stdout = Join-Path $evidenceRoot ($Name + '.stdout.log')
    $stderr = Join-Path $evidenceRoot ($Name + '.stderr.log')
    $started = [DateTime]::UtcNow
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $process = Start-Process -FilePath $shellPath -ArgumentList @('-NoProfile', '-File', ('"' + $runner + '"')) -WorkingDirectory $nativeRoot -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru -Wait
    $watch.Stop()
    $metadata = [ordered]@{
        Name = $Name
        SourcePackage = $Source
        Destination = $Destination
        LinkFlags = $LinkFlags
        GoWrapper = $wrapper
        StartedAtUtc = $started.ToString('o')
        FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
        WallSeconds = $watch.Elapsed.TotalSeconds
        ExitCode = $process.ExitCode
        StdoutPath = $stdout
        StderrPath = $stderr
        StdoutBytes = (Get-Item -LiteralPath $stdout).Length
        StderrBytes = (Get-Item -LiteralPath $stderr).Length
    }
    $metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $evidenceRoot ($Name + '-build.json')) -Encoding utf8
    if ($process.ExitCode -ne 0) { throw "$Name build failed; preserved its captured streams in EvidenceDirectory." }
}

function Read-CheckedPE([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    $reader = [IO.BinaryReader]::new($stream)
    try {
        if ($reader.ReadUInt16() -ne 0x5A4D) { throw 'Missing DOS header.' }
        $stream.Position = 0x3C
        $offset = $reader.ReadInt32()
        if ($offset -lt 64 -or $offset + 94 -gt $stream.Length) { throw 'Invalid PE offset.' }
        $stream.Position = $offset
        if ($reader.ReadUInt32() -ne 0x00004550) { throw 'Missing PE header.' }
        $machine = $reader.ReadUInt16()
        $stream.Position = $offset + 24
        $magic = $reader.ReadUInt16()
        $stream.Position = $offset + 92
        $subsystem = $reader.ReadUInt16()
        if ($machine -ne 0x8664 -or $magic -ne 0x20B -or $subsystem -ne 2) { throw 'Built binary is not the required Windows amd64 GUI PE.' }
        return [ordered]@{ Machine = '0x8664'; OptionalHeaderMagic = '0x020B'; Subsystem = $subsystem }
    } finally { $reader.Dispose(); $stream.Dispose() }
}
$engines = @()
foreach ($definition in $engineDefinitions) {
    $engineRoot = Join-Path $outputRoot $definition.RelativePath
    New-Item -ItemType Directory -Path $engineRoot -ErrorAction Stop | Out-Null
    $worker = Join-Path $engineRoot $definition.Filename
    Invoke-PackageBuild ('worker-' + $definition.Name) $definition.SourcePackage $worker "-s -w -H=windowsgui -X main.engineVersion=$Version"
    $workerItem = Get-Item -LiteralPath $worker
    if ($workerItem.Length -le 0 -or $workerItem.Length -gt 64MB) { throw 'Worker executable exceeds the runtime cap.' }
    $workerPE = Read-CheckedPE $worker
    $manifest = [ordered]@{
        schema = 1
        engine = $definition.Engine
        engineVersion = $Version
        protocol = 1
        format = $definition.Format
        goos = 'windows'
        goarch = 'amd64'
        filename = $definition.Filename
        size = $workerItem.Length
        sha256 = (Get-FileHash -LiteralPath $worker -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    $manifestPath = Join-Path $engineRoot 'manifest.json'
    $manifestText = $manifest | ConvertTo-Json
    if ([Text.Encoding]::UTF8.GetByteCount($manifestText) -gt 16KB) { throw 'Engine manifest exceeds the runtime cap.' }
    [IO.File]::WriteAllText($manifestPath, $manifestText, [Text.UTF8Encoding]::new($false))
    $engines += [ordered]@{
        Engine = $definition.Engine
        Format = $definition.Format
        SourcePackage = $definition.SourcePackage
        RelativePath = $definition.RelativePath
        SourceMaxBytes = $definition.SourceMaxBytes
        PayloadMaxBytes = $definition.PayloadMaxBytes
        WorkerPath = $worker
        WorkerBytes = $workerItem.Length
        WorkerSHA256 = $manifest.sha256
        WorkerMtimeUtc = $workerItem.LastWriteTimeUtc.ToString('o')
        WorkerPE = $workerPE
        ManifestPath = $manifestPath
        ManifestSHA256 = (Get-FileHash -LiteralPath $manifestPath -Algorithm SHA256).Hash
    }
}
$hostExe = Join-Path $outputRoot 'levnote-native.exe'
Invoke-PackageBuild 'host' './cmd/levnote' $hostExe '-s -w -H=windowsgui'
$hostPE = Read-CheckedPE $hostExe
Write-PortableAssets $nativeRoot $outputRoot $Version
Write-PortableReleaseManifest $outputRoot $Version
Assert-PortableLocalLinks $outputRoot
$releasePath = Join-Path $outputRoot 'release-manifest.json'
[ordered]@{
    Status = 'PACKAGED; build and asset checks passed, product acceptance is separate'
    RecordCreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    Version = $Version
    TargetGOOS = 'windows'
    TargetGOARCH = 'amd64'
    TargetGOAMD64 = 'v1'
    HostPath = $hostExe
    HostBytes = (Get-Item -LiteralPath $hostExe).Length
    HostSHA256 = (Get-FileHash -LiteralPath $hostExe -Algorithm SHA256).Hash
    HostMtimeUtc = (Get-Item -LiteralPath $hostExe).LastWriteTimeUtc.ToString('o')
    HostPE = $hostPE
    Engines = $engines
    ReleaseManifestPath = $releasePath
    ReleaseManifestSHA256 = (Get-FileHash -LiteralPath $releasePath -Algorithm SHA256).Hash
    OutputDirectory = $outputRoot
    EvidenceDirectory = $evidenceRoot
    LocalAuthoredLinksChecked = $true
    SourceCopies = @(Get-PortableAssetCopies $nativeRoot | ForEach-Object {
        [ordered]@{ Source = $_.Source; RelativePath = $_.Relative; SHA256 = (Get-FileHash -LiteralPath $_.Source -Algorithm SHA256).Hash }
    })
    Scope = 'Fixed repository host plus three one-shot export engines and one readonly scene engine. Digest consistency is not publisher authentication or a sandbox.'
} | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $evidenceRoot 'package-result.json') -Encoding utf8
Write-Output $outputRoot
