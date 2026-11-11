$ErrorActionPreference = 'Stop'

$repo = 'F:\AIproject\LevNote'
$examplesDir = Join-Path $repo 'native\examples\plugins'
$portableReadme = Join-Path $repo 'native\portable\README.md'
$acceptedBundle = Join-Path $repo 'native\bin\levnote-native-worker-20261110.19'

$checks = [System.Collections.Generic.List[object]]::new()
function Add-Check([string]$name, [bool]$pass, [string]$detail) {
    $checks.Add([pscustomobject][ordered]@{
        Name = $name
        Pass = $pass
        Detail = $detail
    })
}

$utf8 = [System.Text.UTF8Encoding]::new($false, $true)
$exampleFiles = @(Get-ChildItem -LiteralPath $examplesDir -Filter '*.json' -File | Sort-Object Name)
$exampleManifests = @()
foreach ($file in $exampleFiles) {
    $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
    $text = $utf8.GetString($bytes)
    $exampleManifests += [pscustomobject]@{
        File = $file.Name
        Manifest = $text | ConvertFrom-Json
    }
}
Add-Check 'all example JSON files are strict UTF-8 and structurally parse' $true "files=$($exampleFiles.Count)"

$newRow = @($exampleManifests | Where-Object File -eq 'diagram-notes.json')
$new = if ($newRow.Count -eq 1) { $newRow[0].Manifest } else { $null }
$newFields = if ($null -ne $new) { @($new.PSObject.Properties.Name | Sort-Object) } else { @() }
$expectedFields = @('apiVersion', 'extensions', 'id', 'name', 'nativeRoute', 'schema', 'version' | Sort-Object)
$newRoutePass = $null -ne $new -and
    [int]$new.schema -eq 3 -and
    [int]$new.apiVersion -eq 1 -and
    $new.id -ceq 'diagram-notes' -and
    $new.version -ceq '1.0.0' -and
    @($new.extensions).Count -eq 1 -and
    $new.extensions[0] -ceq '.diagramnote' -and
    $new.nativeRoute -ceq 'builtin.drawio' -and
    (($newFields -join ',') -ceq ($expectedFields -join ','))
Add-Check 'diagram-notes example has the exact schema3/API1 data route' $newRoutePass 'id=diagram-notes, extension=.diagramnote, route=builtin.drawio'

$portableBytes = [System.IO.File]::ReadAllBytes($portableReadme)
$portableText = $utf8.GetString($portableBytes)
$versionPlaceholders = [regex]::Matches($portableText, '\{\{VERSION\}\}').Count
Add-Check 'portable README retains exactly one version placeholder' ($versionPlaceholders -eq 1) "placeholders=$versionPlaceholders"

$jsonBlocks = [regex]::Matches(
    $portableText,
    '```json\r?\n(.*?)\r?\n```',
    [Text.RegularExpressions.RegexOptions]::Singleline
)
$inlineManifests = @(
    foreach ($match in $jsonBlocks) {
        $match.Groups[1].Value | ConvertFrom-Json
    }
)
$inlineRoutes = @($inlineManifests | ForEach-Object nativeRoute)
$expectedRoutes = @('builtin.markdown', 'builtin.mindmap', 'builtin.drawio')
Add-Check 'portable README contains three structurally valid inline schema3 examples' (
    $inlineManifests.Count -eq 3 -and
    [int]$inlineManifests[0].schema -eq 3 -and
    [int]$inlineManifests[1].schema -eq 3 -and
    [int]$inlineManifests[2].schema -eq 3 -and
    ($inlineRoutes -join ',') -ceq ($expectedRoutes -join ',')
) "blocks=$($inlineManifests.Count), routes=$($inlineRoutes -join ',')"

$exampleReadme = Join-Path $examplesDir 'README.md'
$exampleReadmeText = $utf8.GetString([System.IO.File]::ReadAllBytes($exampleReadme))
$exampleLinks = @(
    [regex]::Matches($exampleReadmeText, '\[[^\]]+\]\(([^)]+)\)') |
        ForEach-Object { $_.Groups[1].Value }
)
$exampleMissing = @(
    $exampleLinks |
        Where-Object { -not (Test-Path -LiteralPath (Join-Path $examplesDir $_)) }
)
Add-Check 'examples README relative links resolve' (
    $exampleLinks.Count -eq 3 -and $exampleMissing.Count -eq 0
) "links=$($exampleLinks.Count), missing=$($exampleMissing.Count)"

$portableLinks = @(
    [regex]::Matches($portableText, '\[[^\]]+\]\(([^)]+)\)') |
        ForEach-Object { $_.Groups[1].Value }
)
$expectedPortableLinks = @(
    'release-manifest.json',
    'LICENSE',
    'THIRD_PARTY_NOTICES.md',
    'licenses/',
    'attribution/'
)
$portableMissing = @(
    $portableLinks |
        Where-Object {
            $relative = $_.TrimEnd('/')
            -not (Test-Path -LiteralPath (Join-Path $acceptedBundle $relative))
        }
)
Add-Check 'portable README retains the five existing local notice/package links' (
    ($portableLinks -join ',') -ceq ($expectedPortableLinks -join ',') -and
    $portableMissing.Count -eq 0
) "links=$($portableLinks.Count), missingInAcceptedBundle=$($portableMissing.Count)"

$acceptedMarker = $portableText.Contains('当前已接受的`.19`随包只支持schema3的 `builtin.markdown` 和') -and
    $portableText.Contains('`.diagramnote` 的 `builtin.drawio` 路由需要后续构建。')
Add-Check 'portable README explicitly distinguishes .19 from the later draw.io route' $acceptedMarker 'accepted=.19, drawio=pending later build'

$pass = @($checks | Where-Object { -not $_.Pass }).Count -eq 0
[pscustomobject][ordered]@{
    Schema = 1
    Status = if ($pass) { 'PASS' } else { 'FAIL' }
    Checks = $checks
    Scope = 'Structural JSON parse, UTF-8 decode, version-marker count and relative authored-link checks only. This is not production manifest admission, Go tests, a build, a worker/host launch or a release claim.'
} | ConvertTo-Json -Depth 6

if (-not $pass) {
    exit 1
}
