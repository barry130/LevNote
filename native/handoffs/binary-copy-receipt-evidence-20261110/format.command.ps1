Set-Location -LiteralPath 'F:\AIproject\LevNote'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('run', 'cmd/gofmt', '-w', 'core/document/binary.go', 'core/document/binary_copy.go', 'core/document/binary_test.go')
