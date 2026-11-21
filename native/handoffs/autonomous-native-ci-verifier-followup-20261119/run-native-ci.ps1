#requires -Version 7.0
param(
    [Parameter(Mandatory=$true)][string]$InputContractPath,
    [Parameter(Mandatory=$true)][ValidatePattern('^[0-9a-f]{64}$')][string]$InputContractSHA256,
    [switch]$VerifyCapturedOnly,
    [switch]$ExecuteReviewed
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
# This revision intentionally contains NO Go/process/product execution lane.
if ($ExecuteReviewed) { throw 'CI execution remains DISABLED; this revision cannot test/vet/build/package/launch or enable CI.' }
if (-not $VerifyCapturedOnly) { throw 'INACTIVE DRAFT; a separately reviewed captured-verification contract and -VerifyCapturedOnly are required.' }
if (-not $IsWindows -or [IntPtr]::Size -ne 8 -or $PSVersionTable.PSVersion.Major -ne 7 -or
    [Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString() -cne 'X64' -or
    [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -cne 'X64') { throw 'Reviewed native Windows x64 PowerShell7 required; unsupported runtime is FAILED, not skip.' }
function InlinePlainFile([string]$Path) {
    if ($Path -notmatch '^[A-Za-z]:[\\/]' -or $Path -match '[\x00-\x1F\x7F]' -or $Path.Substring(2).Contains(':')) { throw 'Bootstrap path must be an absolute local drive path.' }
    $raw=$Path.Substring(3).Replace('\','/')
    if (-not $raw -or $raw -match '//|(^|/)\.{1,2}(/|$)|[*?"<>|]' -or $raw.EndsWith('/') -or
        $raw -match '(^|/)[^/]*[. ](/|$)|(^|/)(?i:CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³])(?:\.|/|$)') { throw 'Noncanonical bootstrap file path.' }
    $full=[IO.Path]::GetFullPath($Path); $current=$full; $first=$true
    while ($current) {
        $item=Get-Item -LiteralPath $current -Force -ErrorAction Stop
        if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -or ($first -and $item.PSIsContainer) -or (-not $first -and -not $item.PSIsContainer)) { throw 'Bootstrap wrong type/reparse chain.' }
        $first=$false; $parent=[IO.Path]::GetDirectoryName($current)
        if (-not $parent -or $parent -ceq $current) { break }; $current=$parent
    }
    return $full
}
function InlinePin([string]$Path,[string]$Hash) {
    if ($Hash -cnotmatch '^[0-9a-f]{64}$') { throw 'Bootstrap SHA256 required.' }
    $full=InlinePlainFile $Path
    if ((Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToLowerInvariant() -cne $Hash) { throw 'Bootstrap pin mismatch.' }
    return $full
}
function InlineUniqueJson([System.Text.Json.JsonElement]$Element) {
    if ($Element.ValueKind -eq [System.Text.Json.JsonValueKind]::Object) {
        $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
        foreach ($property in $Element.EnumerateObject()) {
            if (-not $seen.Add($property.Name)) { throw 'Duplicate/aliased contract JSON property.' }
            InlineUniqueJson $property.Value
        }
    } elseif ($Element.ValueKind -eq [System.Text.Json.JsonValueKind]::Array) {
        foreach ($member in $Element.EnumerateArray()) { InlineUniqueJson $member }
    }
}
$contractFull=InlinePin $InputContractPath $InputContractSHA256
$text=[Text.UTF8Encoding]::new($false,$true).GetString([IO.File]::ReadAllBytes($contractFull))
$document=[System.Text.Json.JsonDocument]::Parse($text)
try {
    InlineUniqueJson $document.RootElement
    if ($document.RootElement.ValueKind -ne [System.Text.Json.JsonValueKind]::Object) { throw 'Contract object required.' }
    $bootstrap=$document.RootElement.Clone()
} finally { $document.Dispose() }
foreach ($name in @('captureVerificationAuthorized','helpersReviewed','sourceFrozen')) {
    if ($bootstrap.GetProperty($name).ValueKind -ne [System.Text.Json.JsonValueKind]::True) { throw "False/missing root-reviewed $name." }
}
if ($bootstrap.GetProperty('schema').GetInt32() -ne 2 -or $bootstrap.GetProperty('purpose').GetString() -cne 'offline-captured-verifier-only') { throw 'Contract schema/purpose differs.' }
$helperNames=@('ci-lib.ps1','markdown-targets.ps1','test-json-reducer.ps1','package-verifier.ps1')
foreach ($name in $helperNames) { $null=InlinePin (Join-Path $PSScriptRoot $name) ($bootstrap.GetProperty('helpers').GetProperty($name).GetString()) }
$null=InlinePin $PSCommandPath ($bootstrap.GetProperty('runnerSHA256').GetString())
$null=InlinePin ($bootstrap.GetProperty('rootReview').GetProperty('path').GetString()) ($bootstrap.GetProperty('rootReview').GetProperty('sha256').GetString())
$runtimePath=[Diagnostics.Process]::GetCurrentProcess().MainModule.FileName
$null=InlinePin $runtimePath ($bootstrap.GetProperty('runtime').GetProperty('sha256').GetString())
if ($runtimePath -cne $bootstrap.GetProperty('runtime').GetProperty('path').GetString() -or
    $PSVersionTable.PSVersion.ToString() -cne $bootstrap.GetProperty('runtime').GetProperty('version').GetString()) { throw 'Actual PowerShell path/version differs from reviewed runtime.' }
$null=InlinePin $contractFull $InputContractSHA256
# Dot-source only pinned new-revision helpers, after bootstrap checks.
. (Join-Path $PSScriptRoot 'ci-lib.ps1')
. (Join-Path $PSScriptRoot 'markdown-targets.ps1')
. (Join-Path $PSScriptRoot 'test-json-reducer.ps1')
. (Join-Path $PSScriptRoot 'package-verifier.ps1')
$contract=StrictJson $text
Keys $contract @('schema','purpose','captureVerificationAuthorized','helpersReviewed','sourceFrozen','roots','helpers','runnerSHA256','rootReview','runtime','inputs','evidenceRoot') @('packageCapture')
Keys $contract.helpers $helperNames
Keys $contract.rootReview @('path','sha256'); Keys $contract.runtime @('path','sha256','version')
RequireArray $contract.roots 'contract roots'; AssertProductRoots $contract.roots
# Root/context/pins are validated BEFORE FreshDirectory or ANY evidence write.
$context=ReadExpectedTestContext $contract.inputs $contract.roots
$contractFact=Pin $contractFull $InputContractSHA256
$rootReview=Pin $contract.rootReview.path $contract.rootReview.sha256
$runtimeFact=Pin $runtimePath $contract.runtime.sha256
$helperFacts=@(foreach ($name in $helperNames) { Pin (Join-Path $PSScriptRoot $name) $contract.helpers[$name] })
$runnerFact=Pin $PSCommandPath $contract.runnerSHA256
$evidence=FreshDirectory $contract.evidenceRoot
$result=[ordered]@{
    schema=1; status='REJECTED_CAPTURE'; scope='Offline captured verification only'
    windowsAcceptance='NOT_WINDOWS_ACCEPTED'; ciEnabled=$false
    startedAtUtc=[DateTime]::UtcNow.ToString('o'); errors=@(); rawJsonlRetained=$false
    contract=$contractFact; rootReview=$rootReview; runtime=$runtimeFact
    runner=$runnerFact; helpers=$helperFacts
    inputPinsBefore=@($context.pins.GetEnumerator() | ForEach-Object { [ordered]@{name=$_.Key; fact=$_.Value} })
}
try {
    $rawDestination=Join-Path $evidence 'stdout.raw.jsonl'
    $inputStream=[IO.File]::OpenRead($context.pins['rawJsonl'].absolutePath)
    $outputStream=[IO.File]::Open($rawDestination,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::Read)
    try { $inputStream.CopyTo($outputStream); $outputStream.Flush($true) } finally { $outputStream.Dispose(); $inputStream.Dispose() }
    $null=Pin $rawDestination $context.pins['rawJsonl'].sha256
    $result.rawJsonlRetained=$true
    $inputs=@{}
    foreach ($name in $contract.inputs.Keys) { $inputs[$name]=@{path=$contract.inputs[$name].path; sha256=$contract.inputs[$name].sha256} }
    $inputs.rawJsonl.path=$rawDestination
    if ($contract.Contains('packageCapture')) {
        Keys $contract.packageCapture @('nativeRoot','bundle','packageEvidence','version')
        $capture=$contract.packageCapture
        if (-not (LocalPath $capture.nativeRoot).Equals((LocalPath (Join-Path $context.sourceRepositoryRoot 'native')),[StringComparison]::OrdinalIgnoreCase)) { throw 'Package source root differs from frozen source inventory repository.' }
        $packageFacts=CheckFivePEPackage $capture.nativeRoot $capture.bundle $capture.packageEvidence $capture.version
        NewJson (Join-Path $evidence 'five-pe-static.json') $packageFacts
    }
    $summary=ReduceTestJson $inputs $contract.roots (Join-Path $evidence 'test-summary.json')
    foreach ($pin in $context.pins.Values) { SameFingerprint $pin (Fingerprint $pin.absolutePath) }
    foreach ($pin in @($contractFact,$rootReview,$runtimeFact,$runnerFact)+$helperFacts) { SameFingerprint $pin (Fingerprint $pin.absolutePath) }
    $result.status='VERIFIED_CAPTURE_ONLY'; $result.packagePassCount=$summary.packagePassCount; $result.testSkipCount=$summary.testSkipCount
} catch { $result.errors=@([ordered]@{detail=$_.Exception.ToString(); scriptStackTrace=[string]$_.ScriptStackTrace}) }
finally {
    $result.finishedAtUtc=[DateTime]::UtcNow.ToString('o')
    NewJson (Join-Path $evidence 'result.json') $result
}
$result | ConvertTo-Json -Depth 8
if ($result.status -cne 'VERIFIED_CAPTURE_ONLY') { exit 1 }
exit 0
