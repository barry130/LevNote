& 'native/scripts/go.ps1' -GoArguments @('test','./app','-run','^TestMarkdownImageReferenceRealAdapterOpenPreservesSource/PNG$','-count=1','-v')
exit $LASTEXITCODE
