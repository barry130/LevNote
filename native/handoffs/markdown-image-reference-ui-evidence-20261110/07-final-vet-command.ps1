Remove-Item Env:LEVNOTE_IMAGE_REFERENCE_CAPTURE_DIR -ErrorAction SilentlyContinue
& 'native/scripts/go.ps1' -GoArguments @('vet','./modules/markdownview','./app')
exit $LASTEXITCODE
