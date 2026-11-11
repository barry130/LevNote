$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
$coalescingStarted = [DateTime]::UtcNow
& .\scripts\go.ps1 -GoArguments @('test','./core/text','./ui/editor','-run','^(TestCoalescing|TestGrowth|TestCompositionIsOneTransactionAndCancelKeepsRedo$|TestNativeInputUndoAndSurrogateBoundaries$|TestCompositionAndReadOnly$|TestIMEThroughNativeTransportDoesNotDoubleInsert$|TestReleaseCompositionAndStyleDoNotLoseHistoryOrStealFocus$|TestIndentMappingDirectionEndpointAndGroupOwnership$|TestSelectionUndoAndIndentCommands$|TestReplaceReleasesDeadPieceTailAfterHistoryReset$)','-count=1')
$coalescingGoExit = $LASTEXITCODE
[ordered]@{StartedUtc=$coalescingStarted.ToString('O');FinishedUtc=[DateTime]::UtcNow.ToString('O');GoExitCode=$coalescingGoExit} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot '01-focused.go-result.json') -Encoding utf8
exit $coalescingGoExit
