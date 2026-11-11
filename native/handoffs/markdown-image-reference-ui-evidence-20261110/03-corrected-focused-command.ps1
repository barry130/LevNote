& 'native/scripts/go.ps1' -GoArguments @('test','./modules/markdownview','./app','-run','^(TestImageReference|TestMarkdownImageReference)','-count=1','-v')
exit $LASTEXITCODE
