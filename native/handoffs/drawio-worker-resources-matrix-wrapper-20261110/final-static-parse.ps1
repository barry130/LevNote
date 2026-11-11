$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$wrapperRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$scriptPaths = @(
        (Join-Path $wrapperRoot 'run-matrix.ps1'),
        (Join-Path $wrapperRoot 'summarize-matrix.ps1')
)
$records = [Collections.Generic.List[object]]::new()
foreach ($script in $scriptPaths) {
        $tokens = $null
        $errors = $null
        [Management.Automation.Language.Parser]::ParseFile($script, [ref]$tokens, [ref]$errors) | Out-Null
    $records.Add([ordered]@{
            Path = $script
            Bytes = [Int64](Get-Item -LiteralPath $script).Length
            SHA256 = (Get-FileHash -LiteralPath $script -Algorithm SHA256).Hash.ToLowerInvariant()
            ParseErrorCount = @($errors).Count
            ParseErrors = @($errors | ForEach-Object {
                [ordered]@{
                    Message = $_.Message
                    StartLine = $_.Extent.StartLineNumber
                    StartColumn = $_.Extent.StartColumnNumber
                }
            })
    })
}
$result = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    PowerShell = $PSVersionTable.PSVersion.ToString()
    ScriptCount = $records.Count
    ParseErrorCount = [int](($records | ForEach-Object { $_.ParseErrorCount } | Measure-Object -Sum).Sum)
    Pass = @($records | Where-Object { $_.ParseErrorCount -ne 0 }).Count -eq 0
    Scope = 'Static parse only; no wrapper execution, tester launch, worker request, host request, compile, preflight, or product run.'
    Scripts = $records
}
[IO.File]::WriteAllText((Join-Path $wrapperRoot 'final-static-parse.json'), ($result | ConvertTo-Json -Depth 12), [Text.UTF8Encoding]::new($false))
$result | ConvertTo-Json -Depth 8
if (-not $result.Pass) { exit 1 }
exit 0
