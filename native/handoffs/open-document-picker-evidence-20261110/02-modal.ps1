$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
$pickerStarted = [DateTime]::UtcNow
& .\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestOpenDocuments(Modal|SinglePointer)','-count=1','-v')
$pickerExit = $LASTEXITCODE
[ordered]@{StartedUtc=$pickerStarted.ToString('O');FinishedUtc=[DateTime]::UtcNow.ToString('O');GoExitCode=$pickerExit} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot '02-modal.go-result.json') -Encoding utf8
exit $pickerExit

