$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')

$records = @(
    foreach ($script in Get-ChildItem -LiteralPath $evidenceRoot -File |
            Where-Object { $_.Extension -eq '.ps1' } |
            Sort-Object Name) {
        $tokens = $null
        $errors = $null
        [Management.Automation.Language.Parser]::ParseFile(
            $script.FullName,
            [ref]$tokens,
            [ref]$errors
        ) | Out-Null
        [ordered]@{
            Path = $script.FullName
            Bytes = [Int64]$script.Length
            SHA256 = (Get-FileHash -LiteralPath $script.FullName -Algorithm SHA256).Hash
            ParseErrorCount = @($errors).Count
            ParseErrors = @($errors | ForEach-Object {
                [ordered]@{
                    Message = $_.Message
                    Extent = $_.Extent.Text
                    StartLine = $_.Extent.StartLineNumber
                    StartColumn = $_.Extent.StartColumnNumber
                }
            })
        }
    }
)
$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    PowerShell = $PSVersionTable.PSVersion.ToString()
    ScriptCount = $records.Count
    ParseErrorCount = [int](($records | Measure-Object ParseErrorCount -Sum).Sum)
    Pass = @($records | Where-Object { $_.ParseErrorCount -ne 0 }).Count -eq 0
    Scripts = $records
}
Write-JsonFile (Join-Path $evidenceRoot 'static-parse.json') $result 16
$result | ConvertTo-Json -Depth 8
if (-not $result.Pass) {
    exit 1
}
