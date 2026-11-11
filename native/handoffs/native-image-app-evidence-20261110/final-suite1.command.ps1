Set-Location -LiteralPath 'F:\AIproject\LevNote'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test', './app', './modules/...', './core/plugins', './core/session', './core/document', './formats/raster', './ui/editor', '-count=1')
exit $LASTEXITCODE
