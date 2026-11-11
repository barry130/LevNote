$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$wrapperRoot = Split-Path -Parent $PSScriptRoot
$resultPath = Join-Path $PSScriptRoot 'final-parse.json'
if (Test-Path -LiteralPath $resultPath) { throw 'Final parse evidence already exists.' }
$records = @(
    foreach ($name in @('run-matrix.ps1', 'summarize-matrix.ps1')) {
        $path = Join-Path $wrapperRoot $name
        $tokens = $null; $parseErrors = $null
        [Management.Automation.Language.Parser]::ParseFile($path, [ref]$tokens, [ref]$parseErrors) | Out-Null
        [ordered]@{
            Path = $path
            Bytes = (Get-Item -LiteralPath $path).Length
            SHA256 = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
            ParseErrors = @($parseErrors | ForEach-Object { $_.Message })
            ParseErrorCount = $parseErrors.Count
        }
    }
)
$result = [ordered]@{
    Schema = 1; CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    PowerShell = $PSVersionTable.PSVersion.ToString()
    Pass = @($records | Where-Object { $_.ParseErrorCount -ne 0 }).Count -eq 0
    Scripts = $records
    Scope = 'Static parse only. No tester/worker/host/Go test/build/preflight.'
}
[IO.File]::WriteAllText($resultPath, ($result | ConvertTo-Json -Depth 8))
$result | ConvertTo-Json -Depth 8
if (-not $result.Pass) { exit 1 }
exit 0
