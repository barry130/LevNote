Set-Location -LiteralPath 'F:\AIproject\LevNote'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test', './app', '-run', '^TestNativeImage', '-count=1', '-v')
exit $LASTEXITCODE
