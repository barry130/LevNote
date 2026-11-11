$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
Remove-Item Env:LEVNOTE_OPEN_DOCUMENTS_CAPTURE_DIR -ErrorAction SilentlyContinue
$pickerStarted = [DateTime]::UtcNow
& .\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestOpenDocuments','-count=1','-v')
$pickerExit = $LASTEXITCODE
[ordered]@{StartedUtc=$pickerStarted.ToString('O');FinishedUtc=[DateTime]::UtcNow.ToString('O');GoExitCode=$pickerExit} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot '01-focused-union.go-result.json') -Encoding utf8
exit $pickerExit
