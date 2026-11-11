$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
$coalescingStarted = [DateTime]::UtcNow
& .\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','./modules/graphview','./app','-run','^(TestGrowth|TestNativeDoubleClickEntryAndCompositionFlushBarriers$|TestForeignGroupRetainsDraftAndRetryDoesNotStealTransaction$|TestNativeCompositionImmediatelyFlushesBeforeSaveSelectionAndExport$|TestGroupConflictRetainsNativeDraftAcrossRefreshSelectionUndoAndRelease$|TestSaveFlushesPendingNativeIMEComposition$|TestRealMarkdownPendingIMECompositionAcrossSaveModeAndTab$|TestRealGraphPendingIMECompositionAcrossSaveModeAndTab$|TestRealDrawioPendingIMECompositionAcrossSaveModeAndTab$|TestFindKeyboardCompositionAndSourceCompositionBarriers$|TestSourceNavigationCompositionBarrierSourceAndRealMarkdown$|TestTabKeyboardSyntheticRoutingAndSourceComposition$|TestTabKeyboardSyntheticRoutingCommitsRichComposition$|TestReopenClosedSyntheticSourceAndRichComposition$)','-count=1')
$coalescingGoExit = $LASTEXITCODE
[ordered]@{StartedUtc=$coalescingStarted.ToString('O');FinishedUtc=[DateTime]::UtcNow.ToString('O');GoExitCode=$coalescingGoExit} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot '02-lifecycle-seams.go-result.json') -Encoding utf8
exit $coalescingGoExit
