& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test','./app','-run','^TestReopenClosedSuccessfulShortcutRepeatAndStaleMenuID$','-count=1','-v')
exit $LASTEXITCODE
