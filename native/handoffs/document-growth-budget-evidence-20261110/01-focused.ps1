$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
& .\scripts\go.ps1 -GoArguments @('test', './core/text', './core/document', './core/search', './ui/editor', './modules/markdownview', './modules/graphview', './app', '-run', '^TestGrowth', '-count=1')
exit $LASTEXITCODE
