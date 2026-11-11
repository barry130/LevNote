Remove-Item Env:LEVNOTE_IMAGE_REFERENCE_CAPTURE_DIR -ErrorAction SilentlyContinue
& 'native/scripts/go.ps1' -GoArguments @('test','./modules/markdownview','./app','-count=1')
exit $LASTEXITCODE
