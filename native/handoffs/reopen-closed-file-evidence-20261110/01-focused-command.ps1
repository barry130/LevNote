& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test','./app','-run','^TestReopenClosed','-count=1','-v')
exit $LASTEXITCODE
