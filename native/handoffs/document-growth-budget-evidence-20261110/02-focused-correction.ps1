$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath 'F:\AIproject\LevNote\native'
& .\scripts\go.ps1 -GoArguments @('test', './ui/editor', './app', '-run', '^TestGrowth', '-count=1')
exit $LASTEXITCODE
