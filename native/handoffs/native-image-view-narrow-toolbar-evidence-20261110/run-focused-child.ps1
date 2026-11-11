$ErrorActionPreference = 'Stop'

& (Join-Path $PSScriptRoot '..\..\scripts\go.ps1') -GoArguments @(
    'test',
    './modules/imageview',
    '-run',
    'TestToolbarControlsRemainReachableAtNarrowWidths',
    '-count=1'
)
exit $LASTEXITCODE
