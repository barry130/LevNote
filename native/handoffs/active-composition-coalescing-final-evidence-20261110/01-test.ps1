$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
$compositionStarted = [DateTime]::UtcNow
& .\scripts\go.ps1 -GoArguments @('test','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app','-count=1')
$compositionGoExit = $LASTEXITCODE
[ordered]@{StartedUtc=$compositionStarted.ToString('O');FinishedUtc=[DateTime]::UtcNow.ToString('O');GoExitCode=$compositionGoExit;ExitSource='scripts/go.ps1 exits with captured native go.exe LASTEXITCODE'} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot '01-test.go-result.json') -Encoding utf8
exit $compositionGoExit
