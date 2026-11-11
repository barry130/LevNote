Set-Location -LiteralPath 'F:\AIproject\LevNote'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test', './core/document', '-run', '^TestBinaryCopy', '-count=1', '-v')
