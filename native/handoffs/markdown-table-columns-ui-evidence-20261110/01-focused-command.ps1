& 'native/scripts/go.ps1' -GoArguments @('test','./modules/markdownview','./app','-run','^(TestTableColumns|TestMarkdownColumns)','-count=1','-v')
