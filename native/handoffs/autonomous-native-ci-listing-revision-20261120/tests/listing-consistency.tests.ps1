#requires -Version 7.0
# Task57 synthetic controls. ALL UNEXECUTED in this source-only delivery.
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'

function FixtureBytes([string]$Name) {
    $path=PlainFile (Join-Path (Join-Path $PSScriptRoot 'fixtures') $Name)
    if ((Get-Item -LiteralPath $path -Force).Length -gt 64KB) { throw 'Checked-in fixture exceeds harness seed limit.' }
    return ,([IO.File]::ReadAllBytes($path))
}
function NewListingPlan([string]$RawFile='listing-two-packages.raw.jsonstream') {
    # Handwritten expected truth, independent of raw parsing/projection. The extra
    # inventory test files stand for files excluded by hypothetical build tags.
    return @{
        RawBytes=(FixtureBytes $RawFile); Capture=$script:appCapture+$script:secondCapture; Lengths=@{}; InputBytes=@{}
        Source=@{schema=1; repositoryRoot='F:/Task57-Fictitious/LevNote'; files=@(
            @{path='native/app/external_test.go'; bytes=1; sha256=('1'*64)},
            @{path='native/app/internal_test.go'; bytes=1; sha256=('1'*64)},
            @{path='native/app/ordering_test.go'; bytes=1; sha256=('1'*64)},
            @{path='native/app/unused_test.go'; bytes=1; sha256=('1'*64)},
            @{path='native/core/second/second.go'; bytes=1; sha256=('1'*64)},
            @{path='native/core/second/second_test.go'; bytes=1; sha256=('1'*64)},
            @{path='native/go.mod'; bytes=1; sha256=('2'*64)}
        )}
        Listed=@(
            @{Package='levnote.local/native/app'; RelativeDirectory='app'; TestGoFiles=@('internal_test.go','ordering_test.go'); XTestGoFiles=@('external_test.go')},
            @{Package='levnote.local/native/core/second'; RelativeDirectory='core/second'; TestGoFiles=@(); XTestGoFiles=@()}
        )
        Expected=@(
            @{Package='levnote.local/native/app'; RelativeDirectory='app'; HasTests=$true},
            @{Package='levnote.local/native/core/second'; RelativeDirectory='core/second'; HasTests=$false}
        )
    }
}
function ChangeRaw([Collections.IDictionary]$Plan,[string]$Old,[string]$New) {
    $text=$script:utf8.GetString($Plan.RawBytes)
    if (-not $text.Contains($Old)) { throw "Missing mutation anchor: $Old" }
    $Plan.RawBytes=$script:utf8.GetBytes($text.Replace($Old,$New))
}
function SetRawArray([Collections.IDictionary]$Plan,[string]$Field,[string]$JsonArray) {
    # Only mutations of the fixed seed, never a normalizer/expected-truth producer.
    $text=$script:utf8.GetString($Plan.RawBytes)
    $pattern='"'+$Field+'":\s*\[[^\]]*\]'
    if (-not [regex]::IsMatch($text,$pattern)) { throw 'Missing fixed array anchor.' }
    $Plan.RawBytes=$script:utf8.GetBytes([regex]::Replace($text,$pattern,('"'+$Field+'": '+$JsonArray)))
}
function WriteFixtureInput([string]$Directory,[string]$Name,[object]$Value,[Collections.IDictionary]$Plan) {
    $path=Join-Path $Directory ($Name+'.input')
    if ($Plan.InputBytes.ContainsKey($Name)) { $bytes=$Plan.InputBytes[$Name] }
    elseif ($Name -ceq 'rawPackageListing') { $bytes=$Plan.RawBytes }
    elseif ($Name -ceq 'rawJsonl') { $bytes=$script:utf8.GetBytes($Plan.Capture) }
    else { $bytes=$script:utf8.GetBytes((ConvertTo-Json -InputObject $Value -Depth 64)) }
    $stream=[IO.File]::Open($path,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try {
        if ($Plan.Lengths.ContainsKey($Name)) {
            # Bounded future-runtime generation: logical length, no giant string
            # or byte allocation. Each override is exactly its documented cap + 1.
            $stream.SetLength($Plan.Lengths[$Name])
        } else { $stream.Write($bytes,0,$bytes.Length) }
        $stream.Flush($true)
    } finally { $stream.Dispose() }
    return @{path=$path; sha256=(Fingerprint $path).sha256}
}
function NewListingFixture([Collections.IDictionary]$Plan) {
    $directory=FreshDirectory (Join-Path $script:fixtureRoot ([Guid]::NewGuid().ToString('N')))
    $target=@{GOOS='windows'; GOARCH='amd64'; GOAMD64='v1'; goVersion='go1.27.3'}
    $inputs=@{}
    $inputs.sourceInventory=WriteFixtureInput $directory 'sourceInventory' $Plan.Source $Plan
    $tool=@{schema=1; target=$target; archiveSHA256=$script:GoArchiveSHA256; archiveBytes=$script:GoArchiveBytes; executable=@{sha256=('3'*64); bytes=1}; inventory=@{sha256=('4'*64); fileCount=1}}
    $inputs.toolchainIdentity=WriteFixtureInput $directory 'toolchainIdentity' $tool $Plan
    $inputs.rawPackageListing=WriteFixtureInput $directory 'rawPackageListing' $null $Plan
    $common=@{schema=1; origin='independent-package-list-before-tests'; roots=$script:ProductRoots; target=$target; modulePath='levnote.local/native'; sourceInventorySHA256=$inputs.sourceInventory.sha256; toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256}
    $listing=$common.Clone(); $listing.packages=$Plan.Listed
    $listing.generation=@{arguments=(@('list','-json')+$script:ProductRoots); rawListingSHA256=$inputs.rawPackageListing.sha256; completedBeforeTest=$true}
    $inputs.packageListing=WriteFixtureInput $directory 'packageListing' $listing $Plan
    $expected=$common.Clone(); $expected.packages=$Plan.Expected; $expected.packageListingSHA256=$inputs.packageListing.sha256
    $inputs.expectedManifest=WriteFixtureInput $directory 'expectedManifest' $expected $Plan
    $inputs.rawJsonl=WriteFixtureInput $directory 'rawJsonl' $null $Plan
    # Coherent synthetic receipts bind every mutation, but attest to no process.
    $receipt=@{schema=1; roots=$script:ProductRoots; target=$target; arguments=(@('test','-json','-count=1','-timeout=15m')+$script:ProductRoots); sourceInventorySHA256=$inputs.sourceInventory.sha256; toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256; expectedManifestSHA256=$inputs.expectedManifest.sha256; rawJsonlSHA256=$inputs.rawJsonl.sha256; exitCode=0; completed=$true; timedOut=$false; cleanupForced=$false; jobEmptyObserved=$true}
    $inputs.runIdentity=WriteFixtureInput $directory 'runIdentity' $receipt $Plan
    return @{inputs=$inputs; summary=(Join-Path $directory 'summary.json')}
}
function CheckListingControl([string]$Name,[scriptblock]$Mutate={},[string]$Reject='',
    [string]$RawFile='listing-two-packages.raw.jsonstream',[string[]]$Roots=$script:ProductRoots) {
    $plan=NewListingPlan $RawFile
    $null=& $Mutate $plan
    $fixture=NewListingFixture $plan
    $before=@{}
    foreach ($inputName in $fixture.inputs.Keys) { $before[$inputName]=Fingerprint $fixture.inputs[$inputName].path }
    $actual=''; $result=$null
    try { $result=ReduceTestJson $fixture.inputs $Roots $fixture.summary } catch { $actual=$_.Exception.Message }
    finally {
        foreach ($inputName in $before.Keys) { SameFingerprint $before[$inputName] (Fingerprint $fixture.inputs[$inputName].path) }
    }
    if ($Reject) {
        if ($actual -cne $Reject -or (Test-Path -LiteralPath $fixture.summary)) { throw "Unexpected rejection for $Name. Expected [$Reject], got [$actual]." }
        $script:rejected++
    } else {
        if ($actual) { throw "Unexpected rejection for ${Name}: $actual" }
        $saved=Json $fixture.summary
        if ($result.status -cne 'VERIFIED_CAPTURE_ONLY' -or $saved.windowsAcceptance -cne 'NOT_WINDOWS_ACCEPTED' -or $saved.ciEnabled -or
            $result.eventLineCount -ne 11 -or $result.packagePassCount -ne 1 -or $result.noTestPackageCount -ne 1 -or
            $result.testPassCount -ne 3 -or $result.testSkipCount -ne 0 -or $result.packages.Count -ne 2 -or $result.tests.Count -ne 3) { throw "Bad summary for $Name" }
        $truth=@(
            @{Test='TestParent'; runLine=2; terminalLine=7},
            @{Test='TestParent/Group'; runLine=3; terminalLine=4},
            @{Test='TestParent/Group/Label'; runLine=5; terminalLine=6}
        )
        for ($i=0; $i -lt $truth.Count; $i++) {
            $row=$result.tests[$i]; $known=$truth[$i]
            if ($row.Package -cne 'levnote.local/native/app' -or $row.Test -cne $known.Test -or $row.runLine -ne $known.runLine -or
                $row.terminalLine -ne $known.terminalLine -or $row.terminal -cne 'pass' -or $row.pauseCount -ne 0 -or $row.contCount -ne 0) { throw "Bad R1 identity for $Name" }
        }
        if ($result.packages[0].Package -cne 'levnote.local/native/app' -or -not $result.packages[0].HasTests -or $result.packages[0].terminalLine -ne 8 -or
            $result.packageSkips[0].Package -cne 'levnote.local/native/core/second' -or $result.packageSkips[0].HasTests -or $result.packageSkips[0].startLine -ne 9 -or
            $result.packageSkips[0].terminalLine -ne 11 -or $result.packageSkips[0].terminal -cne 'skip' -or
            $result.packageSkips[0].output[0].text -cne "?   `tlevnote.local/native/core/second`t[no test files]`n") { throw "Bad no-test witness for $Name" }
        if ($result.inputPins.Count -ne 7 -or $saved.inputPins.Count -ne 7 -or $saved.status -cne $result.status -or
            $result.expectedPackageIdentities.Count -ne 2 -or $result.expectedPackageIdentities[0] -cne 'levnote.local/native/app' -or
            $result.expectedPackageIdentities[1] -cne 'levnote.local/native/core/second') { throw 'Summary omitted/changed input/package identities.' }
        foreach ($view in @($result,$saved)) {
            $seenPins=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
            foreach ($pin in $view.inputPins) {
                if (-not $before.ContainsKey($pin.name) -or -not $seenPins.Add($pin.name)) { throw 'Summary has unknown/duplicate input pin.' }
                SameFingerprint $before[$pin.name] $pin.fact
            }
            if ($seenPins.Count -ne 7) { throw 'Summary pin set incomplete.' }
        }
        $script:accepted++
    }
    Write-Output "PASS $Name"
}

# No fixture setup, library load or public call is permitted before this guard.
if (-not $IsWindows -or $PSVersionTable.PSVersion.Major -ne 7 -or
    [Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString() -cne 'X64' -or
    [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -cne 'X64') { throw 'Fixture setup requires native Windows x64 PowerShell 7.' }
$revision=Split-Path $PSScriptRoot -Parent
. (Join-Path $revision 'ci-lib.ps1')
. (Join-Path $revision 'raw-package-listing.ps1')
. (Join-Path $revision 'test-json-reducer.ps1')
$script:utf8=[Text.UTF8Encoding]::new($false,$true)
$script:fixtureRoot=FreshDirectory (Join-Path (PlainDirectory ([IO.Path]::GetTempPath())) ('levnote-listing-fixtures-'+[Guid]::NewGuid().ToString('N')))
$script:accepted=0; $script:rejected=0
$fixturePins=@{}
foreach ($name in @('listing-two-packages.raw.jsonstream','listing-truncated.reject.jsonstream','listing-case-alias.reject.jsonstream','listing-incomplete.reject.jsonstream','listing-outside-roots.reject.jsonstream')) {
    $fixturePins[$name]=Fingerprint (Join-Path (Join-Path $PSScriptRoot 'fixtures') $name)
}
# Actual event truth is handwritten, with a slash-label sibling running after
# Group has completed, while the exact top-level TestParent remains live.
$script:appCapture=@'
{"Action":"start","Package":"levnote.local/native/app"}
{"Action":"run","Package":"levnote.local/native/app","Test":"TestParent"}
{"Action":"run","Package":"levnote.local/native/app","Test":"TestParent/Group"}
{"Action":"pass","Package":"levnote.local/native/app","Test":"TestParent/Group"}
{"Action":"run","Package":"levnote.local/native/app","Test":"TestParent/Group/Label"}
{"Action":"pass","Package":"levnote.local/native/app","Test":"TestParent/Group/Label"}
{"Action":"pass","Package":"levnote.local/native/app","Test":"TestParent"}
{"Action":"pass","Package":"levnote.local/native/app"}
'@+"`n"
$script:secondCapture=@'
{"Action":"start","Package":"levnote.local/native/core/second"}
{"Action":"output","Package":"levnote.local/native/core/second","Output":"?   \tlevnote.local/native/core/second\t[no test files]\n"}
{"Action":"skip","Package":"levnote.local/native/core/second"}
'@+"`n"
$script:noTestApp=@'
{"Action":"start","Package":"levnote.local/native/app"}
{"Action":"output","Package":"levnote.local/native/app","Output":"?   \tlevnote.local/native/app\t[no test files]\n"}
{"Action":"skip","Package":"levnote.local/native/app"}
'@+"`n"
$seed=$script:utf8.GetString((FixtureBytes 'listing-two-packages.raw.jsonstream'))
$boundary=$seed.IndexOf("`n{`n",[StringComparison]::Ordinal)
if ($boundary -lt 0) { throw 'Missing fixed seed object boundary.' }
$script:firstObject=$seed.Substring(0,$boundary+1)
$script:secondObject=$seed.Substring($boundary+1)

CheckListingControl 'two raw packages, omitted arrays, R1 sibling and exact no-test witness'
CheckListingControl 'raw package order irrelevant' { param($p) $p.RawBytes=$script:utf8.GetBytes($script:secondObject+$script:firstObject) }
CheckListingControl 'ordinal filename sets allow different order' { param($p) $p.Listed[0].TestGoFiles=@('ordering_test.go','internal_test.go') }
CheckListingControl 'absent internal array independently empty' { param($p) SetRawArray $p 'TestGoFiles' '[]'; ChangeRaw $p '"TestGoFiles": []' '"IgnoredEmptyInternal": []'; $p.Listed[0].TestGoFiles=@() }
CheckListingControl 'absent external array independently empty' { param($p) SetRawArray $p 'XTestGoFiles' '[]'; ChangeRaw $p '"XTestGoFiles": []' '"IgnoredEmptyExternal": []'; $p.Listed[0].XTestGoFiles=@() }
CheckListingControl 'real omitted boolean flags may be explicit false' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "Incomplete": false, "DepOnly": false, "ForTest": "",'; ChangeRaw $p '"Main": true,' '"Main": true, "Indirect": false, "Version": "",' }
CheckListingControl 'lexical slash style equivalent' { param($p) ChangeRaw $p 'F:\\Task57-Fictitious\\LevNote\\native' 'F:/Task57-Fictitious/LevNote/native' }
CheckListingControl 'depth 64 boundary in ignored value' { param($p) ChangeRaw $p '"Name": "app",' ('"Name": "app", "IgnoredDepth": '+('['*63)+'0'+(']'*63)+',') }

CheckListingControl 'truncated final object' -RawFile 'listing-truncated.reject.jsonstream' -Reject 'Truncated raw package listing object.'
CheckListingControl 'Windows case alias rows' -RawFile 'listing-case-alias.reject.jsonstream' -Reject 'Windows case alias raw package identity: levnote.local/native/App'
CheckListingControl 'incomplete package row' -RawFile 'listing-incomplete.reject.jsonstream' -Reject 'Raw package requires absent/false Incomplete.'
CheckListingControl 'outside exact six roots' -RawFile 'listing-outside-roots.reject.jsonstream' -Reject 'Package outside exact six roots/module: levnote.local/native/outside'

CheckListingControl 'coherently omitted raw second package' { param($p) $p.Listed=@($p.Listed[0]); $p.Expected=@($p.Expected[0]); $p.Capture=$script:appCapture } -Reject 'Raw package omitted from normalized listing: levnote.local/native/core/second'
CheckListingControl 'coherently extra normalized package' { param($p)
    $p.Listed+=@{Package='levnote.local/native/core/third'; RelativeDirectory='core/third'; TestGoFiles=@(); XTestGoFiles=@()}
    $p.Expected+=@{Package='levnote.local/native/core/third'; RelativeDirectory='core/third'; HasTests=$false}
    $p.Capture+=$script:secondCapture.Replace('core/second','core/third')
} -Reject 'Normalized package absent from raw listing: levnote.local/native/core/third'
CheckListingControl 'coherent no-test reclassification' { param($p) $p.Listed[0].TestGoFiles=@(); $p.Listed[0].XTestGoFiles=@(); $p.Expected[0].HasTests=$false; $p.Capture=$script:noTestApp+$script:secondCapture } -Reject 'Raw/expected HasTests differs: levnote.local/native/app'
CheckListingControl 'raw no-test package actually has a test file' { param($p) ChangeRaw $p '"Name": "second",' '"Name": "second", "TestGoFiles": ["second_test.go"],' } -Reject 'Raw/expected HasTests differs: levnote.local/native/core/second'
CheckListingControl 'coherent normalized filename omission' { param($p) $p.Listed[0].TestGoFiles=@('internal_test.go') } -Reject 'Raw/normalized TestGoFiles differ: levnote.local/native/app'
CheckListingControl 'coherent normalized filename extra with source membership' { param($p) $p.Listed[0].TestGoFiles+=@('unused_test.go') } -Reject 'Raw/normalized TestGoFiles differ: levnote.local/native/app'
CheckListingControl 'coherent internal external swap' { param($p) $p.Listed[0].TestGoFiles=@('external_test.go'); $p.Listed[0].XTestGoFiles=@('internal_test.go','ordering_test.go') } -Reject 'Raw/normalized TestGoFiles differ: levnote.local/native/app'
CheckListingControl 'one internal file moved to external' { param($p) $p.Listed[0].TestGoFiles=@('ordering_test.go'); $p.Listed[0].XTestGoFiles=@('external_test.go','internal_test.go') } -Reject 'Raw/normalized TestGoFiles differ: levnote.local/native/app'
CheckListingControl 'raw external filename omitted only' { param($p) SetRawArray $p 'XTestGoFiles' '[]' } -Reject 'Raw/normalized XTestGoFiles differ: levnote.local/native/app'
CheckListingControl 'coherent companion directory moved outside exact app root' { param($p) $p.Listed[0].Package='levnote.local/native/app/sub'; $p.Listed[0].RelativeDirectory='app/sub'; $p.Expected[0].Package='levnote.local/native/app/sub'; $p.Expected[0].RelativeDirectory='app/sub' } -Reject 'Package outside exact six roots/module: levnote.local/native/app/sub'
CheckListingControl 'coherent accepted-root package and directory reassignment' { param($p) $p.Listed[1].Package='levnote.local/native/core/moved'; $p.Listed[1].RelativeDirectory='core/moved'; $p.Expected[1].Package='levnote.local/native/core/moved'; $p.Expected[1].RelativeDirectory='core/moved'; $p.Capture=$script:appCapture+$script:secondCapture.Replace('core/second','core/moved') } -Reject 'Raw package omitted from normalized listing: levnote.local/native/core/second'
CheckListingControl 'raw Dir mismatch' { param($p) ChangeRaw $p '"Dir": "F:\\Task57-Fictitious\\LevNote\\native\\app"' '"Dir": "F:\\Task57-Fictitious\\LevNote\\native\\core\\second"' } -Reject 'Raw package Dir differs: levnote.local/native/app'
CheckListingControl 'raw Dir case mismatch' { param($p) ChangeRaw $p '\\native\\app"' '\\native\\App"' } -Reject 'Raw package Dir differs: levnote.local/native/app'
CheckListingControl 'source repository root mismatch' { param($p) $p.Source.repositoryRoot='F:/Task57-Other/LevNote' } -Reject 'Raw package Dir differs: levnote.local/native/app'
CheckListingControl 'raw Module.Path mismatch' { param($p) ChangeRaw $p '"Path": "levnote.local/native"' '"Path": "other.local/native"' } -Reject 'Raw Module.Path differs.'
CheckListingControl 'raw Module.Dir mismatch' { param($p) ChangeRaw $p '"Dir": "F:\\Task57-Fictitious\\LevNote\\native"' '"Dir": "F:\\Task57-Other\\LevNote\\native"' } -Reject 'Raw Module.Dir differs; replacement/workspace layout unsupported.'
CheckListingControl 'raw Module.GoMod mismatch' { param($p) ChangeRaw $p '\\native\\go.mod"' '\\native\\other.mod"' } -Reject 'Raw Module.GoMod differs; replacement/workspace layout unsupported.'
CheckListingControl 'raw module replacement unsupported' { param($p) ChangeRaw $p '"Main": true,' '"Main": true, "Replace": null,' } -Reject 'Raw module replacement/error unsupported.'
CheckListingControl 'raw module non-main flag' { param($p) ChangeRaw $p '"Main": true' '"Main": false' } -Reject 'Raw Module.Main must be true.'
CheckListingControl 'raw module main flag wrong type' { param($p) ChangeRaw $p '"Main": true' '"Main": "true"' } -Reject 'Raw Module.Main must be true.'
CheckListingControl 'raw module main flag null' { param($p) ChangeRaw $p '"Main": true' '"Main": null' } -Reject 'Raw Module.Main must be true.'
CheckListingControl 'raw Module null' { param($p) ChangeRaw $p '"Module": {' '"Module": null, "IgnoredModule": {' } -Reject 'Raw Module must be object.'
CheckListingControl 'raw Dir nonstring' { param($p) ChangeRaw $p '"Dir": "F:\\Task57-Fictitious\\LevNote\\native\\app"' '"Dir": 1' } -Reject 'Invalid lexical Windows path: Dir'
CheckListingControl 'raw lexical traversal' { param($p) ChangeRaw $p '\\native\\app"' '\\native\\..\\app"' } -Reject 'Invalid lexical Windows path: Dir'
CheckListingControl 'raw wrong module prefix case' { param($p) ChangeRaw $p '"ImportPath": "levnote.local/native/app"' '"ImportPath": "LevNote.local/native/app"' } -Reject 'Raw package module prefix differs: LevNote.local/native/app'
CheckListingControl 'raw ForTest row' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "ForTest": "levnote.local/native/app",' } -Reject 'Raw package ForTest rows unsupported.'
CheckListingControl 'raw ForTest wrong type' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "ForTest": false,' } -Reject 'Raw package ForTest rows unsupported.'
CheckListingControl 'raw dependency-only flag' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "DepOnly": true,' } -Reject 'Raw package requires absent/false DepOnly.'
CheckListingControl 'raw incomplete flag null' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "Incomplete": null,' } -Reject 'Raw package requires absent/false Incomplete.'
CheckListingControl 'raw incomplete flag wrong type' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "Incomplete": "false",' } -Reject 'Raw package requires absent/false Incomplete.'
CheckListingControl 'raw Error presence' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "Error": {"Err":"synthetic"},' } -Reject 'Raw package Error/DepsErrors rows refused.'
CheckListingControl 'raw DepsErrors presence' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "DepsErrors": [],' } -Reject 'Raw package Error/DepsErrors rows refused.'
CheckListingControl 'duplicate exact raw identity' { param($p) $p.RawBytes=$script:utf8.GetBytes($script:firstObject+$script:firstObject+$script:secondObject) } -Reject 'Duplicate raw package identity: levnote.local/native/app'
CheckListingControl 'duplicate exact JSON property' { param($p) ChangeRaw $p '"ImportPath": "levnote.local/native/app",' '"ImportPath": "levnote.local/native/app", "ImportPath": "levnote.local/native/app",' } -Reject 'Duplicate/aliased JSON property: ImportPath'
CheckListingControl 'duplicate case alias JSON property' { param($p) ChangeRaw $p '"ImportPath": "levnote.local/native/app",' '"ImportPath": "levnote.local/native/app", "importPath": "levnote.local/native/app",' } -Reject 'Duplicate/aliased JSON property: importPath'
CheckListingControl 'single mis-cased package property' { param($p) ChangeRaw $p '"ImportPath"' '"importPath"' } -Reject 'Mis-cased package property: importPath'
CheckListingControl 'single mis-cased optional package flag' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "incomplete": false,' } -Reject 'Mis-cased package property: incomplete'
CheckListingControl 'single mis-cased module property' { param($p) ChangeRaw $p '"GoMod"' '"goMod"' } -Reject 'Mis-cased module property: goMod'
CheckListingControl 'duplicate nested ignored property' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "Ignored": {"v":1,"V":2},' } -Reject 'Duplicate/aliased JSON property: V'
CheckListingControl 'malformed ignored JSON still rejected' { param($p) ChangeRaw $p '"Name": "app",' '"Name": "app", "Ignored": [1,],' } -Reject 'Malformed JSON object in raw package listing.'
CheckListingControl 'missing exact required raw field' { param($p) ChangeRaw $p '"ImportPath"' '"IgnoredPath"' } -Reject 'Missing raw package field: ImportPath'

CheckListingControl 'internal array null' { param($p) SetRawArray $p 'TestGoFiles' 'null' } -Reject 'JSON array required: raw TestGoFiles'
CheckListingControl 'external array null' { param($p) SetRawArray $p 'XTestGoFiles' 'null' } -Reject 'JSON array required: raw XTestGoFiles'
CheckListingControl 'internal array wrong kind' { param($p) SetRawArray $p 'TestGoFiles' '"internal_test.go"' } -Reject 'JSON array required: raw TestGoFiles'
CheckListingControl 'external array wrong kind' { param($p) SetRawArray $p 'XTestGoFiles' '{}' } -Reject 'JSON array required: raw XTestGoFiles'
CheckListingControl 'non-string filename' { param($p) SetRawArray $p 'TestGoFiles' '[1]' } -Reject 'Raw test filename must be string: levnote.local/native/app / TestGoFiles'
CheckListingControl 'duplicate filename within field' { param($p) SetRawArray $p 'TestGoFiles' '["internal_test.go","internal_test.go"]' } -Reject 'Raw test filename invalid/aliased: levnote.local/native/app / TestGoFiles'
CheckListingControl 'filename alias across fields' { param($p) SetRawArray $p 'XTestGoFiles' '["INTERNAL_test.go"]' } -Reject 'Raw test filename invalid/aliased: levnote.local/native/app / XTestGoFiles'
CheckListingControl 'non-test basename' { param($p) SetRawArray $p 'TestGoFiles' '["internal.go"]' } -Reject 'Raw test filename invalid/aliased: levnote.local/native/app / TestGoFiles'
CheckListingControl 'filename not a basename' { param($p) SetRawArray $p 'TestGoFiles' '["nested/internal_test.go"]' } -Reject 'Raw test filename invalid/aliased: levnote.local/native/app / TestGoFiles'
CheckListingControl 'raw filename absent from inventory' { param($p) SetRawArray $p 'TestGoFiles' '["unlisted_test.go"]' } -Reject 'Raw test filename absent from source inventory: levnote.local/native/app / unlisted_test.go'

CheckListingControl 'bad UTF8' { param($p) $p.RawBytes=[byte[]]@(195,40) } -Reject 'Invalid UTF-8: rawPackageListing'
CheckListingControl 'UTF8 BOM' { param($p) $p.RawBytes=[byte[]](@(239,187,191)+$p.RawBytes) } -Reject 'UTF-8 BOM refused: rawPackageListing'
CheckListingControl 'top-level array' { param($p) $p.RawBytes=$script:utf8.GetBytes('['+$script:firstObject+']') } -Reject 'Raw listing requires top-level objects and JSON whitespace only.'
CheckListingControl 'trailing junk' { param($p) $p.RawBytes=$script:utf8.GetBytes($script:firstObject+$script:secondObject+'junk') } -Reject 'Raw listing requires top-level objects and JSON whitespace only.'
CheckListingControl 'trailing scalar' { param($p) $p.RawBytes=$script:utf8.GetBytes($script:firstObject+$script:secondObject+'null') } -Reject 'Raw listing requires top-level objects and JSON whitespace only.'
CheckListingControl 'non-JSON whitespace' { param($p) $p.RawBytes=$script:utf8.GetBytes($script:firstObject+[char]160+$script:secondObject) } -Reject 'Raw listing requires top-level objects and JSON whitespace only.'
CheckListingControl 'only whitespace' { param($p) $p.RawBytes=$script:utf8.GetBytes(" `t`r`n") } -Reject 'Raw package listing contains no objects.'
CheckListingControl 'mismatched delimiters' { param($p) $p.RawBytes=$script:utf8.GetBytes('{"Ignored":[}') } -Reject 'Mismatched delimiter in raw package listing.'
CheckListingControl 'unterminated escaped string' { param($p) $p.RawBytes=$script:utf8.GetBytes('{"Ignored":"escaped\"}') } -Reject 'Truncated raw package listing object.'

foreach ($descriptorRole in @('sourceInventory','toolchainIdentity','expectedManifest','packageListing','runIdentity')) {
    # Invoked synchronously before the loop advances; no deferred closure scope.
    $mutation={ param($p) $p.InputBytes[$descriptorRole]=[byte[]]@(195,40) }
    CheckListingControl ("descriptor bad UTF8 $descriptorRole") $mutation -Reject "Invalid UTF-8: $descriptorRole"
    $mutation={ param($p) $p.InputBytes[$descriptorRole]=[byte[]]@(239,187,191,123,125) }
    CheckListingControl ("descriptor BOM $descriptorRole") $mutation -Reject "UTF-8 BOM refused: $descriptorRole"
}

# Each limit uses the actual production constant or documented boundary. No cap
# is lowered and no private helper is called as a substitute for ReduceTestJson.
foreach ($role in @('rawJsonl','rawPackageListing','packageListing','expectedManifest','sourceInventory','toolchainIdentity','runIdentity')) {
    $limit=$script:ListingRevisionByteLimits[$role]
    $mutation={ param($p) $p.Lengths[$role]=$limit+1 }
    CheckListingControl ("byte cap $role") $mutation -Reject "Input byte limit/empty: $role (maximum $limit)."
}
CheckListingControl 'zero raw listing bytes' { param($p) $p.RawBytes=[byte[]]::new(0) } -Reject 'Input byte limit/empty: rawPackageListing (maximum 33554432).'
CheckListingControl 'object byte cap' { param($p) $p.RawBytes=$script:utf8.GetBytes('{"Ignored":"'+('x'*2MB)+'"}') } -Reject 'Raw listing object exceeds 2MiB.'
CheckListingControl 'object count cap' { param($p) $p.RawBytes=$script:utf8.GetBytes($script:firstObject*4097) } -Reject 'Raw listing object count exceeds 4096.'
CheckListingControl 'depth 65' { param($p) ChangeRaw $p '"Name": "app",' ('"Name": "app", "IgnoredDepth": '+('['*64)+'0'+(']'*64)+',') } -Reject 'Raw listing nesting exceeds depth 64.'
foreach ($field in @('TestGoFiles','XTestGoFiles')) {
    $mutation={ param($p) SetRawArray $p $field ('['+([string]::Join(',',([string[]]@('"limit_test.go"')*2049)))+']') }
    CheckListingControl ("raw per-field filename cap $field") $mutation -Reject "Entry count limit: raw $field (maximum 2048)."
    $mutation={ param($p) $p.Listed[0][$field]=[object[]]@('limit_test.go')*2049 }
    CheckListingControl ("normalized per-field filename cap $field") $mutation -Reject "Entry count limit: normalized $field (maximum 2048)."
}
CheckListingControl 'source inventory entry cap' { param($p) $p.Source.files=[object[]]@(@{path='native/go.mod'; bytes=1; sha256=('2'*64)})*65537 } -Reject 'Entry count limit: source inventory files (maximum 65536).'
CheckListingControl 'expected package entry cap' { param($p) $p.Expected=[object[]]@($p.Expected[0])*4097 } -Reject 'Entry count limit: expected/listing packages (maximum 4096).'
CheckListingControl 'normalized package entry cap' { param($p) $p.Listed=[object[]]@($p.Listed[0])*4097 } -Reject 'Entry count limit: expected/listing packages (maximum 4096).'
CheckListingControl 'changed roots fail before preflight' -Roots @('./cmd/...','./app/','./core/...','./formats/...','./modules/...','./ui/...') -Reject 'Changed/reordered product root at index 0.'
foreach ($name in $fixturePins.Keys) { SameFingerprint $fixturePins[$name] (Fingerprint $fixturePins[$name].absolutePath) }
Write-Output "Synthetic controls: $script:accepted accepted, $script:rejected rejected. Inputs retained at $script:fixtureRoot"
