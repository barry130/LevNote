$env:LEVNOTE_REOPEN_SCREENSHOT = 'F:\AIproject\LevNote\native\handoffs\reopen-closed-file-evidence-20261110\recent-menu-760x480.png'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test','./app','-run','^TestReopenClosed(FailedCloseNeverRecords|WindowCloseCancelsRunningImage|MenuGeometryAndEmptyState)$','-count=1','-v')
exit $LASTEXITCODE
