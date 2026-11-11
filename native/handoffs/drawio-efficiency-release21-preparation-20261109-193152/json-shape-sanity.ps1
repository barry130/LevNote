$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')

function Read-ZeroWarnings([string]$Path, [string]$PropertyName) {
    $state = Get-JsonArrayProperty $Path $PropertyName
    $items = [object[]]@($state.Elements)
    $json = ConvertTo-Json -InputObject $items -Depth 6 -Compress
    if ($items.Count -ne 0 -or $json -cne '[]') {
        throw "Expected zero warnings serialized as []; found $($items.Count): $json"
    }
    return [pscustomobject][ordered]@{
        Kind = $state.Kind
        Count = $items.Count
        Json = $json
    }
}

function Test-Rejected([string]$Path, [string]$PropertyName) {
    try {
        Read-ZeroWarnings $Path $PropertyName | Out-Null
        return $false
    } catch {
        return $true
    }
}

$temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) `
    ('levnote-release21-warning-shape-' + [Guid]::NewGuid().ToString('N'))
$checks = [Collections.Generic.List[object]]::new()
try {
    New-Item -ItemType Directory -Path $temporaryRoot -ErrorAction Stop | Out-Null
    $accepted = [ordered]@{
        'raw-null' = @('Warnings', '{"Warnings":null}', 'Null')
        'raw-empty-array' = @('Warnings', '{"Warnings":[]}', 'Array')
        'normalized-array' = @('Warnings', '{"Warnings":[]}', 'Array')
    }
    foreach ($name in $accepted.Keys) {
        $propertyName = $accepted[$name][0]
        $pathAsText = Join-Path $temporaryRoot ($name + '.json')
        [IO.File]::WriteAllText($pathAsText, $accepted[$name][1], [Text.UTF8Encoding]::new($false))
        $shape = Read-ZeroWarnings $pathAsText $propertyName
        $checks.Add([ordered]@{
            Name = $name
            Pass = $shape.Kind -ceq $accepted[$name][2] -and
                $shape.Count -eq 0 -and $shape.Json -ceq '[]'
            Kind = $shape.Kind
            Count = $shape.Count
            Json = $shape.Json
        })
    }

    $rejected = [ordered]@{
        'nonempty-array' = @('Warnings', '{"Warnings":[1]}')
        'scalar-number' = @('Warnings', '{"Warnings":0}')
        'scalar-string' = @('Warnings', '{"Warnings":"none"}')
        'object' = @('Warnings', '{"Warnings":{}}')
        'null-element' = @('Warnings', '{"Warnings":[null]}')
        'missing-property' = @('Warnings', '{}')
        'duplicate-property' = @('Warnings', '{"Warnings":[],"Warnings":[]}')
        'case-mismatch-lower' = @('Warnings', '{"warnings":[]}')
        'case-mismatch-upper' = @('warnings', '{"Warnings":[]}')
        'normalized-nonempty-array' = @('Warnings', '{"Warnings":[1]}')
        'normalized-scalar-number' = @('Warnings', '{"Warnings":0}')
        'normalized-scalar-string' = @('Warnings', '{"Warnings":"none"}')
        'normalized-object' = @('Warnings', '{"Warnings":{}}')
        'normalized-null-element' = @('Warnings', '{"Warnings":[null]}')
        'wrong-root' = @('Warnings', '[]')
        'malformed-json' = @('Warnings', '{"Warnings":[}')
    }
    foreach ($name in $rejected.Keys) {
        $pathAsText = Join-Path $temporaryRoot ($name + '.json')
        [IO.File]::WriteAllText($pathAsText, $rejected[$name][1], [Text.UTF8Encoding]::new($false))
        $checks.Add([ordered]@{
            Name = $name
            Pass = Test-Rejected $pathAsText $rejected[$name][0]
        })
    }
} finally {
    if (Test-Path -LiteralPath $temporaryRoot) {
        foreach ($file in @(Get-ChildItem -LiteralPath $temporaryRoot -File)) {
            Remove-Item -LiteralPath $file.FullName -Force
        }
        Remove-Item -LiteralPath $temporaryRoot -Force
    }
}

$pass = @($checks | Where-Object { -not $_.Pass }).Count -eq 0
$result = [ordered]@{
    Schema = 1
    PowerShell = $PSVersionTable.PSVersion.ToString()
    AcceptedCases = $accepted.Count
    RejectedCases = $rejected.Count
    Pass = $pass
    Checks = $checks
}
$result | ConvertTo-Json -Depth 6
if (-not $pass) {
    exit 1
}
