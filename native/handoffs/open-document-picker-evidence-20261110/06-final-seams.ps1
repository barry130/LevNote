$env:LEVNOTE_OPEN_DOCUMENTS_CAPTURE_DIR = 'F:\AIproject\LevNote\native\handoffs\open-document-picker-evidence-20261110\capture'
$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
$pickerStarted = [DateTime]::UtcNow
& .\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestOpenDocuments(ModalQueryComposition|MetadataUnicodeBoundsAndReuse|HundredVirtualRowsAndQueryNavigation|MinimumWindowCapture)$','-count=1','-v')
$pickerExit = $LASTEXITCODE
[ordered]@{StartedUtc=$pickerStarted.ToString('O');FinishedUtc=[DateTime]::UtcNow.ToString('O');GoExitCode=$pickerExit} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot '06-final-seams.go-result.json') -Encoding utf8
exit $pickerExit




