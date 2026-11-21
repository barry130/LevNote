#requires -Version 7.0
# UNEXECUTED synthetic JSONL projections for the R1 lifecycle contract.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function NewFixtureEvent([string]$Action, [string]$Test = '') {
    $event = [ordered]@{Action=$Action; Package=$script:package}
    if ($Test) { $event.Test = $Test }
    return $event
}

function ReadRawFixtureEvents([string]$Path) {
    $events = [Collections.Generic.List[object]]::new()
    foreach ($line in [IO.File]::ReadLines((PlainFile $Path),[Text.UTF8Encoding]::new($false,$true))) {
        $events.Add((StrictJson $line))
    }
    return ,([object[]]$events.ToArray())
}

function NewSlashFixture([object[]]$Events, [string]$RawPath='') {
    $directory = FreshDirectory (Join-Path $script:fixtureRoot ([Guid]::NewGuid().ToString('N')))
    $target = @{GOOS='windows'; GOARCH='amd64'; GOAMD64='v1'; goVersion='go1.27.3'}
    $inputs = @{}
    $source = @{
        schema=1; repositoryRoot=$directory; files=@(
            @{path='native/app/fixture_test.go'; bytes=1; sha256=('1'*64)},
            @{path='native/go.mod'; bytes=1; sha256=('2'*64)}
        )
    }
    $tool = @{
        schema=1; target=$target; archiveSHA256=$script:GoArchiveSHA256; archiveBytes=$script:GoArchiveBytes
        executable=@{sha256=('3'*64); bytes=1}; inventory=@{sha256=('4'*64); fileCount=1}
    }
    foreach ($name in @('sourceInventory','toolchainIdentity')) {
        $path = Join-Path $directory ($name+'.json')
        $document = if ($name -ceq 'sourceInventory') { $source } else { $tool }
        NewJson $path $document
        $inputs[$name] = @{path=$path; sha256=(Fingerprint $path).sha256}
    }
    # Synthetic descriptors and receipt satisfy shape/pins. They attest to no execution.
    $listed = @(@{Package=$script:package; RelativeDirectory='app'; TestGoFiles=@('fixture_test.go'); XTestGoFiles=@()})
    $rawListing = Join-Path $directory 'raw-listing.json'
    NewJson $rawListing @{ImportPath=$script:package; TestGoFiles=@('fixture_test.go')}
    $inputs.rawPackageListing = @{path=$rawListing; sha256=(Fingerprint $rawListing).sha256}
    $manifestFields = @{
        schema=1; origin='independent-package-list-before-tests'; roots=$script:ProductRoots; target=$target
        modulePath='levnote.local/native'; sourceInventorySHA256=$inputs.sourceInventory.sha256
        toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256
    }
    $listing = $manifestFields.Clone()
    $listing.packages = $listed
    $listing.generation = @{
        arguments=(@('list','-json')+$script:ProductRoots)
        rawListingSHA256=$inputs.rawPackageListing.sha256; completedBeforeTest=$true
    }
    $listingPath = Join-Path $directory 'listing.json'
    NewJson $listingPath $listing
    $inputs.packageListing = @{path=$listingPath; sha256=(Fingerprint $listingPath).sha256}
    $expected = $manifestFields.Clone()
    $expected.packageListingSHA256 = $inputs.packageListing.sha256
    $expected.packages = @(@{Package=$script:package; RelativeDirectory='app'; HasTests=$true})
    $expectedPath = Join-Path $directory 'expected.json'
    NewJson $expectedPath $expected
    $inputs.expectedManifest = @{path=$expectedPath; sha256=(Fingerprint $expectedPath).sha256}
    $raw = Join-Path $directory 'capture.jsonl'
    if ($RawPath) {
        # Preserve the checked-in raw fixture bytes; never serialize a wrapper/catalog.
        $bytes = [IO.File]::ReadAllBytes((PlainFile $RawPath))
    } else {
        $lines = [string[]]@($Events | ForEach-Object { ConvertTo-Json -InputObject $_ -Depth 8 -Compress })
        $bytes = [Text.UTF8Encoding]::new($false,$true).GetBytes(([string]::Join("`n",$lines)+"`n"))
    }
    $stream = [IO.File]::Open($raw,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try { $stream.Write($bytes,0,$bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
    $inputs.rawJsonl = @{path=$raw; sha256=(Fingerprint $raw).sha256}
    $receipt = @{
        schema=1; roots=$script:ProductRoots; target=$target
        arguments=(@('test','-json','-count=1','-timeout=15m')+$script:ProductRoots)
        sourceInventorySHA256=$inputs.sourceInventory.sha256; toolchainIdentitySHA256=$inputs.toolchainIdentity.sha256
        expectedManifestSHA256=$inputs.expectedManifest.sha256; rawJsonlSHA256=$inputs.rawJsonl.sha256
        exitCode=0; completed=$true; timedOut=$false; cleanupForced=$false; jobEmptyObserved=$true
    }
    $runPath = Join-Path $directory 'run.json'
    NewJson $runPath $receipt
    $inputs.runIdentity = @{path=$runPath; sha256=(Fingerprint $runPath).sha256}
    return [pscustomobject]@{inputs=$inputs; summary=(Join-Path $directory 'summary.json')}
}

function ExpectAccepted([string]$Name, [object[]]$Body, [string[]]$TerminalOrder, [string[]]$ParallelTests=@(), [string]$SkippedTest='', [string]$RawFile='') {
    $rawPath = ''; $rawBefore = $null
    if ($RawFile) {
        $rawPath = PlainFile (RelativeFile (Join-Path $PSScriptRoot 'fixtures') $RawFile)
        $rawBefore = Fingerprint $rawPath
        $events = ReadRawFixtureEvents $rawPath
    } else {
        $events = @((NewFixtureEvent start)) + $Body + @((NewFixtureEvent pass))
    }
    $fixture = NewSlashFixture $events $rawPath
    $before = Fingerprint $fixture.inputs.rawJsonl.path
    $result = ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary
    $saved = Json $fixture.summary
    $skipCount = [int](-not [string]::IsNullOrEmpty($SkippedTest))
    if ($saved.status -cne 'VERIFIED_CAPTURE_ONLY' -or $saved.windowsAcceptance -cne 'NOT_WINDOWS_ACCEPTED' -or $saved.ciEnabled -or
        $result.packagePassCount -ne 1 -or $result.eventLineCount -ne $events.Count -or
        $result.testPassCount -ne ($TerminalOrder.Count - $skipCount) -or $result.testSkipCount -ne $skipCount -or
        $result.tests.Count -ne $TerminalOrder.Count) {
        throw "Unexpected summary for $Name"
    }
    $ordered = @($result.tests | Sort-Object -Property { $_.terminalLine })
    for ($i=0; $i -lt $ordered.Count; $i++) {
        $row = $ordered[$i]
        $parallelCount = [int]($row.Test -cin $ParallelTests)
        $terminal = if ($row.Test -ceq $SkippedTest) { 'skip' } else { 'pass' }
        if ($row.Package -cne $script:package -or $row.Test -cne $TerminalOrder[$i] -or $row.terminal -cne $terminal -or
            $events[($row.runLine - 1)].Action -cne 'run' -or $events[($row.runLine - 1)].Test -cne $row.Test -or
            $events[($row.terminalLine - 1)].Action -cne $terminal -or $events[($row.terminalLine - 1)].Test -cne $row.Test -or
            $row.pauseCount -ne $parallelCount -or $row.contCount -ne $parallelCount) {
            throw "Unexpected identity or lifecycle for $Name at index $i"
        }
    }
    SameFingerprint $before (Fingerprint $fixture.inputs.rawJsonl.path)
    if ($null -ne $rawBefore) { SameFingerprint $rawBefore (Fingerprint $rawPath) }
    Write-Output "PASS $Name"
}

function ExpectRejected([string]$Name, [object[]]$Events, [string]$Message, [string]$RawFile='') {
    $rawPath = ''; $rawBefore = $null
    if ($RawFile) {
        $rawPath = PlainFile (RelativeFile (Join-Path $PSScriptRoot 'fixtures') $RawFile)
        $rawBefore = Fingerprint $rawPath
        $Events = ReadRawFixtureEvents $rawPath
    }
    $fixture = NewSlashFixture $Events $rawPath
    $before = Fingerprint $fixture.inputs.rawJsonl.path
    $actual = $null
    try { $null = ReduceTestJson $fixture.inputs $script:ProductRoots $fixture.summary } catch { $actual = $_.Exception.Message }
    if ($actual -cne $Message -or (Test-Path -LiteralPath $fixture.summary)) {
        throw "Unexpected rejection for $Name. Expected [$Message], got [$actual]."
    }
    SameFingerprint $before (Fingerprint $fixture.inputs.rawJsonl.path)
    if ($null -ne $rawBefore) { SameFingerprint $rawBefore (Fingerprint $rawPath) }
    Write-Output "PASS $Name"
}

if (-not $IsWindows -or $PSVersionTable.PSVersion.Major -ne 7 -or
    [Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString() -cne 'X64' -or
    [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -cne 'X64') {
    throw 'Fixture setup requires native Windows x64 PowerShell 7.'
}
$revision = Split-Path $PSScriptRoot -Parent
. (Join-Path $revision 'ci-lib.ps1')
. (Join-Path $revision 'test-json-reducer.ps1')
$script:package = 'levnote.local/native/app'
$temp = PlainDirectory ([IO.Path]::GetTempPath())
$script:fixtureRoot = FreshDirectory (Join-Path $temp ('levnote-slash-fixtures-'+[Guid]::NewGuid().ToString('N')))
Write-Output "Synthetic fixture directory $script:fixtureRoot"

$parent = 'TestParent'
$group = 'TestParent/Group'
$label = 'TestParent/Group/Label'
$dedup = 'TestParent/Group/Label#01'
$start = NewFixtureEvent start
$packagePass = NewFixtureEvent pass
$parentRun = NewFixtureEvent run $parent
$parentPass = NewFixtureEvent pass $parent
$groupRun = NewFixtureEvent run $group
$groupPass = NewFixtureEvent pass $group
$labelRun = NewFixtureEvent run $label
$labelPass = NewFixtureEvent pass $label

# Nine independent raw files: lifecycle-only projections, never real captures.
ExpectAccepted 'raw slash-free examples' @() @('Example','ExampleWidget') -RawFile '01-top-level-example.jsonl'
ExpectAccepted 'raw prefix siblings Group terminal first' @() @($group,$label,$parent) -RawFile '02-prefix-siblings-group-first.jsonl'
ExpectAccepted 'raw prefix siblings Label terminal first' @() @($label,$group,$parent) @($group) -RawFile '03-prefix-siblings-label-first.jsonl'
ExpectRejected 'raw slash after true top terminal' @() 'Slash test run lacks a live observed top-level identity.' -RawFile '04-after-top-terminal.reject.jsonl'
ExpectAccepted 'raw genuine nesting control' @() @($label,$group,$parent) -RawFile '05-genuinely-nested.jsonl'
ExpectAccepted 'raw slash label deduplication' @() @($label,$dedup,$parent) -RawFile '06-slash-dedup.jsonl'
ExpectAccepted 'raw parallel prefix siblings with deduplication' @() @($group,$dedup,$label,$parent) @($group,$label,$dedup) -RawFile '07-parallel-dedup.jsonl'
ExpectRejected 'raw one terminal leaves another slash identity unfinished' @() 'Top-level test completes before observed slash test terminal.' -RawFile '08-top-blocked-unfinished.reject.jsonl'
# match.go unique() gives a repeated label #01. With this parent still live,
# another identical full run is a corrupted replay, not a legal repeated label.
ExpectRejected 'raw duplicate exact key while first is paused' @() 'Duplicate test run.' -RawFile '09-duplicate-exact-key.reject.jsonl'

# Independent t.Run("Group") and t.Run("Group/Label") calls by TestParent.
ExpectAccepted 'sequential sibling Group then Group/Label' @(
    $parentRun, $groupRun, $groupPass, $labelRun, $labelPass, $parentPass
) @($group,$label,$parent)
ExpectAccepted 'sequential sibling Group/Label then Group' @(
    $parentRun, $labelRun, $labelPass, $groupRun, $groupPass, $parentPass
) @($label,$group,$parent)

# One sibling calls Parallel and resumes after the synchronous sibling finishes.
ExpectAccepted 'Group terminal while the slash-label sibling is paused' @(
    $parentRun, $labelRun, (NewFixtureEvent pause $label), $groupRun, $groupPass, (NewFixtureEvent cont $label), $labelPass, $parentPass
) @($group,$label,$parent) @($label)
ExpectAccepted 'Group/Label terminal while its prefix-named sibling is paused' @(
    $parentRun, $groupRun, (NewFixtureEvent pause $group), $labelRun, $labelPass, (NewFixtureEvent cont $group), $groupPass, $parentPass
) @($label,$group,$parent) @($group)

# Genuine nesting arises when Group itself calls t.Run("Label").
ExpectAccepted 'nested control Group runs Label' @(
    $parentRun, $groupRun, $labelRun, $labelPass, $groupPass, $parentPass
) @($label,$group,$parent)
ExpectAccepted 'slash-bearing label without any observed Group' @(
    $parentRun, $labelRun, $labelPass, $parentPass
) @($label,$parent)
$skipReason = NewFixtureEvent output $label
$skipReason['Output'] = "    fixture_test.go:7: fixture reason`n"
ExpectAccepted 'slash skip with reason releases the top-level identity' @(
    $parentRun, $labelRun, $skipReason, (NewFixtureEvent skip $label), $parentPass
) @($label,$parent) -SkippedTest $label

# Two t.Run("Group/Label") calls get distinct names even if both call Parallel.
$parallelNames = @(
    $parentRun, $labelRun, (NewFixtureEvent pause $label), (NewFixtureEvent run $dedup), (NewFixtureEvent pause $dedup),
    (NewFixtureEvent cont $label), (NewFixtureEvent cont $dedup), (NewFixtureEvent pass $dedup), $labelPass, $parentPass
)
ExpectAccepted 'repeated slash label has unique parallel full identities' $parallelNames @($dedup,$label,$parent) @($label,$dedup)
# Preserve legitimate slash-free actions such as testing's Example entry points.
ExpectAccepted 'slash-free Example remains supported' @((NewFixtureEvent run 'Example'),(NewFixtureEvent pass 'Example')) @('Example')

ExpectRejected 'orphan slash run after actual top-level terminal' @(
    $start, $parentRun, $parentPass, $labelRun, $labelPass, $packagePass
) 'Slash test run lacks a live observed top-level identity.'
ExpectRejected 'slash run without any top-level run' @(
    $start, $labelRun, $labelPass, $packagePass
) 'Slash test run lacks a live observed top-level identity.'
ExpectRejected 'top-level terminal while a slash identity is still paused' @(
    $start, $parentRun, $labelRun, (NewFixtureEvent pause $label), $parentPass, (NewFixtureEvent cont $label), $labelPass, $packagePass
) 'Top-level test completes before observed slash test terminal.'
ExpectRejected 'one slash terminal does not release a second live identity' @(
    $start, $parentRun, $labelRun, (NewFixtureEvent pause $label), (NewFixtureEvent run $dedup), (NewFixtureEvent pass $dedup),
    $parentPass, (NewFixtureEvent cont $label), $labelPass, $packagePass
) 'Top-level test completes before observed slash test terminal.'
ExpectRejected 'duplicate exact slash run while first is paused' @(
    $start, $parentRun, $labelRun, (NewFixtureEvent pause $label), $labelRun, (NewFixtureEvent cont $label), $labelPass, $parentPass, $packagePass
) 'Duplicate test run.'
ExpectRejected 'duplicate exact slash terminal while top-level is live' @(
    $start, $parentRun, $labelRun, $labelPass, $labelPass, $parentPass, $packagePass
) 'Duplicate terminal or event after test completion.'
ExpectRejected 'package terminal while top-level is still active' @(
    $start, $parentRun, $labelRun, $labelPass, $packagePass
) 'Package completes before test terminal.'

# A cached result replays one package transcript. Concatenating two transcripts is invalid.
$whole = @($start) + $parallelNames + @($packagePass)
ExpectRejected 'two complete copies do not form one capture' ($whole + $whole) 'Duplicate/invalid package start.'
