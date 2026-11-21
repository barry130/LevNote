#requires -Version 7.0
# NOT EXECUTED for this revision. Future Windows PS7 fixtures only; no Pester or Go.
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$cases=[Collections.Generic.List[object]]::new(); $work=$null
function Assert([bool]$Condition,[string]$Message) { if (-not $Condition) { throw $Message } }
function Reject([scriptblock]$Action) {
    $rejected=$false
    try { $null=& $Action } catch { $rejected=$true }
    if (-not $rejected) { throw 'Expected rejection, but operation succeeded.' }
}
function Case([string]$Name,[scriptblock]$Action) {
    try { $null=& $Action; $cases.Add([ordered]@{name=$Name; status='PASS'}) }
    catch { $cases.Add([ordered]@{name=$Name; status='FAILED'; detail=$_.Exception.ToString(); stack=[string]$_.ScriptStackTrace}) }
}
function TextFile([string]$Path,[string]$Text) {
    $stream=[IO.File]::Open($Path,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try { $bytes=[Text.UTF8Encoding]::new($false,$true).GetBytes($Text); $stream.Write($bytes,0,$bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
}
function BytesFile([string]$Path,[byte[]]$Bytes) {
    $stream=[IO.File]::Open($Path,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try { $stream.Write($Bytes,0,$Bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
}
function NewFivePEFixture([string]$NativeRoot) {
    # Inert PE HEADER BYTES only, never runnable products and never launched.
    $root=FreshDirectory (Join-Path $work ('five-pe-'+[Guid]::NewGuid().ToString('N')))
    $evidence=FreshDirectory (Join-Path $work ('five-pe-record-'+[Guid]::NewGuid().ToString('N')))
    foreach ($directory in @('licenses','attribution','engines')) { $null=FreshDirectory (Join-Path $root $directory) }
    $assets=@(
        @((Join-Path (Split-Path $NativeRoot -Parent) 'LICENSE'),'LICENSE'),
        @((Join-Path $NativeRoot 'THIRD_PARTY_NOTICES.md'),'attribution/SOURCE_THIRD_PARTY_NOTICES.txt'),
        @((Join-Path $NativeRoot 'vendor/github.com/egoist/mygo/README.md'),'attribution/mygo-README.md'),
        @((Join-Path $NativeRoot 'vendor/github.com/ebitengine/purego/README.md'),'attribution/purego-README.md'),
        @((Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch'),'attribution/mygo-v0.8.3-no-startup-cleanup.patch'),
        @((Join-Path $NativeRoot 'dependency-patches/mygo-v0.8.3-bitmap-dispose.patch'),'attribution/mygo-v0.8.3-bitmap-dispose.patch'),
        @((Join-Path $NativeRoot 'dependency-patches/README.md'),'attribution/dependency-patches-README.md')
    )
    foreach ($name in @('Go-BSD-3-Clause.txt','go-text-typesetting.txt','mygo-MIT.txt','purego-Apache-2.0.txt','x-image-BSD-3-Clause.txt','x-net-BSD-3-Clause.txt','x-text-BSD-3-Clause.txt')) { $assets+=,@((Join-Path $NativeRoot ('licenses/'+$name)),('licenses/'+$name)) }
    foreach ($asset in $assets) { BytesFile (Join-Path $root $asset[1]) ([IO.File]::ReadAllBytes((PlainFile $asset[0]))) }
    $version='ci.fixture'; $readme=[IO.File]::ReadAllText((Join-Path $NativeRoot 'portable/README.md')).Replace('{{VERSION}}',$version)
    TextFile (Join-Path $root 'README.md') $readme
    $notice=[IO.File]::ReadAllText((Join-Path $NativeRoot 'THIRD_PARTY_NOTICES.md'))
    $notice=$notice.Replace('[LICENSE](../LICENSE)','[LICENSE](LICENSE)').Replace('(vendor/github.com/ebitengine/purego/README.md)','(attribution/purego-README.md)').Replace('(dependency-patches/mygo-v0.8.3-no-startup-cleanup.patch)','(attribution/mygo-v0.8.3-no-startup-cleanup.patch)').Replace('(dependency-patches/README.md)','(attribution/dependency-patches-README.md)').Replace('(dependency-patches/mygo-v0.8.3-bitmap-dispose.patch)','(attribution/mygo-v0.8.3-bitmap-dispose.patch)')
    TextFile (Join-Path $root 'THIRD_PARTY_NOTICES.md') ([IO.File]::ReadAllText((Join-Path $NativeRoot 'portable/NOTICE-PREAMBLE.md'))+"`n"+$notice)
    $bytes=[byte[]]::new(512)
    [BitConverter]::GetBytes([uint16]0x5A4D).CopyTo($bytes,0)
    [BitConverter]::GetBytes([int]0x80).CopyTo($bytes,0x3C)
    [BitConverter]::GetBytes([uint32]0x4550).CopyTo($bytes,0x80)
    [BitConverter]::GetBytes([uint16]0x8664).CopyTo($bytes,0x84)
    [BitConverter]::GetBytes([uint16]0x20B).CopyTo($bytes,0x98)
    [BitConverter]::GetBytes([uint16]2).CopyTo($bytes,0xDC)
    BytesFile (Join-Path $root 'levnote-native.exe') $bytes
    $definitions=@(
        @('markdown-export','levnote.markdown-export','html','levnote-markdown-export.exe'),
        @('mindmap-svg','levnote.mindmap-svg','svg','levnote-mindmap-svg.exe'),
        @('mindmap-preview','levnote.mindmap-preview','mindmap-scene-v1','levnote-mindmap-preview.exe'),
        @('drawio-svg','levnote.drawio-svg','svg','levnote-drawio-svg.exe')
    )
    $engines=@()
    foreach ($definition in $definitions) {
        $relative='engines/'+$definition[0]; $directory=FreshDirectory (Join-Path $root $relative)
        $workerPath=Join-Path $directory $definition[3]; BytesFile $workerPath $bytes; $worker=Fingerprint $workerPath
        $metadataPath=Join-Path $directory 'manifest.json'
        NewJson $metadataPath ([ordered]@{schema=1; engine=$definition[1]; engineVersion=$version; protocol=1; format=$definition[2]; goos='windows'; goarch='amd64'; filename=$definition[3]; size=$worker.bytes; sha256=$worker.sha256})
        $engines+=,[ordered]@{Engine=$definition[1]; Format=$definition[2]; RelativePath=$relative; WorkerPath=$workerPath; WorkerBytes=$worker.bytes; WorkerSHA256=$worker.sha256; ManifestPath=$metadataPath; ManifestSHA256=(Fingerprint $metadataPath).sha256}
    }
    $files=@(Inventory $root | ForEach-Object { [ordered]@{path=$_.path; size=$_.bytes; sha256=$_.sha256} })
    $manifestPath=Join-Path $root 'release-manifest.json'
    NewJson $manifestPath ([ordered]@{schema=1; version=$version; goos='windows'; goarch='amd64'; goamd64='v1'; configuration='%APPDATA%/LevNoteNative'; excludes=@('release-manifest.json'); files=$files})
    $hostFact=Fingerprint (Join-Path $root 'levnote-native.exe')
    NewJson (Join-Path $evidence 'package-result.json') ([ordered]@{Status='PACKAGED; build and asset checks passed, product acceptance is separate'; Version=$version; TargetGOOS='windows'; TargetGOARCH='amd64'; TargetGOAMD64='v1'; OutputDirectory=$root; EvidenceDirectory=$evidence; HostPath=$hostFact.absolutePath; HostBytes=$hostFact.bytes; HostSHA256=$hostFact.sha256; Engines=$engines; ReleaseManifestPath=$manifestPath; ReleaseManifestSHA256=(Fingerprint $manifestPath).sha256})
    return [pscustomobject]@{root=$root; evidence=$evidence; version=$version}
}
function E([string]$Action,[string]$Package,[string]$Test='',[string]$Output='',[string]$OutputType='') {
    $event=[ordered]@{Action=$Action; Package=$Package}
    if ($Test) { $event.Test=$Test }
    if ($Action -ceq 'output') { $event.Output=$Output; if ($OutputType) { $event.OutputType=$OutputType } }
    return $event
}
function CopyEvents([object[]]$Events) {
    $wrapper=StrictJson (ConvertTo-Json -InputObject @{events=$Events} -Depth 64)
    return ,$wrapper.events
}
function NewFixture(
    [object[]]$Events,
    [scriptblock]$MutateExpected=$null,[scriptblock]$MutateRun=$null,[scriptblock]$MutateListing=$null,
    [switch]$OverrideRaw,[string]$RawText=''
) {
    $directory=FreshDirectory (Join-Path $work ('json-'+[Guid]::NewGuid().ToString('N')))
    $target=[ordered]@{GOOS='windows'; GOARCH='amd64'; GOAMD64='v1'; goVersion='go1.27.3'}
    # Synthetic file/package identities, explicitly NOT the live product package set.
    $source=[ordered]@{schema=1; repositoryRoot=$directory; files=@(
        [ordered]@{path='native/app/fixture_test.go'; bytes=7; sha256=('1'*64)},
        [ordered]@{path='native/core/fixture/fixture_test.go'; bytes=7; sha256=('2'*64)},
        [ordered]@{path='native/go.mod'; bytes=7; sha256=('3'*64)}
    )}
    $tool=[ordered]@{schema=1; target=$target; archiveSHA256=$script:GoArchiveSHA256; archiveBytes=$script:GoArchiveBytes; executable=@{sha256=('4'*64); bytes=1}; inventory=@{sha256=('5'*64); fileCount=1}}
    $inputs=@{}
    foreach ($name in @('sourceInventory','toolchainIdentity')) {
        $path=Join-Path $directory ($name+'.json')
        $value=if ($name -ceq 'sourceInventory') { $source } else { $tool }
        NewJson $path $value; $inputs[$name]=@{path=$path; sha256=(Fingerprint $path).sha256}
    }
    $packages=@(
        [ordered]@{Package=$script:p; RelativeDirectory='app'; HasTests=$true},
        [ordered]@{Package=$script:n; RelativeDirectory='cmd/fixture'; HasTests=$false},
        [ordered]@{Package=$script:q; RelativeDirectory='core/fixture'; HasTests=$true}
    )
    $listed=@(
        [ordered]@{Package=$script:p; RelativeDirectory='app'; TestGoFiles=@('fixture_test.go'); XTestGoFiles=@()},
        [ordered]@{Package=$script:n; RelativeDirectory='cmd/fixture'; TestGoFiles=@(); XTestGoFiles=@()},
        [ordered]@{Package=$script:q; RelativeDirectory='core/fixture'; TestGoFiles=@(); XTestGoFiles=@('fixture_test.go')}
    )
    $rawListingPath=Join-Path $directory 'independent-package-listing.raw.json'
    TextFile $rawListingPath (ConvertTo-Json -InputObject $listed -Depth 20)
    $inputs.rawPackageListing=@{path=$rawListingPath; sha256=(Fingerprint $rawListingPath).sha256}
    $listing=[ordered]@{schema=1; origin='independent-package-list-before-tests'; roots=$script:ProductRoots; target=$target; modulePath='levnote.local/native'; sourceInventorySHA256=$inputs.sourceInventory.sha256; toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256; generation=@{arguments=(@('list','-json')+$script:ProductRoots); rawListingSHA256=$inputs.rawPackageListing.sha256; completedBeforeTest=$true}; packages=$listed}
    if ($null -ne $MutateListing) { $null=& $MutateListing $listing }
    $listingPath=Join-Path $directory 'package-listing.json'; NewJson $listingPath $listing
    $inputs.packageListing=@{path=$listingPath; sha256=(Fingerprint $listingPath).sha256}
    $expected=[ordered]@{schema=1; origin='independent-package-list-before-tests'; roots=$script:ProductRoots; target=$target; modulePath='levnote.local/native'; sourceInventorySHA256=$inputs.sourceInventory.sha256; toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256; packageListingSHA256=$inputs.packageListing.sha256; packages=$packages}
    if ($null -ne $MutateExpected) { $null=& $MutateExpected $expected }
    $expectedPath=Join-Path $directory 'expected-packages.json'; NewJson $expectedPath $expected
    $inputs.expectedManifest=@{path=$expectedPath; sha256=(Fingerprint $expectedPath).sha256}
    $rawPath=Join-Path $directory 'stdout.raw.jsonl'
    if (-not $OverrideRaw) { $RawText=[string]::Join("`n",[string[]]@($Events | ForEach-Object { ConvertTo-Json -InputObject $_ -Depth 20 -Compress }))+"`n" }
    TextFile $rawPath $RawText
    $inputs.rawJsonl=@{path=$rawPath; sha256=(Fingerprint $rawPath).sha256}
    $run=[ordered]@{schema=1; roots=$script:ProductRoots; target=$target; arguments=(@('test','-json','-count=1','-timeout=15m')+$script:ProductRoots); sourceInventorySHA256=$inputs.sourceInventory.sha256; toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256; expectedManifestSHA256=$inputs.expectedManifest.sha256; rawJsonlSHA256=$inputs.rawJsonl.sha256; exitCode=0; completed=$true; timedOut=$false; cleanupForced=$false; jobEmptyObserved=$true}
    if ($null -ne $MutateRun) { $null=& $MutateRun $run }
    $runPath=Join-Path $directory 'run-identity.json'; NewJson $runPath $run
    $inputs.runIdentity=@{path=$runPath; sha256=(Fingerprint $runPath).sha256}
    return [pscustomobject]@{directory=$directory; inputs=$inputs; summary=(Join-Path $directory 'summary.json')}
}
function RejectEvents([object[]]$Events) {
    $fixture=NewFixture -Events $Events
    $before=Fingerprint $fixture.inputs.rawJsonl.path
    Reject { ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary }
    Assert (-not (Test-Path -LiteralPath $fixture.summary)) 'Rejected stream wrote a success summary.'
    SameFingerprint $before (Fingerprint $fixture.inputs.rawJsonl.path)
}
try {
    if (-not $IsWindows -or [IntPtr]::Size -ne 8 -or $PSVersionTable.PSVersion.Major -ne 7 -or
        [Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString() -cne 'X64' -or
        [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -cne 'X64') { throw 'FAILED setup: native Windows x64 PowerShell7 required; no non-Windows skip.' }
    $revision=Split-Path $PSScriptRoot -Parent
    foreach ($name in @('ci-lib.ps1','markdown-targets.ps1','test-json-reducer.ps1','package-verifier.ps1')) { . (Join-Path $revision $name) }
    $repository=$PSScriptRoot
    for ($i=0; $i -lt 4; $i++) { $repository=Split-Path $repository -Parent }
    $portable=Join-Path $repository 'native/portable/README.md'; $portableBefore=Fingerprint $portable
    $portableText=[IO.File]::ReadAllText($portable,[Text.UTF8Encoding]::new($false,$true))
    $temp=PlainDirectory ([IO.Path]::GetTempPath())
    $work=FreshDirectory (Join-Path $temp ('levnote-ci-verifier-fixtures-'+[Guid]::NewGuid().ToString('N')))
    $tree=FreshDirectory (Join-Path $work 'package')
    foreach ($directory in @('licenses','attribution')) { $null=FreshDirectory (Join-Path $tree $directory) }
    foreach ($name in @('LICENSE','THIRD_PARTY_NOTICES.md','release-manifest.json')) { TextFile (Join-Path $tree $name) 'fixture' }
    TextFile (Join-Path $tree 'licenses/license.txt') 'fixture'
    Case 'canonical file and directory types' {
        Assert ((ResolveMarkdownTarget $tree 'LICENSE').kind -ceq 'File') 'File kind differs.'
        foreach ($name in @('licenses/','attribution/')) {
            $resolved=ResolveMarkdownTarget $tree $name
            Assert ($resolved.kind -ceq 'Directory' -and (Within $resolved.absolutePath $tree)) 'Not an ordinary in-root directory.'
        }
    }
    Case 'portable and generated README literal links, source bytes unchanged' {
        Assert ($portableText.Contains('[licenses/](licenses/)') -and $portableText.Contains('[attribution/](attribution/)')) 'Current portable directory link literals changed.'
        $originalLinks=@(CheckMarkdownLinks $tree $portableText 'portable/README.md')
        $generated=$portableText.Replace('{{VERSION}}','ci.fixture')
        $path=Join-Path $tree 'README.md'; TextFile $path $generated
        $generatedLinks=@(CheckMarkdownLinks $tree ([IO.File]::ReadAllText($path)) 'README.md')
        Assert ($originalLinks.Count -eq 5 -and $generatedLinks.Count -eq 5) 'README local-link set differs.'
        Assert (@($generatedLinks | Where-Object { $_.target.kind -ceq 'Directory' }).Count -eq 2) 'README directory intents lost.'
        SameFingerprint $portableBefore (Fingerprint $portable)
    }
    Case 'notices accept explicit directory target with a label without slash' {
        $links=@(CheckMarkdownLinks $tree '[licenses](licenses/) [attribution](attribution/) [LICENSE](LICENSE)' 'notice-fixture')
        Assert ($links.Count -eq 3) 'Notice fixture links missing.'
    }
    Case 'complete five-PE static checker uses common resolver with unchanged generated README' {
        $native=Join-Path $repository 'native'; $fixture=NewFivePEFixture $native
        $facts=CheckFivePEPackage $native $fixture.root $fixture.evidence $fixture.version
        Assert ($facts.status -ceq 'PACKAGED_STATIC' -and $facts.pe.Count -eq 5 -and $facts.files.Count -eq 26) 'Five-PE static identity differs.'
        Assert (@($facts.markdownLinks | Where-Object { $_.document -ceq 'README.md' -and $_.target.kind -ceq 'Directory' }).Count -eq 2) 'Five-PE checker lost README directory links.'
        SameFingerprint $portableBefore (Fingerprint $portable)
        [IO.File]::AppendAllText((Join-Path $fixture.root 'README.md'),'drift',[Text.UTF8Encoding]::new($false))
        Reject { CheckFivePEPackage $native $fixture.root $fixture.evidence $fixture.version }
    }
    Case 'fragment and HTTP classification, no local suffix splitting' {
        Assert ((ParseMarkdownTarget '#section?query').kind -ceq 'Fragment') 'Fragment treated as path.'
        Assert ((ParseMarkdownTarget 'https://example.invalid/x?q=y#z').kind -ceq 'External') 'HTTP treated as path.'
        foreach ($raw in @('LICENSE#section','LICENSE?download=1','?q=x','licenses/#part','mailto:x','LICENSE%23x')) { Reject { ResolveMarkdownTarget $tree $raw } }
    }
    foreach ($raw in @('../outside/LICENSE','../../outside/','/absolute/path','C:/outside','//server/share/','\\server\share','licenses\license.txt','','licenses//','licenses///','a//b','./LICENSE','licenses/../LICENSE','licenses/./','licenses/missing/','missing','LICENSE/','licenses','CON','name.','name ','a:b')) {
        Case ('reject raw/type '+$raw) { Reject { ResolveMarkdownTarget $tree $raw } }
    }
    Case 'RelativeFile remains file-spelling only' { Reject { RelativeFile $tree 'licenses/' }; Reject { RelativeFile $tree '../outside' }; Reject { PlainFile (RelativeFile $tree 'licenses') } }
    Case 'explicit intent disagreement rejects before filesystem probing' { Reject { ParseMarkdownTarget 'licenses/' 'File' }; Reject { ParseMarkdownTarget 'licenses' 'Directory' } }
    Case 'directory prefix file is not a directory' { TextFile (Join-Path $tree 'prefix.txt') 'fixture'; Reject { ResolveMarkdownTarget $tree 'prefix/' } }
    Case 'malformed inline link syntax fails closed' { foreach ($text in @('[x]()','[x](LICENSE','[x](a(b))','[x](LICENSE "title")')) { Reject { CheckMarkdownLinks $tree $text 'malformed' } } }
    Case 'Junction setup and both directory/file rejection (setup failure is FAILED)' {
        $outside=FreshDirectory (Join-Path $work 'junction-outside'); TextFile (Join-Path $outside 'LICENSE') 'outside'
        $package=FreshDirectory (Join-Path $work 'junction-package'); $junction=Join-Path $package 'licenses'
        try { New-Item -ItemType Junction -Path $junction -Target $outside -ErrorAction Stop | Out-Null } catch { throw "FAILED Junction setup: $($_.Exception.Message)" }
        Assert ([bool]((Get-Item -LiteralPath $junction -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) 'FAILED Junction control is not a reparse point.'
        Reject { ResolveMarkdownTarget $package 'licenses/' }
        Reject { ResolveMarkdownTarget $package 'licenses/LICENSE' }
        Reject { PlainFile (RelativeFile $package 'licenses/LICENSE') }
        Reject { ResolveMarkdownTarget $junction 'LICENSE' }
    }
    Case 'file and directory symbolic-link controls (setup failure is FAILED)' {
        $outside=FreshDirectory (Join-Path $work 'symlink-outside'); TextFile (Join-Path $outside 'LICENSE') 'outside'
        $package=FreshDirectory (Join-Path $work 'symlink-package')
        try {
            New-Item -ItemType SymbolicLink -Path (Join-Path $package 'file-link') -Target (Join-Path $outside 'LICENSE') -ErrorAction Stop | Out-Null
            New-Item -ItemType SymbolicLink -Path (Join-Path $package 'dir-link') -Target $outside -ErrorAction Stop | Out-Null
        } catch { throw "FAILED symlink setup: $($_.Exception.Message)" }
        Reject { ResolveMarkdownTarget $package 'file-link' }; Reject { ResolveMarkdownTarget $package 'dir-link/' }; Reject { ResolveMarkdownTarget $package 'dir-link/LICENSE' }
    }
    $script:p='levnote.local/native/app'; $script:n='levnote.local/native/cmd/fixture'; $script:q='levnote.local/native/core/fixture'
    $base=@(
        (E 'start' $p), (E 'start' $n), (E 'output' $n '' "?   `t$n`t[no test files]`n"),
        (E 'run' $p 'TestParent'), (E 'start' $q), (E 'run' $p 'TestParent/Child'),
        (E 'output' $p 'TestParent/Child' "fixture skip reason`n"), (E 'skip' $p 'TestParent/Child'),
        (E 'run' $q 'TestParent'), (E 'pause' $q 'TestParent'), (E 'skip' $n), (E 'cont' $q 'TestParent'),
        (E 'output' $q 'TestParent' "fixture log`n"), (E 'pass' $q 'TestParent'), (E 'pass' $q),
        (E 'pass' $p 'TestParent'), (E 'pass' $p)
    )
    Case 'complete interleaved identities, parent/child, parallel and distinct package/test keys' {
        $fixture=NewFixture -Events $base; $before=Fingerprint $fixture.inputs.rawJsonl.path
        $summary=ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary
        Assert ($summary.packagePassCount -eq 2 -and $summary.noTestPackageCount -eq 1 -and $summary.testPassCount -eq 2 -and $summary.testSkipCount -eq 1) 'Exact counts differ.'
        Assert ($summary.packages.Count -eq 3 -and $summary.tests.Count -eq 3 -and $summary.eventLineCount -eq $base.Count) 'Exact identity/line set differs.'
        Assert ($summary.testSkips[0].Package -ceq $p -and $summary.testSkips[0].Test -ceq 'TestParent/Child' -and $summary.testSkips[0].output[0].text -ceq "fixture skip reason`n") 'Skip identity/output lost.'
        Assert ($summary.packageSkips[0].Package -ceq $n -and $summary.packageSkips[0].terminal -ceq 'skip') 'No-test package treated as tested pass.'
        $saved=Json $fixture.summary
        Assert ($saved.status -ceq 'VERIFIED_CAPTURE_ONLY' -and $saved.expectedPackageIdentities.Count -eq 3) 'Saved identity summary differs.'
        SameFingerprint $before (Fingerprint $fixture.inputs.rawJsonl.path)
        Reject { ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary }
    }
    Case 'typed JSON preserves timestamp strings, booleans, empty arrays and ordinal keys' {
        $value=StrictJson '{"Time":"2026-11-19T22:00:00Z","HasTests":false,"items":[],"nested":{"count":1},"nullValue":null}'
        Assert ($value.Time -is [string] -and $value.HasTests -is [bool] -and -not $value.HasTests -and $value.items -is [array] -and $value.items.Count -eq 0 -and $null -eq $value.nullValue) 'JSON kind preservation failed.'
        Reject { Keys (StrictJson '{"Keys":["Action","Package"],"Action":"start","Package":"x"}') @('Action','Package') }
        $events=CopyEvents $base
        foreach ($event in $events) { $event.Time='2026-11-19T22:00:00.1234567Z' }
        $fixture=NewFixture -Events $events; $summary=ReduceTestJson $fixture.inputs $script:ProductRoots
        Assert ($summary.packages.Count -eq 3) 'Timestamp-bearing valid capture rejected.'
    }
    Case 'slash in a child label does not fabricate intermediate identities' {
        $events=@($base[0..14]) + @((E 'run' $p 'TestParent/Group/Label'),(E 'pass' $p 'TestParent/Group/Label')) + @($base[15..16])
        $fixture=NewFixture -Events $events; $summary=ReduceTestJson $fixture.inputs $script:ProductRoots
        Assert ($summary.tests.Count -eq 4) 'Full nested test identity collapsed.'
    }
    Case 'missing expected package' { RejectEvents @($base | Where-Object { $_.Package -cne $n }) }
    Case 'unexpected package' { RejectEvents (@((E 'start' 'levnote.local/native/core/unknown'))+$base) }
    Case 'outside roots package' { RejectEvents (@((E 'start' 'levnote.local/native/handoffs/snapshot'))+$base) }
    Case 'orphan package terminal' { RejectEvents @($base[1..16]) }
    Case 'duplicate package start' { RejectEvents (@($base[0])+$base) }
    Case 'duplicate package terminal' { RejectEvents ($base+@((E 'pass' $p))) }
    Case 'missing package terminal at EOF' { RejectEvents @($base[0..15]) }
    Case 'output only cannot satisfy completion' { RejectEvents @((E 'start' $p),(E 'output' $p '' "only output`n")) }
    Case 'package fail always rejects' { $events=CopyEvents $base; $events[16].Action='fail'; RejectEvents $events }
    Case 'test fail always rejects' { $events=CopyEvents $base; $events[13].Action='fail'; RejectEvents $events }
    Case 'orphan test terminal/output' { RejectEvents @($base[0..4]+$base[6..16]) }
    Case 'duplicate test run' { RejectEvents @($base[0..5]+@($base[5])+$base[6..16]) }
    Case 'duplicate test terminal' { RejectEvents @($base[0..7]+@($base[7])+$base[8..16]) }
    Case 'unterminated parent' { RejectEvents @($base[0..14]+@($base[16])) }
    Case 'unterminated child and premature parent completion' { RejectEvents @($base[0..6]+$base[8..16]) }
    Case 'nested run without parent' { RejectEvents @($base[0..2]+$base[4..16]) }
    Case 'test-bearing package skip is not completion' { $events=CopyEvents $base; $events[16].Action='skip'; RejectEvents $events }
    Case 'no-test package emitting Test identity' { RejectEvents @($base[0..1]+@((E 'run' $n 'TestUnexpected'))+$base[2..16]) }
    Case 'no-test package pass rejects' { $events=CopyEvents $base; $events[10].Action='pass'; RejectEvents $events }
    Case 'no-test package without exact marker' { RejectEvents @($base[0..1]+$base[3..16]) }
    Case 'wrong no-test output/identity/marker' {
        foreach ($text in @("echo [no test files]`n","?   `twrong/package`t[no test files]`n","?   `t$n`t[no tests to run]`n")) { $events=CopyEvents $base; $events[2].Output=$text; RejectEvents $events }
    }
    Case 'test skip without reason output' { RejectEvents @($base[0..5]+$base[7..16]) }
    Case 'skip framing alone is not a captured reason' { $events=CopyEvents $base; $events[6].OutputType='frame'; $events[6].Output="--- SKIP: TestParent/Child (0.00s)`n"; RejectEvents $events }
    Case 'identity-less test skip cannot become package skip' { $events=CopyEvents $base; $events[7].Remove('Test'); RejectEvents $events }
    Case 'empty Package' { $events=CopyEvents $base; $events[0].Package=''; RejectEvents $events }
    Case 'missing Package' { $events=CopyEvents $base; $events[0].Remove('Package'); RejectEvents $events }
    Case 'empty Test identity' { $events=CopyEvents $base; $events[3].Test=''; RejectEvents $events }
    Case 'unknown and unsupported actions reject independently' {
        foreach ($action in @('unknown','bench','attr','artifacts','build-output','build-fail')) { $events=CopyEvents $base; $events[12].Action=$action; RejectEvents $events }
    }
    Case 'pause/cont lifecycle invalid' { $events=CopyEvents $base; $events[9].Action='cont'; RejectEvents $events }
    Case 'JSON malformed, duplicate property, alias, nonobject and trailing data reject' {
        foreach ($raw in @('{broken',('{"Action":"start","Action":"start","Package":"'+$p+'"}'),('{"Action":"start","Package":"'+$p+'","package":"'+$p+'"}'),'[]','{} trailing'," `n")) {
            $fixture=NewFixture -Events @() -OverrideRaw -RawText $raw
            Reject { ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary }
            Assert (-not (Test-Path -LiteralPath $fixture.summary)) 'Malformed JSON wrote success summary.'
        }
    }
    Case 'raw drift fails pin, raw is not rewritten by reducer' {
        $fixture=NewFixture -Events $base; [IO.File]::AppendAllText($fixture.inputs.rawJsonl.path," `n",[Text.UTF8Encoding]::new($false))
        Reject { ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary }
    }
    Case 'expected pin and independent HasTests mismatch reject' {
        $fixture=NewFixture -Events $base; $fixture.inputs.expectedManifest.sha256='0'*64; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots }
        $fixture=NewFixture -Events $base -MutateExpected { param($value) $value.packages[0].HasTests=$false }; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots }
    }
    Case 'expected duplicate, unsorted, outside-root and missing classifications reject' {
        foreach ($change in @(
            {param($v) $v.packages[1].Package=$v.packages[0].Package},
            {param($v) $v.packages=@($v.packages[2],$v.packages[1],$v.packages[0])},
            {param($v) $v.packages[0].Package='levnote.local/native/app/sub'; $v.packages[0].RelativeDirectory='app/sub'},
            {param($v) $v.packages[0].Remove('HasTests')}
        )) { $fixture=NewFixture -Events $base -MutateExpected $change; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots } }
    }
    Case 'source/toolchain identity and target drift reject' {
        foreach ($change in @(
            {param($v) $v.sourceInventorySHA256='0'*64},
            {param($v) $v.toolchainIdentitySHA256='0'*64},
            {param($v) $v.target.GOOS='linux'},
            {param($v) $v.target.GOARCH='arm64'},
            {param($v) $v.target.GOAMD64='v3'},
            {param($v) $v.target.goVersion='go1.27.2'}
        )) { $fixture=NewFixture -Events $base -MutateExpected $change; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots } }
    }
    Case 'manifest/receipt JSON types and embedded roots cannot be coerced' {
        foreach ($change in @(
            {param($v) $v.schema=$true}, {param($v) $v.target.GOOS=$true},
            {param($v) $v.sourceInventorySHA256=$true}, {param($v) $v.packages[0].HasTests='true'},
            {param($v) $v.roots=@('./...')},
            {param($v) $v.roots=@('./cmd/...','./app/','./core/...','./formats/...','./modules/...','./ui/...')}
        )) { $fixture=NewFixture -Events $base -MutateExpected $change; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots } }
        foreach ($change in @(
            {param($v) $v.roots=@('./...')}, {param($v) $v.completed='true'},
            {param($v) $v.exitCode=$false}, {param($v) $v.schema=$true}
        )) { $fixture=NewFixture -Events $base -MutateRun $change; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots } }
    }
    Case 'run receipt must bind pre-generated identities and completed command' {
        foreach ($change in @(
            {param($v) $v.expectedManifestSHA256='0'*64}, {param($v) $v.rawJsonlSHA256='0'*64},
            {param($v) $v.sourceInventorySHA256='0'*64}, {param($v) $v.toolchainIdentitySHA256='0'*64},
            {param($v) $v.exitCode=1}, {param($v) $v.completed=$false}, {param($v) $v.timedOut=$true},
            {param($v) $v.cleanupForced=$true}, {param($v) $v.jobEmptyObserved=$false},
            {param($v) $v.arguments[4]='./...'}
        )) { $fixture=NewFixture -Events $base -MutateRun $change; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots } }
    }
    Case 'independent listing pin/ordering/provenance rejects' {
        foreach ($change in @(
            {param($v) $v.generation.completedBeforeTest=$false},
            {param($v) $v.generation.rawListingSHA256='0'*64},
            {param($v) $v.generation.arguments[2]='./...'},
            {param($v) $v.packages[0].TestGoFiles=@('missing_test.go')},
            {param($v) $v.roots=@('./...')}
        )) { $fixture=NewFixture -Events $base -MutateListing $change; Reject { ReduceTestJson $fixture.inputs $script:ProductRoots } }
    }
    Case 'root count/removal/reorder/substitution rejects before any input read or summary write' {
        $badSets=@(@('./...'),@('./app/','./cmd/...','./core/...','./formats/...','./modules/...'),@('./cmd/...','./app/','./core/...','./formats/...','./modules/...','./ui/...'),@('./app/','./cmd/...','./core/...','./formats/...','./modules/...','./...'))
        foreach ($roots in $badSets) {
            $summaryPath=Join-Path $work ('must-not-write-'+[Guid]::NewGuid().ToString('N')+'.json')
            $message=''
            try { $null=ReduceTestJson @{} $roots $summaryPath } catch { $message=$_.Exception.Message }
            Assert ($message -match 'product root') 'Did not reject root arguments first.'
            Assert (-not (Test-Path -LiteralPath $summaryPath)) 'Bad roots wrote output.'
        }
    }
    SameFingerprint $portableBefore (Fingerprint $portable)
} catch { $cases.Add([ordered]@{name='fixture setup/final preservation'; status='FAILED'; detail=$_.Exception.ToString(); stack=[string]$_.ScriptStackTrace}) }
$failed=@($cases | Where-Object { $_.status -ceq 'FAILED' }).Count
$result=[ordered]@{schema=1; status=$(if ($failed) { 'FAILED' } else { 'PASS_FIXTURES_ONLY' }); total=$cases.Count; passed=$cases.Count-$failed; failed=$failed; cases=$cases.ToArray(); retainedTempRoot=$work; windowsAcceptance='NOT_WINDOWS_ACCEPTED'; ciEnabled=$false; GoInvoked=$false; scope='Synthetic verifier fixtures only; Junction/symlink setup failure is FAILED, never skip.'}
if ($work) { NewJson (Join-Path $work 'fixture-result.json') $result }
$result | ConvertTo-Json -Depth 12
if ($failed -or -not $cases.Count) { exit 1 }
exit 0
