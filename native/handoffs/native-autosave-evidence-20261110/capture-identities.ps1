param([Parameter(Mandatory = $true)][ValidateSet('03-source-before','06-source-after','07-corrected-source-before','10-corrected-source-after')][string]$Name)
$ErrorActionPreference = 'Stop'
$repoRoot = 'F:\AIproject\LevNote'
$expected = @(Get-Content -LiteralPath (Join-Path $PSScriptRoot 'source-freeze.json') -Raw | ConvertFrom-Json)
$keyboardResult = Get-Content -LiteralPath (Join-Path $repoRoot 'native\handoffs\native-image-keyboard-result.md') -Raw
$matches = [regex]::Matches($keyboardResult, '\| `(?<path>native/[^`]+)` \| (?<bytes>\d+) \| `(?<sha256>[A-F0-9]{64})` \|')
if ($expected.Count -ne 11 -or $matches.Count -ne 4) { throw 'Expected exactly11 autosave and4 keyboard source records' }
foreach ($match in $matches) {
    $expected += [pscustomobject]@{path=$match.Groups['path'].Value;bytes=[long]$match.Groups['bytes'].Value;sha256=$match.Groups['sha256'].Value}
}
$rows = foreach ($entry in $expected) {
    $path = Join-Path $repoRoot $entry.path
    $item = Get-Item -LiteralPath $path
    $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
    [ordered]@{path=$entry.path;bytes=$item.Length;sha256=$hash;expectedBytes=$entry.bytes;expectedSHA256=$entry.sha256;matches=($item.Length -eq $entry.bytes -and $hash -ceq $entry.sha256)}
}
$target = Join-Path $PSScriptRoot "$Name.json"
if (Test-Path -LiteralPath $target) { throw "Evidence exists: $target" }
$result = [ordered]@{recordedUtc=[DateTimeOffset]::UtcNow.ToString('o');sourceCount=$rows.Count;allMatch=(@($rows | Where-Object { -not $_.matches }).Count -eq 0);sources=@($rows)}
[IO.File]::WriteAllText($target,($result | ConvertTo-Json -Depth 5)+"`r`n",[Text.UTF8Encoding]::new($false))
if (-not $result.allMatch) { throw "Frozen source mismatch; preserved $target" }
Write-Output "Verified $($rows.Count)/15 frozen source identities: $target"
