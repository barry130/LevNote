#requires -Version 7.0
# Offline, fail-closed Go 1.27.3 event reducer. No Go invocation or identity discovery.
Set-StrictMode -Version Latest
function RequireArray([object]$Value,[string]$Label) {
    if ($Value -isnot [array]) { throw "JSON array required: $Label" }
}
function RequireInteger([object]$Value,[long]$Minimum,[string]$Label) {
    if (($Value -isnot [long] -and $Value -isnot [int]) -or $Value -lt $Minimum) { throw "Invalid integer: $Label" }
}
function RequireString([object]$Value,[string]$Label) {
    if ($Value -isnot [string] -or [string]::IsNullOrEmpty($Value)) { throw "Nonempty JSON string required: $Label" }
}
function Schema1([object]$Value) {
    RequireInteger $Value 1 'schema'
    if ($Value -ne 1) { throw 'Unsupported schema.' }
}
function TestTarget([Collections.IDictionary]$Target) {
    Keys $Target @('GOOS','GOARCH','GOAMD64','goVersion')
    foreach ($name in @('GOOS','GOARCH','GOAMD64','goVersion')) { RequireString $Target[$name] $name }
    if ($Target.GOOS -cne 'windows' -or $Target.GOARCH -cne 'amd64' -or $Target.GOAMD64 -cne 'v1' -or $Target.goVersion -cne 'go1.27.3') {
        throw 'Expected Windows/amd64/v1 and go1.27.3 identity.'
    }
}
function TestArguments([object]$Arguments,[string[]]$Prefix) {
    RequireArray $Arguments 'arguments'
    $expected=@($Prefix)+$script:ProductRoots
    if ($Arguments.Count -ne $expected.Count) { throw 'Command argument count differs.' }
    for ($i=0; $i -lt $expected.Count; $i++) {
        if ($Arguments[$i] -isnot [string] -or $Arguments[$i] -cne $expected[$i]) { throw "Command argument differs at $i." }
    }
}
function TestPackageIdentity([string]$Package,[string]$RelativeDirectory) {
    AssertRelativeGrammar $RelativeDirectory
    $inside=$RelativeDirectory -ceq 'app'
    foreach ($root in @('cmd','core','formats','modules','ui')) {
        if ($RelativeDirectory -ceq $root -or $RelativeDirectory.StartsWith($root+'/',[StringComparison]::Ordinal)) { $inside=$true }
    }
    if (-not $inside -or $Package -cne ('levnote.local/native/'+$RelativeDirectory)) { throw "Package outside exact six roots/module: $Package" }
}
function ReadExpectedTestContext([Collections.IDictionary]$Inputs,[string[]]$Roots) {
    AssertProductRoots $Roots # BEFORE any file read, evidence directory or output.
    $names=@('rawJsonl','expectedManifest','sourceInventory','toolchainIdentity','packageListing','rawPackageListing','runIdentity')
    Keys $Inputs $names
    $pins=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    $buffers=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    $paths=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($name in $names) {
        Keys $Inputs[$name] @('path','sha256')
        RequireString $Inputs[$name].path 'input path'; SHA256Value $Inputs[$name].sha256
        $bounded=ReadListingRevisionInput $Inputs[$name].path $Inputs[$name].sha256 $name ($name -cne 'rawJsonl')
        $fact=$bounded.fact
        if (-not $paths.Add($fact.absolutePath)) { throw 'Identity/raw input paths must be distinct.' }
        $pins.Add($name,$fact); $buffers.Add($name,$bounded.bytes)
    }
    $source=ReadListingRevisionJson $buffers['sourceInventory'] 'sourceInventory'
    Keys $source @('schema','repositoryRoot','files') @('scope')
    Schema1 $source.schema; RequireString $source.repositoryRoot 'source repositoryRoot'
    $null=LocalPath $source.repositoryRoot
    if ($source.Contains('scope')) { RequireString $source.scope 'source inventory scope' }
    RequireArray $source.files 'source inventory files'
    AssertListingRevisionCount $source.files.Count $script:ListingRevisionSourceCount 'source inventory files'
    if (-not $source.files.Count) { throw 'Empty source inventory.' }
    $sourcePaths=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    $sourceAliases=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $previous=$null
    foreach ($file in $source.files) {
        Keys $file @('path','bytes','sha256') @('absolutePath','lastWriteUtcTicks')
        RequireString $file.path 'source file path'; AssertRelativeGrammar $file.path
        RequireInteger $file.bytes 0 'source file bytes'; SHA256Value $file.sha256
        if ($file.Contains('absolutePath')) {
            RequireString $file.absolutePath 'source absolutePath'
            $absolute=LocalPath $file.absolutePath
            $joined=LocalPath (Join-Path $source.repositoryRoot $file.path)
            if (-not $absolute.Equals($joined,[StringComparison]::OrdinalIgnoreCase) -or -not (Within $absolute $source.repositoryRoot)) { throw 'Source absolute/relative path identity differs.' }
        }
        if ($file.Contains('lastWriteUtcTicks')) { RequireInteger $file.lastWriteUtcTicks 0 'source lastWriteUtcTicks' }
        if (-not $sourcePaths.Add($file.path) -or -not $sourceAliases.Add($file.path) -or ($null -ne $previous -and [StringComparer]::Ordinal.Compare($previous,$file.path) -ge 0)) {
            throw 'Source inventory must have unique, ordinal-sorted paths.'
        }
        $previous=$file.path
    }
    if (-not $sourcePaths.Contains('native/go.mod')) { throw 'Source inventory omits module identity.' }
    $tool=ReadListingRevisionJson $buffers['toolchainIdentity'] 'toolchainIdentity'
    Keys $tool @('schema','target','archiveSHA256','archiveBytes','executable','inventory')
    Schema1 $tool.schema; TestTarget $tool.target; SHA256Value $tool.archiveSHA256
    RequireInteger $tool.archiveBytes 1 'Go archive bytes'
    if ($tool.archiveSHA256 -cne $script:GoArchiveSHA256 -or $tool.archiveBytes -ne $script:GoArchiveBytes) { throw 'Pinned Go distribution identity differs.' }
    Keys $tool.executable @('sha256','bytes'); SHA256Value $tool.executable.sha256; RequireInteger $tool.executable.bytes 1 'Go executable bytes'
    Keys $tool.inventory @('sha256','fileCount'); SHA256Value $tool.inventory.sha256; RequireInteger $tool.inventory.fileCount 1 'Go inventory file count'
    $expected=ReadListingRevisionJson $buffers['expectedManifest'] 'expectedManifest'
    Keys $expected @('schema','origin','roots','target','modulePath','sourceInventorySHA256','toolchainIdentitySHA256','packageListingSHA256','packages')
    $listing=ReadListingRevisionJson $buffers['packageListing'] 'packageListing'
    Keys $listing @('schema','origin','roots','target','modulePath','sourceInventorySHA256','toolchainIdentitySHA256','generation','packages')
    foreach ($document in @($expected,$listing)) {
        Schema1 $document.schema; RequireString $document.origin 'origin'; RequireString $document.modulePath 'modulePath'
        SHA256Value $document.sourceInventorySHA256; SHA256Value $document.toolchainIdentitySHA256
        if ($document.origin -cne 'independent-package-list-before-tests' -or $document.modulePath -cne 'levnote.local/native') { throw 'Independent package manifest origin/schema/module differs.' }
        RequireArray $document.roots 'manifest roots'; AssertProductRoots $document.roots; TestTarget $document.target
        if ($document.sourceInventorySHA256 -cne $pins['sourceInventory'].sha256 -or $document.toolchainIdentitySHA256 -cne $pins['toolchainIdentity'].sha256) {
            throw 'Expected package identities are not bound to this source/toolchain inventory.'
        }
        RequireArray $document.packages 'expected/listing packages'
        AssertListingRevisionCount $document.packages.Count $script:ListingRevisionPackageCount 'expected/listing packages'
    }
    SHA256Value $expected.packageListingSHA256
    if ($expected.packageListingSHA256 -cne $pins['packageListing'].sha256) { throw 'Independent package-listing pin differs.' }
    Keys $listing.generation @('arguments','rawListingSHA256','completedBeforeTest')
    SHA256Value $listing.generation.rawListingSHA256
    TestArguments $listing.generation.arguments @('list','-json')
    if ($listing.generation.completedBeforeTest -isnot [bool] -or -not $listing.generation.completedBeforeTest -or
        $listing.generation.rawListingSHA256 -cne $pins['rawPackageListing'].sha256) { throw 'Independent pre-test listing provenance/pin differs.' }
    if (-not $expected.packages.Count -or $expected.packages.Count -ne $listing.packages.Count) { throw 'Empty/mismatched expected and independent package sets.' }
    $aliases=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $previous=$null
    for ($i=0; $i -lt $expected.packages.Count; $i++) {
        $package=$expected.packages[$i]; $listed=$listing.packages[$i]
        Keys $package @('Package','RelativeDirectory','HasTests')
        Keys $listed @('Package','RelativeDirectory','TestGoFiles','XTestGoFiles')
        foreach ($value in @($package,$listed)) { RequireString $value.Package 'Package'; RequireString $value.RelativeDirectory 'RelativeDirectory' }
        TestPackageIdentity $package.Package $package.RelativeDirectory
        if ($package.HasTests -isnot [bool] -or -not $aliases.Add($package.Package) -or
            ($null -ne $previous -and [StringComparer]::Ordinal.Compare($previous,$package.Package) -ge 0)) { throw 'Expected package identities must be unique and ordinal-sorted with boolean HasTests.' }
        $previous=$package.Package
        if ($listed.Package -cne $package.Package -or $listed.RelativeDirectory -cne $package.RelativeDirectory) { throw 'Independent package listing identities differ.' }
        RequireArray $listed.TestGoFiles 'TestGoFiles'; RequireArray $listed.XTestGoFiles 'XTestGoFiles'
        AssertListingRevisionCount $listed.TestGoFiles.Count $script:ListingRevisionFileCount 'normalized TestGoFiles'
        AssertListingRevisionCount $listed.XTestGoFiles.Count $script:ListingRevisionFileCount 'normalized XTestGoFiles'
        $testFiles=@($listed.TestGoFiles)+@($listed.XTestGoFiles)
        if ($package.HasTests -ne ($testFiles.Count -gt 0)) { throw 'HasTests differs from independent TestGoFiles/XTestGoFiles.' }
        $seenFiles=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
        foreach ($name in $testFiles) {
            if ($name -isnot [string]) { throw 'Non-string listed test filename.' }
            AssertRelativeGrammar $name
            if ($name.Contains('/') -or -not $name.EndsWith('_test.go',[StringComparison]::Ordinal) -or -not $seenFiles.Add($name) -or
                -not $sourcePaths.Contains('native/'+$package.RelativeDirectory+'/'+$name)) { throw 'Listed test file omitted from source inventory/aliased/invalid.' }
        }
    }
    AssertRawPackageListing $buffers['rawPackageListing'] $source.repositoryRoot $sourcePaths $listing.packages $expected.packages
    $run=ReadListingRevisionJson $buffers['runIdentity'] 'runIdentity'
    Keys $run @('schema','roots','target','arguments','sourceInventorySHA256','toolchainIdentitySHA256','expectedManifestSHA256','rawJsonlSHA256','exitCode','completed','timedOut','cleanupForced','jobEmptyObserved')
    Schema1 $run.schema
    foreach ($name in @('sourceInventorySHA256','toolchainIdentitySHA256','expectedManifestSHA256','rawJsonlSHA256')) { SHA256Value $run[$name] }
    RequireArray $run.roots 'run roots'; AssertProductRoots $run.roots; TestTarget $run.target
    TestArguments $run.arguments @('test','-json','-count=1','-timeout=15m')
    if ($run.sourceInventorySHA256 -cne $pins['sourceInventory'].sha256 -or $run.toolchainIdentitySHA256 -cne $pins['toolchainIdentity'].sha256 -or
        $run.expectedManifestSHA256 -cne $pins['expectedManifest'].sha256 -or $run.rawJsonlSHA256 -cne $pins['rawJsonl'].sha256) { throw 'Captured command source/toolchain/manifest/raw identity differs.' }
    RequireInteger $run.exitCode 0 'command exitCode'
    foreach ($name in @('completed','jobEmptyObserved')) { if ($run[$name] -isnot [bool] -or -not $run[$name]) { throw "Command receipt lacks $name." } }
    foreach ($name in @('timedOut','cleanupForced')) { if ($run[$name] -isnot [bool] -or $run[$name]) { throw "Command receipt has invalid $name." } }
    if ($run.exitCode -ne 0) { throw 'Nonzero command exit cannot be upgraded by events.' }
    $packages=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($package in $expected.packages) { $packages.Add($package.Package,$package) }
    return [pscustomobject]@{expected=$expected; packages=$packages; pins=$pins; run=$run; sourceRepositoryRoot=$source.repositoryRoot}
}
function ReduceTestJson([Collections.IDictionary]$Inputs,[string[]]$Roots,[string]$SummaryPath='') {
    AssertProductRoots $Roots # even a bad root with nonexistent files must fail here first.
    $context=ReadExpectedTestContext $Inputs $Roots
    if ($SummaryPath) {
        $destination=LocalPath $SummaryPath; $null=PlainDirectory ([IO.Path]::GetDirectoryName($destination))
        foreach ($pin in $context.pins.Values) { if ($destination.Equals($pin.absolutePath,[StringComparison]::OrdinalIgnoreCase)) { throw 'Summary aliases an input.' } }
        if (Test-Path -LiteralPath $destination) { throw 'Summary must be create-only.' }
    }
    $raw=$context.pins['rawJsonl']
    if ($raw.bytes -lt 1 -or $raw.bytes -gt 256MB) { throw 'Raw JSONL is empty or exceeds the explicit 256MiB review cap.' }
    $states=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($package in $context.expected.packages) {
        $states.Add($package.Package,[ordered]@{Package=$package.Package; HasTests=$package.HasTests; startLine=0; terminalLine=0; terminal=''; output=[Collections.Generic.List[object]]::new(); tests=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)})
    }
    $lineNumber=0
    foreach ($line in [IO.File]::ReadLines($raw.absolutePath,[Text.UTF8Encoding]::new($false,$true))) {
        $lineNumber++
        if ([string]::IsNullOrWhiteSpace($line) -or $line.Length -gt 1MB) { throw "Blank/oversized JSONL line $lineNumber." }
        try { $event=StrictJson $line } catch { throw "Malformed JSONL line ${lineNumber}: $($_.Exception.Message)" }
        Keys $event @('Action','Package') @('Time','Test','Elapsed','Output','OutputType','FailedBuild')
        if ($event.Action -isnot [string] -or $event.Package -isnot [string] -or -not $event.Package -or -not $states.ContainsKey($event.Package)) { throw "Unknown/empty package or invalid action at line $lineNumber." }
        if ($event.Action -cnotin @('start','run','pause','cont','output','pass','skip','fail')) { throw "Unknown/unsupported action at line ${lineNumber}: $($event.Action)" }
        $test=$null
        if ($event.Contains('Test')) {
            if ($event.Test -isnot [string] -or [string]::IsNullOrWhiteSpace($event.Test) -or $event.Test -match '[\x00-\x1F\x7F]') { throw 'Empty/invalid full Test identity.' }
            $test=$event.Test
        }
        if ($event.Contains('Time') -and ($event.Time -isnot [string] -or -not $event.Time)) { throw 'Invalid Time field.' }
        if ($event.Contains('Elapsed')) {
            if ($event.Elapsed -isnot [ValueType] -or $event.Elapsed -is [bool] -or $event.Elapsed -lt 0 -or
                [double]::IsNaN([double]$event.Elapsed) -or [double]::IsInfinity([double]$event.Elapsed)) { throw 'Invalid Elapsed field.' }
        }
        if ($event.Action -cne 'output' -and ($event.Contains('Output') -or $event.Contains('OutputType'))) { throw 'Output fields on non-output event.' }
        if ($event.Contains('FailedBuild') -and $event.Action -cne 'fail') { throw 'FailedBuild on non-fail event.' }
        $state=$states[$event.Package]
        if ($event.Action -ceq 'fail') { throw "Fail event at line ${lineNumber}: $($event.Package) / $test" }
        if ($event.Action -ceq 'start') {
            if ($null -ne $test -or $state.startLine -or $state.terminalLine) { throw 'Duplicate/invalid package start.' }
            $state.startLine=$lineNumber; continue
        }
        if (-not $state.startLine -or $state.terminalLine) { throw 'Orphan event or event after package terminal.' }
        if ($null -ne $test) {
            if (-not $state.HasTests) { throw 'No-test package emitted a Test identity.' }
            if ($event.Action -ceq 'run') {
                if ($state.tests.ContainsKey($test)) { throw 'Duplicate test run.' }
                # Slash-free Test fields remain independent package-scoped records,
                # including Example names. Do not infer a Go function-name grammar.
                $topLevelName=$null
                $slash=$test.IndexOf([char]'/')
                if ($slash -ge 0) {
                    # Only the exact first component links to an observed top level.
                    # A later slash can belong to one label: no middle-parent edges.
                    $topLevelName=$test.Substring(0,$slash)
                    if (-not $state.tests.ContainsKey($topLevelName) -or $state.tests[$topLevelName].terminalLine) {
                        throw 'Slash test run lacks a live observed top-level identity.'
                    }
                }
                $state.tests.Add($test,[ordered]@{
                    Package=$event.Package; Test=$test; runLine=$lineNumber; terminalLine=0; terminal=''; phase='running'
                    pauseCount=0; contCount=0; output=[Collections.Generic.List[object]]::new()
                    topLevelName=$topLevelName; activeSlashCount=0
                })
                if ($null -ne $topLevelName) { $state.tests[$topLevelName].activeSlashCount++ }
                continue
            }
            if (-not $state.tests.ContainsKey($test)) { throw 'Test event/terminal without unique run.' }
            $identity=$state.tests[$test]
            if ($identity.terminalLine) { throw 'Duplicate terminal or event after test completion.' }
            if ($event.Action -ceq 'pause') {
                if ($identity.phase -cne 'running') { throw 'Invalid/duplicate pause.' }
                $identity.phase='paused'; $identity.pauseCount++; continue
            }
            if ($event.Action -ceq 'cont') {
                if ($identity.phase -cne 'paused') { throw 'Continuation without pause.' }
                $identity.phase='running'; $identity.contCount++; continue
            }
        } else {
            $identity=$state
            if ($event.Action -cin @('run','pause','cont')) { throw 'Test action without Test identity.' }
        }
        if ($event.Action -ceq 'output') {
            if (-not $event.Contains('Output') -or $event.Output -isnot [string] -or $event.Output.Length -eq 0) { throw 'Output action lacks nonempty string Output.' }
            $outputType=if ($event.Contains('OutputType')) { $event.OutputType } else { '' }
            if ($outputType -isnot [string] -or $outputType -cnotin @('','frame','error','error-continue')) { throw 'Unknown OutputType.' }
            $identity.output.Add([ordered]@{line=$lineNumber; text=$event.Output; outputType=$outputType}); continue
        }
        if ($event.Action -cnotin @('pass','skip')) { throw 'Unsupported lifecycle transition.' }
        if ($null -ne $test) {
            if ($identity.phase -cne 'running') { throw 'Test terminal while paused.' }
            if ($identity.activeSlashCount -gt 0) { throw 'Top-level test completes before observed slash test terminal.' }
            if ($event.Action -ceq 'skip' -and -not @($identity.output | Where-Object { $_.outputType -cne 'frame' -and -not [string]::IsNullOrWhiteSpace($_.text) }).Count) {
                throw 'Test skip lacks captured non-framing output/reason.'
            }
        } elseif ($state.HasTests) {
            if ($event.Action -cne 'pass' -or -not $state.tests.Count) { throw 'Test-bearing package needs package pass and at least one observed run.' }
            foreach ($child in $state.tests.Values) { if (-not $child.terminalLine) { throw 'Package completes before test terminal.' } }
        } else {
            $actual=[string]::Concat([string[]]@($state.output | ForEach-Object { $_.text }))
            $marker="?   `t$($state.Package)`t[no test files]`n"
            if ($event.Action -cne 'skip' -or $state.tests.Count -or $actual -cne $marker) { throw 'Wrong no-test terminal or exact standard [no test files] output.' }
        }
        $identity.terminalLine=$lineNumber; $identity.terminal=$event.Action
        # Each full slash identity releases exactly one slot after its unique terminal.
        if ($null -ne $test -and $null -ne $identity.topLevelName) { $state.tests[$identity.topLevelName].activeSlashCount-- }
    }
    $packageRows=[Collections.Generic.List[object]]::new(); $testRows=[Collections.Generic.List[object]]::new()
    $packageSkips=[Collections.Generic.List[object]]::new(); $testSkips=[Collections.Generic.List[object]]::new()
    $packagePass=0; $testPass=0
    foreach ($package in $context.expected.packages) {
        $state=$states[$package.Package]
        if (-not $state.startLine -or -not $state.terminalLine) { throw "Missing/unterminated expected package: $($package.Package)" }
        $names=[string[]]@($state.tests.Keys); [Array]::Sort($names,[StringComparer]::Ordinal)
        foreach ($name in $names) {
            $identity=$state.tests[$name]
            if (-not $identity.terminalLine) { throw "Unterminated full test identity: $($identity.Package) / $name" }
            $row=[ordered]@{Package=$identity.Package; Test=$identity.Test; runLine=$identity.runLine; terminalLine=$identity.terminalLine; terminal=$identity.terminal; pauseCount=$identity.pauseCount; contCount=$identity.contCount; output=$identity.output.ToArray()}
            $testRows.Add($row)
            if ($identity.terminal -ceq 'skip') { $testSkips.Add($row) } else { $testPass++ }
        }
        $row=[ordered]@{Package=$state.Package; HasTests=$state.HasTests; startLine=$state.startLine; terminalLine=$state.terminalLine; terminal=$state.terminal; testCount=$state.tests.Count; output=$state.output.ToArray()}
        $packageRows.Add($row)
        if ($state.HasTests) { $packagePass++ } else { $packageSkips.Add($row) }
    }
    foreach ($entry in $context.pins.GetEnumerator()) {
        $after=ReadListingRevisionInput $entry.Value.absolutePath $entry.Value.sha256 $entry.Key $false
        SameFingerprint $entry.Value $after.fact
    }
    $summary=[ordered]@{schema=1; status='VERIFIED_CAPTURE_ONLY'; windowsAcceptance='NOT_WINDOWS_ACCEPTED'; ciEnabled=$false; roots=$Roots; target=$context.expected.target; expectedPackageIdentities=@($context.expected.packages | ForEach-Object { $_.Package }); eventLineCount=$lineNumber; failCount=0; packagePassCount=$packagePass; noTestPackageCount=$packageSkips.Count; testPassCount=$testPass; testSkipCount=$testSkips.Count; packages=$packageRows.ToArray(); tests=$testRows.ToArray(); packageSkips=$packageSkips.ToArray(); testSkips=$testSkips.ToArray(); inputPins=@($context.pins.GetEnumerator() | ForEach-Object { [ordered]@{name=$_.Key; fact=$_.Value} }); limitations='Only observed Test identities are checked. Slash identities are linked to their top-level identity only. JSONL does not encode intermediate parent edges. Raw, normalized and expected package/test-file projections agree only for the supplied bounded bytes. Authentic capture, real source/full-toolchain completeness, environment and chronology require independent Windows evidence. Unsupported event families fail closed.'}
    if ($SummaryPath) { NewJson $SummaryPath $summary }
    return $summary
}
