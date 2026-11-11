Set-Location -LiteralPath 'F:\AIproject\LevNote'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('run', 'cmd/gofmt', '-w', 'app/autosave.go', 'app/autosave_test.go', 'app/app.go', 'app/state.go', 'app/view.go', 'app/workspace.go', 'app/export_worker.go', 'app/binary_actions.go', 'app/outline.go', 'app/plugins.go', 'app/source_navigation.go')
exit $LASTEXITCODE
