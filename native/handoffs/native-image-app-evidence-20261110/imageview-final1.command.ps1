Set-Location -LiteralPath 'F:\AIproject\LevNote'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test', './modules/imageview', '-count=1')
exit $LASTEXITCODE
