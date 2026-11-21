#requires -Version 7.0
# Verification only; no package/build/worker invocation. Requires ci-lib + markdown-targets.
Set-StrictMode -Version Latest
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
    $native=PlainDirectory $NativeRoot; $root=PlainDirectory $Bundle
    $record=Json (Join-Path (PlainDirectory $PackageEvidence) 'package-result.json')
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
    foreach ($name in @('levnote-native.exe','README.md','THIRD_PARTY_NOTICES.md')) { $null=$required.Add($name) }
    $assets=@(
        @((Join-Path (Split-Path $native -Parent) 'LICENSE'),'LICENSE'),
        @((Join-Path $native 'THIRD_PARTY_NOTICES.md'),'attribution/SOURCE_THIRD_PARTY_NOTICES.txt'),
        @((Join-Path $native 'vendor/github.com/egoist/mygo/README.md'),'attribution/mygo-README.md'),
        @((Join-Path $native 'vendor/github.com/ebitengine/purego/README.md'),'attribution/purego-README.md'),
        @((Join-Path $native 'dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch'),'attribution/mygo-v0.8.3-no-startup-cleanup.patch'),
        @((Join-Path $native 'dependency-patches/mygo-v0.8.3-bitmap-dispose.patch'),'attribution/mygo-v0.8.3-bitmap-dispose.patch'),
        @((Join-Path $native 'dependency-patches/README.md'),'attribution/dependency-patches-README.md')
    )
    foreach ($name in @('Go-BSD-3-Clause.txt','go-text-typesetting.txt','mygo-MIT.txt','purego-Apache-2.0.txt','x-image-BSD-3-Clause.txt','x-net-BSD-3-Clause.txt','x-text-BSD-3-Clause.txt')) {
        $assets+=,@((Join-Path $native ('licenses/'+$name)),('licenses/'+$name))
    }
    foreach ($asset in $assets) {
        $null=$required.Add($asset[1]); $before=Fingerprint $asset[0]; $copy=Fingerprint (RelativeFile $root $asset[1])
        if ($before.bytes -ne $copy.bytes -or $before.sha256 -cne $copy.sha256) { throw "Shipped attribution/license copy differs: $($asset[1])" }
    }
    $sourceReadme=PlainFile (Join-Path $native 'portable/README.md')
    $expectedReadme=[Text.UTF8Encoding]::new($false,$true).GetBytes([IO.File]::ReadAllText($sourceReadme,[Text.UTF8Encoding]::new($false,$true)).Replace('{{VERSION}}',$Version))
    $actualReadme=[IO.File]::ReadAllBytes((PlainFile (Join-Path $root 'README.md')))
    if ($actualReadme.Length -ne $expectedReadme.Length -or [Convert]::ToBase64String($actualReadme) -cne [Convert]::ToBase64String($expectedReadme)) { throw 'Generated portable README bytes differ from source/version.' }
    $links=[Collections.Generic.List[object]]::new()
    foreach ($name in @('README.md','THIRD_PARTY_NOTICES.md')) {
        foreach ($link in @(CheckMarkdownLinks $root ([IO.File]::ReadAllText((PlainFile (Join-Path $root $name)),[Text.UTF8Encoding]::new($false,$true))) $name)) { $links.Add($link) }
    }
    if ($record.Status -cne 'PACKAGED; build and asset checks passed, product acceptance is separate' -or $record.Version -cne $Version -or
        $record.TargetGOOS -cne 'windows' -or $record.TargetGOARCH -cne 'amd64' -or $record.TargetGOAMD64 -cne 'v1' -or @($record.Engines).Count -ne 4 -or
        (LocalPath $record.OutputDirectory) -cne $root -or (LocalPath $record.EvidenceDirectory) -cne (LocalPath $PackageEvidence)) { throw 'Package result identity differs.' }
    $peFacts=[Collections.Generic.List[object]]::new(); $hostFact=Fingerprint (Join-Path $root 'levnote-native.exe')
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
        $null=Pin $metadataPath $match[0].ManifestSHA256.ToLowerInvariant()
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
    $null=Pin $manifestPath $record.ReleaseManifestSHA256.ToLowerInvariant()
    return [ordered]@{status='PACKAGED_STATIC'; windowsAcceptance='NOT_WINDOWS_ACCEPTED'; scope='Exact host + four fixed worker GUI PEs, full25 assets and self-excluded manifest; no launch'; pe=$peFacts.ToArray(); markdownLinks=$links.ToArray(); files=@(Inventory $root)}
}
