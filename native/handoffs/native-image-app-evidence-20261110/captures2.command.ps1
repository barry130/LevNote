Set-Location -LiteralPath 'F:\AIproject\LevNote'
$env:LEVNOTE_IMAGE_CAPTURE_DIR = 'F:\AIproject\LevNote\native\handoffs\native-image-app-evidence-20261110'
& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @('test', './app', '-run', '^TestNativeImageCaptures$', '-count=1', '-v')
exit $LASTEXITCODE
